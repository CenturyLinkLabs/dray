package api

import (
	"bytes"
	"errors"
	"io"
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
	notFoundErr error
	serverErr   error
	jm          *mockJobManager
	r           *mockRequestHelper
	w           *httptest.ResponseRecorder
)

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func setUp() {
	j = &job.Job{ID: "123"}
	jm = &mockJobManager{}
	r = &mockRequestHelper{}
	w = httptest.NewRecorder()
	notFoundErr = job.NotFoundError(j.ID)
	serverErr = errors.New("oops")

}

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

func TestListJobsSuccess(t *testing.T) {
	setUp()

	jobs := []job.Job{*j}
	jm.On("ListAll").Return(jobs, nil)

	listJobs(jm, r, w)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[{\"id\":\"123\"}]\n", w.Body.String())
	jm.Mock.AssertExpectations(t)
	r.Mock.AssertExpectations(t)
}

func TestListJobsError(t *testing.T) {
	setUp()

	jm.On("ListAll").Return(nil, serverErr)

	listJobs(jm, r, w)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "", w.Body.String())
}

func TestCreateJobSuccess(t *testing.T) {
	setUp()
	payload := "{\"name\":\"foo\"}\n"
	body := nopCloser{bytes.NewBufferString(payload)}

	jm.On("Create", mock.AnythingOfType("*job.Job")).Return(nil)
	jm.On("Execute", mock.AnythingOfType("*job.Job")).Return(nil)
	r.On("Body").Return(body)

	createJob(jm, r, w)
	time.Sleep(time.Millisecond)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, payload, w.Body.String())
	jm.Mock.AssertExpectations(t)
	r.Mock.AssertExpectations(t)
}

func TestCreateJobJSONError(t *testing.T) {
	setUp()
	body := nopCloser{bytes.NewBufferString("")}

	r.On("Body").Return(body)

	createJob(jm, r, w)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "", w.Body.String())
	jm.Mock.AssertExpectations(t)
	r.Mock.AssertExpectations(t)
}

func TestCreateJobError(t *testing.T) {
	setUp()
	body := nopCloser{bytes.NewBufferString("{}")}

	jm.On("Create", mock.AnythingOfType("*job.Job")).Return(serverErr)
	r.On("Body").Return(body)

	createJob(jm, r, w)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "", w.Body.String())
	jm.Mock.AssertExpectations(t)
	r.Mock.AssertExpectations(t)
}

func TestGetJobSuccess(t *testing.T) {
	setUp()

	jm.On("GetByID", j.ID).Return(j, nil)
	r.On("Param", "jobid").Return(j.ID)

	getJob(jm, r, w)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "{\"id\":\"123\"}\n", w.Body.String())
	jm.Mock.AssertExpectations(t)
	r.Mock.AssertExpectations(t)
}

func TestGetJobNotFound(t *testing.T) {
	setUp()

	jm.On("GetByID", j.ID).Return(nil, notFoundErr)
	r.On("Param", "jobid").Return(j.ID)

	getJob(jm, r, w)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "", w.Body.String())
	jm.Mock.AssertExpectations(t)
	r.Mock.AssertExpectations(t)
}

func TestGetJobServerError(t *testing.T) {
	setUp()

	jm.On("GetByID", j.ID).Return(nil, serverErr)
	r.On("Param", "jobid").Return(j.ID)

	getJob(jm, r, w)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "", w.Body.String())
	jm.Mock.AssertExpectations(t)
	r.Mock.AssertExpectations(t)
}

func TestGetJobLogSuccess(t *testing.T) {
	setUp()
	index := 99
	jobLog := &job.JobLog{Lines: []string{"foo", "bar"}}

	jm.On("GetByID", j.ID).Return(j, nil)
	jm.On("GetLog", j, index).Return(jobLog, nil)
	r.On("Param", "jobid").Return(j.ID)
	r.On("Query", "index").Return(strconv.Itoa(index))

	getJobLog(jm, r, w)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "{\"lines\":[\"foo\",\"bar\"]}\n", w.Body.String())
	jm.Mock.AssertExpectations(t)
	r.Mock.AssertExpectations(t)
}

func TestGetJobLogNotFound(t *testing.T) {
	setUp()
	index := 99

	jm.On("GetByID", j.ID).Return(nil, notFoundErr)
	r.On("Param", "jobid").Return(j.ID)
	r.On("Query", "index").Return(strconv.Itoa(index))

	getJobLog(jm, r, w)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "", w.Body.String())
	jm.Mock.AssertExpectations(t)
	r.Mock.AssertExpectations(t)
}

func TestGetJobLogError(t *testing.T) {
	setUp()
	index := 99

	jm.On("GetByID", j.ID).Return(j, nil)
	jm.On("GetLog", j, index).Return(nil, serverErr)
	r.On("Param", "jobid").Return(j.ID)
	r.On("Query", "index").Return(strconv.Itoa(index))

	getJobLog(jm, r, w)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "", w.Body.String())
	jm.Mock.AssertExpectations(t)
	r.Mock.AssertExpectations(t)
}

func TestDeleteJobSuccess(t *testing.T) {
	setUp()

	jm.On("GetByID", j.ID).Return(j, nil)
	jm.On("Delete", j).Return(nil)
	r.On("Param", "jobid").Return(j.ID)

	deleteJob(jm, r, w)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "", w.Body.String())
	jm.Mock.AssertExpectations(t)
	r.Mock.AssertExpectations(t)
}

func TestDeleteJobNotFound(t *testing.T) {
	setUp()

	jm.On("GetByID", j.ID).Return(nil, notFoundErr)
	r.On("Param", "jobid").Return(j.ID)

	deleteJob(jm, r, w)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "", w.Body.String())
	jm.Mock.AssertExpectations(t)
	r.Mock.AssertExpectations(t)
}

func TestDeleteJobError(t *testing.T) {
	setUp()

	jm.On("GetByID", j.ID).Return(j, nil)
	jm.On("Delete", j).Return(serverErr)
	r.On("Param", "jobid").Return(j.ID)

	deleteJob(jm, r, w)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "", w.Body.String())
	jm.Mock.AssertExpectations(t)
	r.Mock.AssertExpectations(t)
}
