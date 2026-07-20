package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// Event описывает событие изменения одной или нескольких метрик.
type Event struct {
	Timestamp int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

// Observer получает события аудита и передаёт их выбранному получателю.
type Observer interface {
	Notify(context.Context, Event) error
}

// Publisher публикует события аудита подписанным наблюдателям.
type Publisher interface {
	Publish(context.Context, Event) error
}

// Manager рассылает события аудита зарегистрированным наблюдателям.
type Manager struct {
	observers []Observer
}

// NewManager создаёт менеджер и исключает из списка пустых наблюдателей.
func NewManager(observers ...Observer) *Manager {
	filtered := make([]Observer, 0, len(observers))
	for _, observer := range observers {
		if observer != nil {
			filtered = append(filtered, observer)
		}
	}

	return &Manager{observers: filtered}
}

// Publish отправляет событие всем наблюдателям и объединяет полученные ошибки.
func (m *Manager) Publish(ctx context.Context, event Event) error {
	errCh := make(chan error, len(m.observers))
	jobs := make(chan Observer, len(m.observers))
	var wg sync.WaitGroup

	workerCount := min(len(m.observers), 4)
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for observer := range jobs {
				errCh <- observer.Notify(ctx, event)
			}
		}()
	}

	for _, observer := range m.observers {
		jobs <- observer
	}
	close(jobs)

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// Close освобождает ресурсы наблюдателей, поддерживающих закрытие.
func (m *Manager) Close() error {
	var errs []error
	for _, observer := range m.observers {
		closer, ok := observer.(io.Closer)
		if !ok {
			continue
		}
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// FileObserver записывает события аудита в файл в формате JSON Lines.
type FileObserver struct {
	path   string
	file   *os.File
	closed bool
	mu     sync.Mutex
}

// NewFileObserver создаёт файловый наблюдатель или возвращает nil для пустого пути.
func NewFileObserver(path string) *FileObserver {
	if path == "" {
		return nil
	}

	return &FileObserver{path: path}
}

// Notify добавляет событие аудита в файл.
func (o *FileObserver) Notify(_ context.Context, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return errors.New("audit file observer is closed")
	}

	if o.file == nil {
		file, err := os.OpenFile(o.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("open audit file: %w", err)
		}
		o.file = file
	}

	if _, err := o.file.Write(append(payload, '\n')); err != nil {
		return fmt.Errorf("write audit file: %w", err)
	}

	return nil
}

// Close закрывает файл аудита, если он был открыт.
func (o *FileObserver) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.closed = true
	if o.file == nil {
		return nil
	}

	err := o.file.Close()
	o.file = nil
	return err
}

// HTTPClient задаёт минимальный интерфейс клиента для отправки событий аудита.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// HTTPObserver отправляет события аудита на HTTP-эндпоинт.
type HTTPObserver struct {
	url    string
	client HTTPClient
}

var auditRetryIntervals = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

type retryHTTPClient struct {
	client HTTPClient
}

func (c retryHTTPClient) Do(req *http.Request) (*http.Response, error) {
	var lastResponse *http.Response
	var lastErr error

	for attempt := 0; attempt <= len(auditRetryIntervals); attempt++ {
		attemptRequest := req.Clone(req.Context())
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			attemptRequest.Body = body
		}

		response, err := c.client.Do(attemptRequest)
		lastResponse, lastErr = response, err
		if !shouldRetryHTTP(response, err) || attempt == len(auditRetryIntervals) {
			return response, err
		}

		if response != nil && response.Body != nil {
			response.Body.Close()
		}

		timer := time.NewTimer(auditRetryIntervals[attempt])
		select {
		case <-req.Context().Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return nil, req.Context().Err()
		case <-timer.C:
		}
	}

	return lastResponse, lastErr
}

func shouldRetryHTTP(response *http.Response, err error) bool {
	if err != nil {
		return true
	}
	return response != nil && (response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError)
}

// NewHTTPObserver создаёт HTTP-наблюдатель или возвращает nil для пустого URL.
func NewHTTPObserver(url string, client HTTPClient) *HTTPObserver {
	if url == "" {
		return nil
	}

	if client == nil {
		client = http.DefaultClient
	}

	return &HTTPObserver{
		url:    url,
		client: retryHTTPClient{client: client},
	}
}

// Notify отправляет событие POST-запросом в формате JSON.
func (o *HTTPObserver) Notify(ctx context.Context, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create audit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("send audit request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("audit receiver returned status %d", resp.StatusCode)
	}

	return nil
}
