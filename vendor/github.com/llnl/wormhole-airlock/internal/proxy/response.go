package proxy

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

// responseWriter is a wrapper around http.ResponseWriter
// that tracks the status code.
type responseWriter struct {
	http.ResponseWriter

	status int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,

		status: http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.status = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("%T does not implement the Hijacker interface", rw.ResponseWriter)
	}

	return h.Hijack()
}
