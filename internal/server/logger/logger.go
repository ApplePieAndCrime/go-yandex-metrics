package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// LoggerInitialize создаёт логгер для режима разработки.
func LoggerInitialize() zap.SugaredLogger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return *zap.NewNop().Sugar()
	}

	defer logger.Sync()

	return *logger.Sugar()
}

type responseData struct {
	http.ResponseWriter
	status int
	size   int
}

func (r *responseData) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseData) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}

	n, err := r.ResponseWriter.Write(b)
	r.size += n
	return n, err
}

// WithLogging возвращает middleware, записывающий параметры HTTP-запроса и ответа.
func WithLogging(logger zap.SugaredLogger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		logFn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			respData := &responseData{
				ResponseWriter: w,
				status:         0,
				size:           0,
			}

			next.ServeHTTP(respData, r)

			if respData.status == 0 {
				respData.status = http.StatusOK
			}
			duration := time.Since(start)

			logger.Infoln(
				"uri", r.RequestURI,
				"method", r.Method,
				"status", respData.status,
				"size", respData.size,
				"duration", duration,
			)

		}
		return http.HandlerFunc(logFn)
	}

}
