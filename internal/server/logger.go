package internal

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

func WithLogging(h http.HandlerFunc) http.HandlerFunc {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// эндпоинт /ping
		uri := r.RequestURI
		// метод запроса
		method := r.Method

		h.ServeHTTP(w, r)

		duration := time.Since(start)

		Sugar.Infoln(
			"uri", uri,
			"method", method,
			"duration", duration,
		)
	}
	return logFn
}
