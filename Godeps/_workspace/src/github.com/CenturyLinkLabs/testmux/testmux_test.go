package testmux

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterFunc(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	status := 202
	handler := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }

	router := Router{}
	router.RegisterFunc(req.Method, req.URL.Path, handler)

	router.ServeHTTP(w, req)

	assert.Equal(t, w.Code, status)
}

func TestRegisterResp(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	status := 202
	body := "Hello"

	router := Router{}
	router.RegisterResp(req.Method, req.URL.Path, status, body)

	router.ServeHTTP(w, req)

	assert.Equal(t, 202, w.Code)
	assert.Equal(t, body+"\n", w.Body.String())
}

func TestServeHTTP_Success(t *testing.T) {
	var w *httptest.ResponseRecorder
	var req *http.Request

	router := Router{}
	router.RegisterResp("GET", "/foo", 200, "")
	router.RegisterResp("GET", "/bar", 201, "")
	router.RegisterResp("PUT", "/foo", 202, "")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/foo", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, w.Code, 200)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/bar", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/foo", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 202, w.Code)

	assert.Empty(t, router.errors)
}

func TestServeHTTP_UnexpectedRequest(t *testing.T) {
	var w *httptest.ResponseRecorder
	var req *http.Request

	router := Router{}
	router.RegisterResp("GET", "/foo", 200, "")
	router.RegisterResp("GET", "/bar", 201, "")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/foo", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, w.Code, 200)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/bar", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/foo", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)

	assert.Len(t, router.errors, 1)
	assert.Contains(t, router.errors, routeError{unexpected, "PUT", "/foo"})
}

func TestServeHTTP_OutOfOrderRequest(t *testing.T) {
	var w *httptest.ResponseRecorder
	var req *http.Request

	router := Router{}
	router.RegisterResp("GET", "/foo", 200, "")
	router.RegisterResp("PUT", "/foo", 202, "")
	router.RegisterResp("GET", "/bar", 201, "")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/foo", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, w.Code, 200)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/bar", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/foo", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 202, w.Code)

	assert.Len(t, router.errors, 2)
	assert.Contains(t, router.errors, routeError{disorderly, "GET", "/bar"})
	assert.Contains(t, router.errors, routeError{disorderly, "PUT", "/foo"})
}

func TestAssertVisited_Success(t *testing.T) {
	router := Router{}
	router.RegisterResp("GET", "/foo", 200, "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	router.ServeHTTP(w, req)

	tt := &testing.T{}
	result := router.AssertVisited(tt)

	assert.True(t, result)
	assert.False(t, tt.Failed())
}

func TestAssertVisited_UnvisitedRoute(t *testing.T) {
	router := Router{}
	router.RegisterResp("GET", "/foo", 200, "")

	tt := &testing.T{}
	result := router.AssertVisited(tt)

	assert.False(t, result)
	assert.True(t, tt.Failed())

	assert.Len(t, router.errors, 1)
	assert.Contains(t, router.errors, routeError{unvisited, "GET", "/foo"})
}

func TestAssertVisited_OutOfOrderRoutes(t *testing.T) {
	var w *httptest.ResponseRecorder
	var req *http.Request

	router := Router{}
	router.RegisterResp("GET", "/foo", 200, "")
	router.RegisterResp("GET", "/bar", 200, "")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/bar", nil)
	router.ServeHTTP(w, req)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/foo", nil)
	router.ServeHTTP(w, req)

	tt := &testing.T{}
	result := router.AssertVisited(tt)

	assert.True(t, result)
	assert.False(t, tt.Failed())
}

func TestAssertVisitedInOrder_Success(t *testing.T) {
	var w *httptest.ResponseRecorder
	var req *http.Request

	router := Router{}
	router.RegisterResp("GET", "/foo", 200, "")
	router.RegisterResp("GET", "/bar", 200, "")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/foo", nil)
	router.ServeHTTP(w, req)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/bar", nil)
	router.ServeHTTP(w, req)

	tt := &testing.T{}
	result := router.AssertVisitedInOrder(tt)

	assert.True(t, result)
	assert.False(t, tt.Failed())
}

func TestAssertVisitedInOrder_OutOfOrderRoutes(t *testing.T) {
	var w *httptest.ResponseRecorder
	var req *http.Request

	router := Router{}
	router.RegisterResp("GET", "/foo", 200, "")
	router.RegisterResp("GET", "/bar", 200, "")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/bar", nil)
	router.ServeHTTP(w, req)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/foo", nil)
	router.ServeHTTP(w, req)

	tt := &testing.T{}
	result := router.AssertVisitedInOrder(tt)

	assert.False(t, result)
	assert.True(t, tt.Failed())
}
