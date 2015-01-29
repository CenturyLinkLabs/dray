package job

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type JobTestSuite struct {
	suite.Suite

	job  *Job
	step *JobStep
	jm   *jobManager
	a    *mockAccessor
	cf   *mockContainerFactory
	c    *mockContainer
	err  error
}

func (suite *JobTestSuite) SetupTest() {
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

	suite.a = &mockAccessor{}
	suite.c = &mockContainer{}
	suite.cf = &mockContainerFactory{}

	suite.jm = &jobManager{accessor: suite.a, containerFactory: suite.cf}
	suite.err = errors.New("oops")
}

func (suite *JobTestSuite) TearDownTest() {
	suite.cf.Mock.AssertExpectations(suite.T())
	suite.c.Mock.AssertExpectations(suite.T())
	suite.a.Mock.AssertExpectations(suite.T())
}

func (suite *JobTestSuite) TestListAll() {
	jobs := []Job{*suite.job}

	suite.a.On("All").Return(jobs, suite.err)

	resultJobs, resultErr := suite.jm.ListAll()

	suite.Equal(jobs, resultJobs)
	suite.Equal(suite.err, resultErr)
}

func (suite *JobTestSuite) TestGetByID() {
	id := "123"

	suite.a.On("Get", id).Return(suite.job, suite.err)

	resultJob, resultErr := suite.jm.GetByID(id)

	suite.Equal(suite.job, resultJob)
	suite.Equal(suite.err, resultErr)
}

func (suite *JobTestSuite) TestCreate() {
	suite.a.On("Create", suite.job).Return(suite.err)

	resultErr := suite.jm.Create(suite.job)

	suite.Equal(suite.err, resultErr)
}

func (suite *JobTestSuite) TestDelete() {
	suite.a.On("Delete", suite.job.ID).Return(suite.err)

	resultErr := suite.jm.Delete(suite.job)

	suite.Equal(suite.err, resultErr)
}

func (suite *JobTestSuite) TestGetLog() {
	index := 3
	jobLog := &JobLog{Index: 3}

	suite.a.On("GetJobLog", suite.job.ID, index).Return(jobLog, suite.err)

	resultLog, resultErr := suite.jm.GetLog(suite.job, index)

	suite.Equal(jobLog, resultLog)
	suite.Equal(suite.err, resultErr)
}

func (suite *JobTestSuite) TestExecuteSuccess() {
	suite.c.On("Create").Return(nil)
	suite.c.On("Attach",
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer")).Return(nil)
	suite.c.On("Start").Return(nil)
	suite.c.On("Inspect").Return(nil)
	suite.c.On("Remove").Return(nil)

	suite.cf.On("NewContainer", suite.step.Source, []string{"y=2", "x=1"}).Return(suite.c)

	suite.a.On("Update", suite.job.ID, "status", "running").Return(nil)
	suite.a.On("Update", suite.job.ID, "completedSteps", "1").Return(nil)
	suite.a.On("Update", suite.job.ID, "status", "complete").Return(nil)

	resultErr := suite.jm.Execute(suite.job)
	time.Sleep(time.Millisecond)

	suite.Nil(resultErr)
}

func (suite *JobTestSuite) TestExecuteContainerCreateError() {
	suite.c.On("Create").Return(suite.err)

	suite.cf.On("NewContainer", suite.step.Source, []string{"y=2", "x=1"}).Return(suite.c)

	suite.a.On("Update", suite.job.ID, "status", "running").Return(nil)
	suite.a.On("Update", suite.job.ID, "status", "error").Return(nil)

	resultErr := suite.jm.Execute(suite.job)

	if suite.Error(resultErr) {
		suite.Equal(suite.err, resultErr)
	}
}

func (suite *JobTestSuite) TestExecuteContainerStartError() {
	suite.c.On("Create").Return(nil)
	suite.c.On("Attach",
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer")).Return(nil)
	suite.c.On("Start").Return(suite.err)
	suite.c.On("Remove").Return(nil)

	suite.cf.On("NewContainer", suite.step.Source, []string{"y=2", "x=1"}).Return(suite.c)

	suite.a.On("Update", suite.job.ID, "status", "running").Return(nil)
	suite.a.On("Update", suite.job.ID, "status", "error").Return(nil)

	resultErr := suite.jm.Execute(suite.job)
	time.Sleep(time.Millisecond)

	if suite.Error(resultErr) {
		suite.Equal(suite.err, resultErr)
	}
}

func (suite *JobTestSuite) TestExecuteContainerInspectError() {
	suite.c.On("Create").Return(nil)
	suite.c.On("Attach",
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer")).Return(nil)
	suite.c.On("Start").Return(nil)
	suite.c.On("Inspect").Return(suite.err)
	suite.c.On("Remove").Return(nil)

	suite.cf.On("NewContainer", suite.step.Source, []string{"y=2", "x=1"}).Return(suite.c)

	suite.a.On("Update", suite.job.ID, "status", "running").Return(nil)
	suite.a.On("Update", suite.job.ID, "status", "error").Return(nil)

	resultErr := suite.jm.Execute(suite.job)

	if suite.Error(resultErr) {
		suite.Equal(suite.err, resultErr)
	}
}

func (suite *JobTestSuite) TestExecuteOutputLogging() {
	output := "line of output"

	c := &mockContainer{output: output}
	c.On("Create").Return(nil)
	c.On("Attach",
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer")).Return(nil)
	c.On("Start").Return(nil)
	c.On("Inspect").Return(nil)
	c.On("Remove").Return(nil)

	suite.cf.On("NewContainer", suite.step.Source, []string{"y=2", "x=1"}).Return(c)

	suite.a.On("Update", suite.job.ID, "status", "running").Return(nil)
	suite.a.On("Update", suite.job.ID, "completedSteps", "1").Return(nil)
	suite.a.On("AppendLogLine", suite.job.ID, output).Return(nil)
	suite.a.On("Update", suite.job.ID, "status", "complete").Return(nil)

	resultErr := suite.jm.Execute(suite.job)

	suite.Nil(resultErr)
}

func TestJobTestSuite(t *testing.T) {
	suite.Run(t, new(JobTestSuite))
}
