package api

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRequestHelper struct {
	mock.Mock
}

func (r *mockRequestHelper) Param(name string) string {
	args := r.Mock.Called(name)
	return args.String(0)
}

func (r *mockRequestHelper) Query(name string) string {
	args := r.Mock.Called(name)
	return args.String(0)
}

func (r *mockRequestHelper) Body() io.ReadCloser {
	args := r.Mock.Called()
	return args.Get(0).(io.ReadCloser)
}

func TestRequestWrapperQueryNoValue(t *testing.T) {
	r, _ := http.NewRequest("GET", "/jobs?name=foo", nil)
	rw := requestWrapper{httpRequest: r}

	val := rw.Query("age")

	assert.Equal(t, "", val)
}

func TestRequestWrapperQuerySingleValue(t *testing.T) {
	r, _ := http.NewRequest("GET", "/jobs?name=foo", nil)
	rw := requestWrapper{httpRequest: r}

	val := rw.Query("name")

	assert.Equal(t, "foo", val)
}

func TestRequestWrapperQueryMultiValue(t *testing.T) {
	r, _ := http.NewRequest("GET", "/jobs?name=foo&name=bar", nil)
	rw := requestWrapper{httpRequest: r}

	val := rw.Query("name")

	assert.Equal(t, "foo", val)
}

func TestRequestWrapeprBody(t *testing.T) {
	expected := "foobar"
	r, _ := http.NewRequest("POST", "/jobs", bytes.NewBufferString("foobar"))
	rw := requestWrapper{httpRequest: r}

	body := rw.Body()
	buffer := &bytes.Buffer{}
	buffer.ReadFrom(body)

	assert.Equal(t, expected, buffer.String())
}
