package api

import (
	"bytes"
	"errors"
	"fmt"
	_ "io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/CenturyLinkLabs/dray/job"
	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	j           *job.Job
	jm          *mockJobManager
	svr         *httptest.Server
	client      *http.Client
	serverErr   error
	notFoundErr error
)

func init() {
	log.SetLevel(log.PanicLevel)
}

type mockJobManager struct {
	mock.Mock
}

func (m *mockJobManager) ListAll() ([]job.Job, error) {
	var jobs []job.Job
	args := m.Mock.Called()

	if jobsArg := args.Get(0); jobsArg != nil {
		jobs = jobsArg.([]job.Job)
	}
	return jobs, args.Error(1)
}

func (m *mockJobManager) GetByID(jobID string) (*job.Job, error) {
	var j *job.Job
	args := m.Mock.Called(jobID)

	if jobArg := args.Get(0); jobArg != nil {
		j = jobArg.(*job.Job)
	}

	return j, args.Error(1)
}

func (m *mockJobManager) Create(j *job.Job) error {
	args := m.Mock.Called(j)
	return args.Error(0)
}

func (m *mockJobManager) Execute(j *job.Job) error {
	args := m.Mock.Called(j)
	return args.Error(0)
}

func (m *mockJobManager) GetLog(j *job.Job, index int) (*job.JobLog, error) {
	var jl *job.JobLog
	args := m.Mock.Called(j, index)

	if logArg := args.Get(0); logArg != nil {
		jl = logArg.(*job.JobLog)
	}

	return jl, args.Error(1)
}

func (m *mockJobManager) Delete(job *job.Job) error {
	args := m.Mock.Called(job)
	return args.Error(0)
}

func setUp() {
	j = &job.Job{ID: "123"}
	jm = &mockJobManager{}
	jobServer := NewServer(jm)
	svr = httptest.NewServer(jobServer.createRouter())
	client = &http.Client{}
	serverErr = errors.New("oops")
	notFoundErr = job.NotFoundError(j.ID)
}

func TestListJobsSuccess(t *testing.T) {
	setUp()

	jobs := []job.Job{*j}
	jm.On("ListAll").Return(jobs, nil)

	res, _ := http.Get(url("jobs"))
	body, _ := ioutil.ReadAll(res.Body)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "application/json", res.Header["Content-Type"][0])
	assert.Equal(t, "[{\"id\":\"123\"}]\n", string(body))
	jm.Mock.AssertExpectations(t)
}

func TestListJobsError(t *testing.T) {
	setUp()

	jm.On("ListAll").Return(nil, serverErr)

	res, _ := http.Get(url("jobs"))
	body, _ := ioutil.ReadAll(res.Body)

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", res.Header["Content-Type"][0])
	assert.Equal(t, "", string(body))
	jm.Mock.AssertExpectations(t)
}

func TestCreateJobSuccess(t *testing.T) {
	setUp()
	payload := "{\"name\":\"foo\"}\n"

	jm.On("Create", mock.AnythingOfType("*job.Job")).Return(nil)
	jm.On("Execute", mock.AnythingOfType("*job.Job")).Return(nil)

	res, _ := http.Post(url("jobs"), "application/json", bytes.NewBufferString(payload))
	body, _ := ioutil.ReadAll(res.Body)
	time.Sleep(time.Millisecond)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, payload, string(body))
	jm.Mock.AssertExpectations(t)
}

func TestCreateJobJSONError(t *testing.T) {
	setUp()

	res, _ := http.Post(url("jobs"), "application/json", bytes.NewBufferString(""))
	body, _ := ioutil.ReadAll(res.Body)

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	assert.Equal(t, "", string(body))
	jm.Mock.AssertExpectations(t)
}

func TestCreateJobError(t *testing.T) {
	setUp()

	jm.On("Create", mock.AnythingOfType("*job.Job")).Return(serverErr)

	res, _ := http.Post(url("jobs"), "application/json", bytes.NewBufferString("{}"))
	body, _ := ioutil.ReadAll(res.Body)

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	assert.Equal(t, "", string(body))
	jm.Mock.AssertExpectations(t)
}

func TestGetJobSuccess(t *testing.T) {
	setUp()

	jm.On("GetByID", j.ID).Return(j, nil)

	res, _ := http.Get(url("jobs/" + j.ID))
	body, _ := ioutil.ReadAll(res.Body)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "{\"id\":\"123\"}\n", string(body))
	jm.Mock.AssertExpectations(t)
}

func TestGetJobNotFound(t *testing.T) {
	setUp()

	jm.On("GetByID", j.ID).Return(nil, notFoundErr)

	res, _ := http.Get(url("jobs/" + j.ID))
	body, _ := ioutil.ReadAll(res.Body)

	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Equal(t, "", string(body))
	jm.Mock.AssertExpectations(t)
}

func TestGetJobServerError(t *testing.T) {
	setUp()

	jm.On("GetByID", j.ID).Return(nil, serverErr)

	res, _ := http.Get(url("jobs/" + j.ID))
	body, _ := ioutil.ReadAll(res.Body)

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	assert.Equal(t, "", string(body))
	jm.Mock.AssertExpectations(t)
}

func TestGetJobLogSuccess(t *testing.T) {
	setUp()
	index := 99
	jobLog := &job.JobLog{Lines: []string{"foo", "bar"}}

	jm.On("GetByID", j.ID).Return(j, nil)
	jm.On("GetLog", j, index).Return(jobLog, nil)

	res, _ := http.Get(url("jobs/" + j.ID + "/log" + "?index=" + strconv.Itoa(index)))
	body, _ := ioutil.ReadAll(res.Body)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "{\"lines\":[\"foo\",\"bar\"]}\n", string(body))
	jm.Mock.AssertExpectations(t)
}

func TestGetJobLogNotFound(t *testing.T) {
	setUp()

	jm.On("GetByID", j.ID).Return(nil, notFoundErr)

	res, _ := http.Get(url("jobs/" + j.ID + "/log"))
	body, _ := ioutil.ReadAll(res.Body)

	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Equal(t, "", string(body))
	jm.Mock.AssertExpectations(t)
}

func TestGetJobLogError(t *testing.T) {
	setUp()
	index := 99

	jm.On("GetByID", j.ID).Return(j, nil)
	jm.On("GetLog", j, index).Return(nil, serverErr)

	res, _ := http.Get(url("jobs/" + j.ID + "/log" + "?index=" + strconv.Itoa(index)))
	body, _ := ioutil.ReadAll(res.Body)

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	assert.Equal(t, "", string(body))
	jm.Mock.AssertExpectations(t)
}

func TestDeleteJobSuccess(t *testing.T) {
	setUp()

	jm.On("GetByID", j.ID).Return(j, nil)
	jm.On("Delete", j).Return(nil)

	req, _ := http.NewRequest("DELETE", url("jobs/"+j.ID), nil)
	res, _ := client.Do(req)
	body, _ := ioutil.ReadAll(res.Body)

	assert.Equal(t, http.StatusNoContent, res.StatusCode)
	assert.Equal(t, "", string(body))
	jm.Mock.AssertExpectations(t)
}

func TestDeleteJobNotFound(t *testing.T) {
	setUp()

	jm.On("GetByID", j.ID).Return(nil, notFoundErr)

	req, _ := http.NewRequest("DELETE", url("jobs/"+j.ID), nil)
	res, _ := client.Do(req)
	body, _ := ioutil.ReadAll(res.Body)

	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Equal(t, "", string(body))
	jm.Mock.AssertExpectations(t)
}

func TestDeleteJobError(t *testing.T) {
	setUp()

	jm.On("GetByID", j.ID).Return(j, nil)
	jm.On("Delete", j).Return(serverErr)

	req, _ := http.NewRequest("DELETE", url("jobs/"+j.ID), nil)
	res, _ := client.Do(req)
	body, _ := ioutil.ReadAll(res.Body)

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	assert.Equal(t, "", string(body))
	jm.Mock.AssertExpectations(t)
}

func url(path string) string {
	return fmt.Sprintf("%s/%s", svr.URL, path)
}
