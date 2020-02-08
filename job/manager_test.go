package job

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type JobManagerTestSuite struct {
	suite.Suite

	job  *Job
	step *JobStep
	jm   *jobManager
	r    *mockRepository
	e    *mockExecutor
	err  error
}

func (suite *JobManagerTestSuite) SetupTest() {
	suite.step = &JobStep{
		Name:        "Step1",
		Source:      "foo/bar",
		Environment: []EnvVar{{Variable: "y", Value: "2"}},
	}

	suite.job = &Job{
		ID:          "123",
		Name:        "foo",
		Environment: []EnvVar{{Variable: "x", Value: "1"}},
		Steps:       []JobStep{*suite.step},
	}

	suite.r = &mockRepository{}
	suite.e = &mockExecutor{}

	suite.jm = &jobManager{repository: suite.r, executor: suite.e}
	suite.err = errors.New("oops")
}

func (suite *JobManagerTestSuite) TearDownTest() {
	suite.r.Mock.AssertExpectations(suite.T())
	suite.e.Mock.AssertExpectations(suite.T())
}

func (suite *JobManagerTestSuite) TestListAll() {
	jobs := []Job{*suite.job}

	suite.r.On("All").Return(jobs, suite.err)

	resultJobs, resultErr := suite.jm.ListAll()

	suite.Equal(jobs, resultJobs)
	suite.Equal(suite.err, resultErr)
}

func (suite *JobManagerTestSuite) TestGetByID() {
	id := "123"

	suite.r.On("Get", id).Return(suite.job, suite.err)

	resultJob, resultErr := suite.jm.GetByID(id)

	suite.Equal(suite.job, resultJob)
	suite.Equal(suite.err, resultErr)
}

func (suite *JobManagerTestSuite) TestCreate() {
	suite.r.On("Create", suite.job).Return(suite.err)

	resultErr := suite.jm.Create(suite.job)

	suite.Equal(suite.err, resultErr)
}

func (suite *JobManagerTestSuite) TestDelete() {
	suite.r.On("Delete", suite.job.ID).Return(suite.err)

	resultErr := suite.jm.Delete(suite.job)

	suite.Equal(suite.err, resultErr)
}

func (suite *JobManagerTestSuite) TestGetLog() {
	index := 3
	jobLog := &JobLog{Index: 3}

	suite.r.On("GetJobLog", suite.job.ID, index).Return(jobLog, suite.err)

	resultLog, resultErr := suite.jm.GetLog(suite.job, index)

	suite.Equal(jobLog, resultLog)
	suite.Equal(suite.err, resultErr)
}

func (suite *JobManagerTestSuite) TestExecuteSuccess() {
	suite.e.On("Start", suite.job, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	suite.e.On("Inspect", suite.job).Return(nil)
	suite.e.On("CleanUp", suite.job).Return(nil)

	suite.r.On("Update", suite.job.ID, "status", "running").Return(nil)
	suite.r.On("Update", suite.job.ID, "completedSteps", "1").Return(nil)
	suite.r.On("Update", suite.job.ID, "status", "complete").Return(nil)
	suite.r.On("Update", suite.job.ID, "createdAt", mock.Anything).Return(nil)
	suite.r.On("Update", suite.job.ID, "finishedIn", mock.Anything).Return(nil)

	resultErr := suite.jm.Execute(suite.job)

	suite.Nil(resultErr)
}

func (suite *JobManagerTestSuite) TestExecuteExecutorStartError() {
	suite.e.On("Start", suite.job, mock.Anything, mock.Anything, mock.Anything).Return(suite.err)

	suite.r.On("Update", suite.job.ID, "status", "running").Return(nil)
	suite.r.On("Update", suite.job.ID, "status", "error").Return(nil)
	suite.r.On("Update", suite.job.ID, "createdAt", mock.Anything).Return(nil)
	suite.r.On("Update", suite.job.ID, "finishedIn", mock.Anything).Return(nil)

	resultErr := suite.jm.Execute(suite.job)

	if suite.Error(resultErr) {
		suite.Equal(suite.err, resultErr)
	}
}

func (suite *JobManagerTestSuite) TestExecuteContainerInspectError() {
	suite.e.On("Start", suite.job, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	suite.e.On("Inspect", suite.job).Return(suite.err)
	suite.e.On("CleanUp", suite.job).Return(nil)

	suite.r.On("Update", suite.job.ID, "status", "running").Return(nil)
	suite.r.On("Update", suite.job.ID, "status", "error").Return(nil)
	suite.r.On("Update", suite.job.ID, "createdAt", mock.Anything).Return(nil)
	suite.r.On("Update", suite.job.ID, "finishedIn", mock.Anything).Return(nil)

	resultErr := suite.jm.Execute(suite.job)

	if suite.Error(resultErr) {
		suite.Equal(suite.err, resultErr)
	}
}

func (suite *JobManagerTestSuite) TestExecuteOutputLogging() {
	suite.e.output = "line of output"

	suite.e.On("Start", suite.job, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	suite.e.On("Inspect", suite.job).Return(nil)
	suite.e.On("CleanUp", suite.job).Return(nil)

	suite.r.On("Update", suite.job.ID, "status", "running").Return(nil)
	suite.r.On("Update", suite.job.ID, "completedSteps", "1").Return(nil)
	suite.r.On("AppendLogLine", suite.job.ID, suite.e.output).Return(nil)
	suite.r.On("Update", suite.job.ID, "status", "complete").Return(nil)
	suite.r.On("Update", suite.job.ID, "createdAt", mock.Anything).Return(nil)
	suite.r.On("Update", suite.job.ID, "finishedIn", mock.Anything).Return(nil)

	resultErr := suite.jm.Execute(suite.job)

	suite.Nil(resultErr)
}

func TestJobManagerTestSuite(t *testing.T) {
	suite.Run(t, new(JobManagerTestSuite))
}
