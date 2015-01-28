package api

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
