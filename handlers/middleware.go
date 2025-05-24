package handlers

import (
	"net/http"
	"time"
)

// LoggingMiddleware registra informações sobre cada requisição HTTP
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Criar um ResponseWriter personalizado para capturar o status code
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Chamar o próximo handler
		next.ServeHTTP(rw, r)

		// Calcular a duração da requisição
		duration := time.Since(start)

		// Registrar a requisição
		LogRequest(r.Method, r.URL.Path, r.RemoteAddr, rw.statusCode, duration)
	})
}

// responseWriter é um wrapper para http.ResponseWriter que captura o status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captura o status code antes de escrevê-lo
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
