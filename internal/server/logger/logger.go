package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

var Sugar zap.SugaredLogger

func LoggerInitialize() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	defer logger.Sync()

	Sugar = *logger.Sugar()
}

func WithLogging(next http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// эндпоинт /ping
		uri := r.RequestURI
		// метод запроса
		method := r.Method

		duration := time.Since(start)

		Sugar.Infoln(
			"uri", uri,
			"method", method,
			"duration", duration,
		)

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(logFn)
}
