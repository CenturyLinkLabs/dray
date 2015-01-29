package api

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CenturyLinkLabs/dray/job"
	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
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

type APITestSuite struct {
	suite.Suite

	j           *job.Job
	jm          *mockJobManager
	svr         *httptest.Server
	client      *http.Client
	serverErr   error
	notFoundErr error
}

func (suite *APITestSuite) SetupTest() {
	suite.j = &job.Job{ID: "123"}
	suite.jm = &mockJobManager{}

	suite.svr = httptest.NewServer(NewServer(suite.jm).createRouter())
	suite.client = &http.Client{}

	suite.serverErr = errors.New("oops")
	suite.notFoundErr = job.NotFoundError(suite.j.ID)
}

func (suite *APITestSuite) TestListJobsSuccess() {
	jobs := []job.Job{*suite.j}
	suite.jm.On("ListAll").Return(jobs, nil)

	res, _ := http.Get(suite.url("jobs"))
	body, _ := ioutil.ReadAll(res.Body)

	suite.Equal(http.StatusOK, res.StatusCode)
	suite.Equal("application/json", res.Header["Content-Type"][0])
	suite.Equal("[{\"id\":\"123\"}]\n", string(body))
	suite.jm.Mock.AssertExpectations(suite.T())
}

func (suite *APITestSuite) TestListJobsError() {
	suite.jm.On("ListAll").Return(nil, suite.serverErr)

	res, _ := http.Get(suite.url("jobs"))
	body, _ := ioutil.ReadAll(res.Body)

	suite.Equal(http.StatusInternalServerError, res.StatusCode)
	suite.Equal("text/plain; charset=utf-8", res.Header["Content-Type"][0])
	suite.Equal("", string(body))
	suite.jm.Mock.AssertExpectations(suite.T())
}

func (suite *APITestSuite) TestCreateJobSuccess() {
	payload := "{\"name\":\"foo\"}\n"

	suite.jm.On("Create", mock.AnythingOfType("*job.Job")).Return(nil)
	suite.jm.On("Execute", mock.AnythingOfType("*job.Job")).Return(nil)

	res, _ := http.Post(suite.url("jobs"), "application/json", bytes.NewBufferString(payload))
	body, _ := ioutil.ReadAll(res.Body)
	time.Sleep(time.Millisecond)

	suite.Equal(http.StatusCreated, res.StatusCode)
	suite.Equal(payload, string(body))
	suite.jm.Mock.AssertExpectations(suite.T())
}

func (suite *APITestSuite) TestCreateJobJSONError() {
	res, _ := http.Post(suite.url("jobs"), "application/json", bytes.NewBufferString(""))
	body, _ := ioutil.ReadAll(res.Body)

	suite.Equal(http.StatusInternalServerError, res.StatusCode)
	suite.Equal("", string(body))
	suite.jm.Mock.AssertExpectations(suite.T())
}

func (suite *APITestSuite) TestCreateJobError() {
	suite.jm.On("Create", mock.AnythingOfType("*job.Job")).Return(suite.serverErr)

	res, _ := http.Post(suite.url("jobs"), "application/json", bytes.NewBufferString("{}"))
	body, _ := ioutil.ReadAll(res.Body)

	suite.Equal(http.StatusInternalServerError, res.StatusCode)
	suite.Equal("", string(body))
	suite.jm.Mock.AssertExpectations(suite.T())
}

func (suite *APITestSuite) TestGetJobSuccess() {
	suite.jm.On("GetByID", suite.j.ID).Return(suite.j, nil)

	res, _ := http.Get(suite.url("jobs", suite.j.ID))
	body, _ := ioutil.ReadAll(res.Body)

	suite.Equal(http.StatusOK, res.StatusCode)
	suite.Equal("{\"id\":\"123\"}\n", string(body))
	suite.jm.Mock.AssertExpectations(suite.T())
}

func (suite *APITestSuite) TestGetJobNotFound() {
	suite.jm.On("GetByID", suite.j.ID).Return(nil, suite.notFoundErr)

	res, _ := http.Get(suite.url("jobs", suite.j.ID))
	body, _ := ioutil.ReadAll(res.Body)

	suite.Equal(http.StatusNotFound, res.StatusCode)
	suite.Equal("", string(body))
	suite.jm.Mock.AssertExpectations(suite.T())
}

func (suite *APITestSuite) TestGetJobServerError() {
	suite.jm.On("GetByID", suite.j.ID).Return(nil, suite.serverErr)

	res, _ := http.Get(suite.url("jobs", suite.j.ID))
	body, _ := ioutil.ReadAll(res.Body)

	suite.Equal(http.StatusInternalServerError, res.StatusCode)
	suite.Equal("", string(body))
	suite.jm.Mock.AssertExpectations(suite.T())
}

func (suite *APITestSuite) TestGetJobLogSuccess() {
	index := 99
	jobLog := &job.JobLog{Lines: []string{"foo", "bar"}}

	suite.jm.On("GetByID", suite.j.ID).Return(suite.j, nil)
	suite.jm.On("GetLog", suite.j, index).Return(jobLog, nil)

	res, _ := http.Get(suite.url("jobs", suite.j.ID, "log") + "?index=" + strconv.Itoa(index))
	body, _ := ioutil.ReadAll(res.Body)

	suite.Equal(http.StatusOK, res.StatusCode)
	suite.Equal("{\"lines\":[\"foo\",\"bar\"]}\n", string(body))
	suite.jm.Mock.AssertExpectations(suite.T())
}

func (suite *APITestSuite) TestGetJobLogNotFound() {
	suite.jm.On("GetByID", suite.j.ID).Return(nil, suite.notFoundErr)

	res, _ := http.Get(suite.url("jobs", suite.j.ID, "log"))
	body, _ := ioutil.ReadAll(res.Body)

	suite.Equal(http.StatusNotFound, res.StatusCode)
	suite.Equal("", string(body))
	suite.jm.Mock.AssertExpectations(suite.T())
}

func (suite *APITestSuite) TestGetJobLogError() {
	index := 99

	suite.jm.On("GetByID", suite.j.ID).Return(suite.j, nil)
	suite.jm.On("GetLog", suite.j, index).Return(nil, suite.serverErr)

	res, _ := http.Get(suite.url("jobs", suite.j.ID, "log") + "?index=" + strconv.Itoa(index))
	body, _ := ioutil.ReadAll(res.Body)

	suite.Equal(http.StatusInternalServerError, res.StatusCode)
	suite.Equal("", string(body))
	suite.jm.Mock.AssertExpectations(suite.T())
}

func (suite *APITestSuite) TestDeleteJobSuccess() {
	suite.jm.On("GetByID", suite.j.ID).Return(suite.j, nil)
	suite.jm.On("Delete", suite.j).Return(nil)

	req, _ := http.NewRequest("DELETE", suite.url("jobs", suite.j.ID), nil)
	res, _ := suite.client.Do(req)
	body, _ := ioutil.ReadAll(res.Body)

	suite.Equal(http.StatusNoContent, res.StatusCode)
	suite.Equal("", string(body))
	suite.jm.Mock.AssertExpectations(suite.T())
}

func (suite *APITestSuite) TestDeleteJobNotFound() {
	suite.jm.On("GetByID", suite.j.ID).Return(nil, suite.notFoundErr)

	req, _ := http.NewRequest("DELETE", suite.url("jobs", suite.j.ID), nil)
	res, _ := suite.client.Do(req)
	body, _ := ioutil.ReadAll(res.Body)

	suite.Equal(http.StatusNotFound, res.StatusCode)
	suite.Equal("", string(body))
	suite.jm.Mock.AssertExpectations(suite.T())
}

func (suite *APITestSuite) TestDeleteJobError() {
	suite.jm.On("GetByID", suite.j.ID).Return(suite.j, nil)
	suite.jm.On("Delete", suite.j).Return(suite.serverErr)

	req, _ := http.NewRequest("DELETE", suite.url("jobs", suite.j.ID), nil)
	res, _ := suite.client.Do(req)
	body, _ := ioutil.ReadAll(res.Body)

	suite.Equal(http.StatusInternalServerError, res.StatusCode)
	suite.Equal("", string(body))
	suite.jm.Mock.AssertExpectations(suite.T())
}

func (suite *APITestSuite) url(parts ...string) string {
	parts = append([]string{suite.svr.URL}, parts...)
	return strings.Join(parts, "/")
}

func TestAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}
