package api

import (
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

// Convenience methods for interacting with http.Request
type requestHelper interface {
	Param(string) string
	Query(string) string
	Body() io.ReadCloser
}

type requestWrapper struct {
	httpRequest *http.Request
}

func (r *requestWrapper) Param(key string) string {
	return mux.Vars(r.httpRequest)[key]
}

func (r *requestWrapper) Query(key string) string {
	v := r.httpRequest.URL.Query()[key]

	if len(v) == 0 {
		return ""
	}

	return v[0]
}

func (r *requestWrapper) Body() io.ReadCloser {
	return r.httpRequest.Body
}
