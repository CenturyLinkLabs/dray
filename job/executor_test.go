package job

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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
	mux     *http.ServeMux
	server  *httptest.Server
	jse     JobStepExecutor
}

func (suite *JobStepExecutorTestSuite) SetupTest() {
	suite.mux = http.NewServeMux()
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
	id := "foo123abc"
	stdIn := &bytes.Buffer{}
	stdOutReader, stdOutWriter := io.Pipe()
	_, stdErrWriter := io.Pipe()

	inspectPath := fmt.Sprintf("/images/%s/json", suite.jobStep.Source)
	suite.mux.HandleFunc(inspectPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "{\"ID\":\"xyz789\"}")
	})

	suite.mux.HandleFunc("/containers/create", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, "{\"ID\":\""+id+"\"}")
	})

	attachPath := fmt.Sprintf("/containers/%s/attach", id)
	suite.mux.HandleFunc(attachPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte{1, 0, 0, 0, 0, 0, 0, 5})
		w.Write([]byte("hello"))
	})

	startPath := fmt.Sprintf("/containers/%s/start", id)
	suite.mux.HandleFunc(startPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	err := suite.jse.Start(suite.job, stdIn, stdOutWriter, stdErrWriter)

	suite.NoError(err)
	suite.Equal(id, suite.job.currentStep().id)

	stdOutScanner := bufio.NewScanner(stdOutReader)
	stdOutScanner.Scan()
	suite.Equal("hello", stdOutScanner.Text())
}

func (suite *JobStepExecutorTestSuite) TestStart_CreateError() {
	stdIn := &bytes.Buffer{}
	_, stdOutWriter := io.Pipe()
	_, stdErrWriter := io.Pipe()

	inspectPath := fmt.Sprintf("/images/%s/json", suite.jobStep.Source)
	suite.mux.HandleFunc(inspectPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "{\"ID\":\"xyz789\"}")
	})

	suite.mux.HandleFunc("/containers/create", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	err := suite.jse.Start(suite.job, stdIn, stdOutWriter, stdErrWriter)

	suite.EqualError(err, "API error (500): ")
}

func (suite *JobStepExecutorTestSuite) TestStart_StartError() {
	id := "foo123abc"
	stdIn := &bytes.Buffer{}
	_, stdOutWriter := io.Pipe()
	_, stdErrWriter := io.Pipe()

	inspectPath := fmt.Sprintf("/images/%s/json", suite.jobStep.Source)
	suite.mux.HandleFunc(inspectPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "{\"ID\":\"xyz789\"}")
	})

	suite.mux.HandleFunc("/containers/create", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, "{\"ID\":\""+id+"\"}")
	})

	attachPath := fmt.Sprintf("/containers/%s/attach", id)
	suite.mux.HandleFunc(attachPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte{1, 0, 0, 0, 0, 0, 0, 5})
		w.Write([]byte("hello"))
	})

	startPath := fmt.Sprintf("/containers/%s/start", id)
	suite.mux.HandleFunc(startPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	err := suite.jse.Start(suite.job, stdIn, stdOutWriter, stdErrWriter)

	suite.EqualError(err, "API error (400): ")
}

func (suite *JobStepExecutorTestSuite) TestStart_MissingImage() {
	pullCalled := false
	id := "foo123abc"
	stdIn := &bytes.Buffer{}
	_, stdOutWriter := io.Pipe()
	_, stdErrWriter := io.Pipe()

	inspectPath := fmt.Sprintf("/images/%s/json", suite.jobStep.Source)
	suite.mux.HandleFunc(inspectPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	suite.mux.HandleFunc("/images/create", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		pullCalled = true
	})

	suite.mux.HandleFunc("/containers/create", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, "{\"ID\":\""+id+"\"}")
	})

	attachPath := fmt.Sprintf("/containers/%s/attach", id)
	suite.mux.HandleFunc(attachPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte{1, 0, 0, 0, 0, 0, 0, 5})
		w.Write([]byte("hello"))
	})

	startPath := fmt.Sprintf("/containers/%s/start", id)
	suite.mux.HandleFunc(startPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	err := suite.jse.Start(suite.job, stdIn, stdOutWriter, stdErrWriter)

	suite.NoError(err)
	suite.True(pullCalled, "Docker image not pulled")
}

func (suite *JobStepExecutorTestSuite) TestStart_PullError() {
	pullCalled := false
	stdIn := &bytes.Buffer{}
	_, stdOutWriter := io.Pipe()
	_, stdErrWriter := io.Pipe()

	inspectPath := fmt.Sprintf("/images/%s/json", suite.jobStep.Source)
	suite.mux.HandleFunc(inspectPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	suite.mux.HandleFunc("/images/create", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		pullCalled = true
	})

	err := suite.jse.Start(suite.job, stdIn, stdOutWriter, stdErrWriter)

	suite.EqualError(err, "API error (404): ")
	suite.True(pullCalled, "Docker image not pulled")
}

func (suite *JobStepExecutorTestSuite) TestStart_ForceRefresh() {
	suite.job.currentStep().Refresh = true
	inspectCalled := false
	pullCalled := false
	removeCalled := false
	imageID := "xyz890"
	id := "foo123abc"
	stdIn := &bytes.Buffer{}
	_, stdOutWriter := io.Pipe()
	_, stdErrWriter := io.Pipe()

	inspectPath := fmt.Sprintf("/images/%s/json", suite.jobStep.Source)
	suite.mux.HandleFunc(inspectPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if inspectCalled {
			fmt.Fprintf(w, "{\"Id\":\"%s\"}", "15930e")
		} else {
			fmt.Fprintf(w, "{\"Id\":\"%s\"}", imageID)
		}
		inspectCalled = true
	})

	suite.mux.HandleFunc("/images/create", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		pullCalled = true
	})

	removePath := fmt.Sprintf("/images/%s", imageID)
	suite.mux.HandleFunc(removePath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		removeCalled = true
	})

	suite.mux.HandleFunc("/containers/create", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "{\"ID\":\"%s\"}", id)
	})

	attachPath := fmt.Sprintf("/containers/%s/attach", id)
	suite.mux.HandleFunc(attachPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte{1, 0, 0, 0, 0, 0, 0, 5})
		w.Write([]byte("hello"))
	})

	startPath := fmt.Sprintf("/containers/%s/start", id)
	suite.mux.HandleFunc(startPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	err := suite.jse.Start(suite.job, stdIn, stdOutWriter, stdErrWriter)

	suite.NoError(err)
	suite.True(pullCalled, "Docker image not pulled")
	suite.True(removeCalled, "Docker image not removed")
}

func (suite *JobStepExecutorTestSuite) TestInspect_Success() {
	path := fmt.Sprintf("/containers/%s/json", suite.jobStep.id)
	suite.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		suite.Equal("GET", r.Method)
		fmt.Fprint(w, "{\"State\":{\"ExitCode\":0}}")
	})

	err := suite.jse.Inspect(suite.job)

	suite.NoError(err)
}

func (suite *JobStepExecutorTestSuite) TestInspect_Error() {
	path := fmt.Sprintf("/containers/%s/json", suite.jobStep.id)
	suite.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		suite.Equal("GET", r.Method)
		w.WriteHeader(http.StatusNotFound)
	})

	err := suite.jse.Inspect(suite.job)

	suite.EqualError(err, "No such container: abc123")
}

func (suite *JobStepExecutorTestSuite) TestInspect_ErrorExit() {
	path := fmt.Sprintf("/containers/%s/json", suite.jobStep.id)
	suite.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		suite.Equal("GET", r.Method)
		fmt.Fprint(w, "{\"State\":{\"ExitCode\":99}}")
	})

	err := suite.jse.Inspect(suite.job)

	suite.EqualError(err, "Container exit code: 99")
}

func (suite *JobStepExecutorTestSuite) TestCleanUp_Success() {
	path := fmt.Sprintf("/containers/%s", suite.jobStep.id)
	suite.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		suite.Equal("DELETE", r.Method)
	})

	err := suite.jse.CleanUp(suite.job)

	suite.NoError(err)
}

func (suite *JobStepExecutorTestSuite) TestCleanUp_Error() {
	path := fmt.Sprintf("/containers/%s", suite.jobStep.id)
	suite.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		suite.Equal("DELETE", r.Method)
		w.WriteHeader(http.StatusNotFound)
	})

	err := suite.jse.CleanUp(suite.job)

	suite.EqualError(err, "No such container: abc123")
}

func TestJobStepExecutor(t *testing.T) {
	suite.Run(t, new(JobStepExecutorTestSuite))
}
