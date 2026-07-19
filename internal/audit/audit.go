package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
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
	var errs []error

	for _, observer := range m.observers {
		if err := observer.Notify(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// FileObserver записывает события аудита в файл в формате JSON Lines.
type FileObserver struct {
	path string
	mu   sync.Mutex
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

	file, err := os.OpenFile(o.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open audit file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(append(payload, '\n')); err != nil {
		return fmt.Errorf("write audit file: %w", err)
	}

	return nil
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
		client: client,
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
