package job

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CenturyLinkLabs/testmux"
	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type mockExecutor struct {
	mock.Mock

	output string
}

func (m *mockExecutor) Start(job *Job, stdIn io.Reader, stdOut, stdErr io.WriteCloser) error {
	args := m.Mock.Called(job, stdIn, stdOut, stdErr)

	if len(m.output) > 0 {
		go func() {
			defer stdOut.Close()
			defer stdErr.Close()
			stdOut.Write([]byte(m.output))
			stdOut.Write([]byte{'\n'})
		}()
	} else {
		stdOut.Close()
		stdErr.Close()
	}

	return args.Error(0)
}

func (m *mockExecutor) Inspect(job *Job) error {
	args := m.Mock.Called(job)
	return args.Error(0)
}

func (m *mockExecutor) CleanUp(job *Job) error {
	args := m.Mock.Called(job)
	return args.Error(0)
}

type JobStepExecutorTestSuite struct {
	suite.Suite

	job     *Job
	jobStep *JobStep
	mux     *testmux.Router
	server  *httptest.Server
	jse     JobStepExecutor
}

func (suite *JobStepExecutorTestSuite) SetupTest() {
	log.SetLevel(log.PanicLevel)

	suite.mux = &testmux.Router{}
	suite.server = httptest.NewServer(suite.mux)
	suite.jse = NewExecutor(suite.server.URL)

	suite.jobStep = &JobStep{
		id:     "abc123",
		Source: "foo",
	}

	suite.job = &Job{
		Steps: []JobStep{*suite.jobStep},
	}
}

func (suite *JobStepExecutorTestSuite) TearDownTest() {
	suite.server.Close()
}

func (suite *JobStepExecutorTestSuite) TestStart_Success() {
	stdIn := &bytes.Buffer{}
	stdOutReader, stdOutWriter := io.Pipe()
	_, stdErrWriter := io.Pipe()

	suite.mux.RegisterResp("GET", "/images/foo/json", http.StatusOK,
		"{\"ID\":\"xyz789\"}")
	suite.mux.RegisterResp("POST", "/containers/create", http.StatusCreated,
		"{\"ID\":\"123abc\"}")
	suite.mux.RegisterResp("POST", "/containers/123abc/start", http.StatusNoContent, "")
	suite.mux.RegisterFunc("POST", "/containers/123abc/attach",
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte{1, 0, 0, 0, 0, 0, 0, 5})
			w.Write([]byte("hello"))
		})

	err := suite.jse.Start(suite.job, stdIn, stdOutWriter, stdErrWriter)

	suite.NoError(err)
	suite.Equal("123abc", suite.job.currentStep().id)

	stdOutScanner := bufio.NewScanner(stdOutReader)
	stdOutScanner.Scan()
	suite.Equal("hello", stdOutScanner.Text())

	suite.mux.AssertVisited(suite.T())
}

func (suite *JobStepExecutorTestSuite) TestStart_CreateError() {
	stdIn := &bytes.Buffer{}
	_, stdOutWriter := io.Pipe()
	_, stdErrWriter := io.Pipe()

	suite.mux.RegisterResp("GET", "/images/foo/json", http.StatusOK,
		"{\"ID\":\"xyz789\"}")
	suite.mux.RegisterResp("POST", "/containers/create", http.StatusInternalServerError, "")

	err := suite.jse.Start(suite.job, stdIn, stdOutWriter, stdErrWriter)

	suite.EqualError(err, "API error (500): \n")
	suite.mux.AssertVisited(suite.T())
}

func (suite *JobStepExecutorTestSuite) TestStart_StartError() {
	stdIn := &bytes.Buffer{}
	stdOutReader, stdOutWriter := io.Pipe()
	_, stdErrWriter := io.Pipe()

	suite.mux.RegisterResp("GET", "/images/foo/json", http.StatusOK,
		"{\"ID\":\"xyz789\"}")
	suite.mux.RegisterResp("POST", "/containers/create", http.StatusCreated,
		"{\"ID\":\"123abc\"}")
	suite.mux.RegisterResp("POST", "/containers/123abc/start", http.StatusBadRequest, "")

	suite.mux.RegisterFunc("POST", "/containers/123abc/attach",
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte{1, 0, 0, 0, 0, 0, 0, 5})
			w.Write([]byte("hello"))
		})

	err := suite.jse.Start(suite.job, stdIn, stdOutWriter, stdErrWriter)

	// Must read in order to block until the attach call is complete
	stdOutReader.Read([]byte{})

	suite.EqualError(err, "API error (400): \n")
	suite.mux.AssertVisited(suite.T())
}

func (suite *JobStepExecutorTestSuite) TestStart_MissingImage() {
	stdIn := &bytes.Buffer{}
	stdOutReader, stdOutWriter := io.Pipe()
	_, stdErrWriter := io.Pipe()

	suite.mux.RegisterResp("GET", "/images/foo/json", http.StatusNotFound, "")
	suite.mux.RegisterResp("POST", "/images/create", http.StatusOK, "")
	suite.mux.RegisterResp("POST", "/containers/create", http.StatusCreated,
		"{\"ID\":\"123abc\"}")
	suite.mux.RegisterResp("POST", "/containers/123abc/start", http.StatusOK, "")

	suite.mux.RegisterFunc("POST", "/containers/123abc/attach",
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte{1, 0, 0, 0, 0, 0, 0, 5})
			w.Write([]byte("hello"))
		})

	err := suite.jse.Start(suite.job, stdIn, stdOutWriter, stdErrWriter)

	// Must read in order to block until the attach call is complete
	stdOutReader.Read([]byte{})

	suite.NoError(err)
	suite.mux.AssertVisited(suite.T())
}

func (suite *JobStepExecutorTestSuite) TestStart_PullError() {
	stdIn := &bytes.Buffer{}
	_, stdOutWriter := io.Pipe()
	_, stdErrWriter := io.Pipe()

	suite.mux.RegisterResp("GET", "/images/foo/json", http.StatusNotFound, "")
	suite.mux.RegisterResp("POST", "/images/create", http.StatusNotFound, "")

	err := suite.jse.Start(suite.job, stdIn, stdOutWriter, stdErrWriter)

	suite.EqualError(err, "API error (404): \n")
	suite.mux.AssertVisited(suite.T())
}

func (suite *JobStepExecutorTestSuite) TestStart_ForceRefresh() {
	suite.job.currentStep().Refresh = true
	stdIn := &bytes.Buffer{}
	stdOutReader, stdOutWriter := io.Pipe()
	_, stdErrWriter := io.Pipe()

	suite.mux.RegisterResp("GET", "/images/foo/json", http.StatusOK,
		"{\"Id\":\"xyz890\"}")
	suite.mux.RegisterResp("POST", "/images/create", http.StatusOK, "")
	suite.mux.RegisterResp("GET", "/images/foo/json", http.StatusOK,
		"{\"Id\":\"15930e\"}")
	suite.mux.RegisterResp("DELETE", "/images/xyz890", http.StatusOK, "")
	suite.mux.RegisterResp("POST", "/containers/create", http.StatusCreated,
		"{\"ID\":\"123abc\"}")
	suite.mux.RegisterResp("POST", "/containers/123abc/start", http.StatusOK, "")

	suite.mux.RegisterFunc("POST", "/containers/123abc/attach",
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte{1, 0, 0, 0, 0, 0, 0, 5})
			w.Write([]byte("hello"))
		})

	err := suite.jse.Start(suite.job, stdIn, stdOutWriter, stdErrWriter)

	// Must read in order to block until the attach call is complete
	stdOutReader.Read([]byte{})

	suite.NoError(err)
	suite.mux.AssertVisited(suite.T())
}

func (suite *JobStepExecutorTestSuite) TestInspect_Success() {
	suite.mux.RegisterResp("GET", "/containers/abc123/json", http.StatusOK,
		"{\"State\":{\"ExitCode\":0}}")

	err := suite.jse.Inspect(suite.job)

	suite.NoError(err)
	suite.mux.AssertVisited(suite.T())
}

func (suite *JobStepExecutorTestSuite) TestInspect_Error() {
	suite.mux.RegisterResp("GET", "/containers/abc123/json", http.StatusNotFound, "")

	err := suite.jse.Inspect(suite.job)

	suite.EqualError(err, "No such container: abc123")
	suite.mux.AssertVisited(suite.T())
}

func (suite *JobStepExecutorTestSuite) TestInspect_ErrorExit() {
	suite.mux.RegisterResp("GET", "/containers/abc123/json", http.StatusOK,
		"{\"State\":{\"ExitCode\":99}}")

	err := suite.jse.Inspect(suite.job)

	suite.EqualError(err, "Container exit code: 99")
	suite.mux.AssertVisited(suite.T())
}

func (suite *JobStepExecutorTestSuite) TestCleanUp_Success() {
	suite.mux.RegisterResp("DELETE", "/containers/abc123", http.StatusNoContent, "")

	err := suite.jse.CleanUp(suite.job)

	suite.NoError(err)
	suite.mux.AssertVisited(suite.T())
}

func (suite *JobStepExecutorTestSuite) TestCleanUp_Error() {
	suite.mux.RegisterResp("DELETE", "/containers/abc123", http.StatusNotFound, "")

	err := suite.jse.CleanUp(suite.job)

	suite.EqualError(err, "No such container: abc123")
	suite.mux.AssertVisited(suite.T())
}

func TestJobStepExecutor(t *testing.T) {
	suite.Run(t, new(JobStepExecutorTestSuite))
}
