package logger

import "net/http"

// responseData — захваченные данные HTTP-ответа для логирования.
type responseData struct {
	status  int         // код статуса HTTP-ответа
	size    int         // размер тела ответа
	headers http.Header // заголовки ответа
}

// loggingResponseWriter — обёртка над http.ResponseWriter для логирования.
// Захватывает код статуса, размер тела и заголовки ответа.
type loggingResponseWriter struct {
	http.ResponseWriter // встраиваем оригинальный http.ResponseWriter
	responseData        *responseData
}

// Write — запись данных для логирования.
// Перехватывает записываемые данные и сохраняет их размер.
func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	// записываем ответ, используя оригинальный http.ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size // захватываем размер
	r.responseData.headers = r.Header()
	return size, err
}

// WriteHeader — запись заголовка для логирования.
// Перехватывает код статуса HTTP-ответа.
func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	// записываем код статуса, используя оригинальный http.ResponseWriter
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode // захватываем код статуса
}
