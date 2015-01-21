package job

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	job  *Job
	step *JobStep
	jm   *jobManager
	a    *mockAccessor
	cf   *mockContainerFactory
	c    *mockContainer
	err  error
)

func setUp() {
	step = &JobStep{
		Name:        "Step1",
		Source:      "foo/bar",
		Environment: []EnvVar{EnvVar{Variable: "y", Value: "2"}},
	}

	job = &Job{
		Name:        "foo",
		Environment: []EnvVar{EnvVar{Variable: "x", Value: "1"}},
		Steps:       []JobStep{*step},
	}

	a = &mockAccessor{}
	c = &mockContainer{}
	cf = &mockContainerFactory{}

	jm = &jobManager{accessor: a, containerFactory: cf}
	err = errors.New("oops")
}

func TestListAll(t *testing.T) {
	setUp()
	jobs := []Job{*job}

	a.On("All").Return(jobs, err)

	resultJobs, resultErr := jm.ListAll()

	assert.Equal(t, jobs, resultJobs)
	assert.Equal(t, err, resultErr)
	a.Mock.AssertExpectations(t)
}

func TestGetByID(t *testing.T) {
	setUp()
	id := "123"

	a.On("Get", id).Return(job, err)

	resultJob, resultErr := jm.GetByID(id)

	assert.Equal(t, job, resultJob)
	assert.Equal(t, err, resultErr)
	a.Mock.AssertExpectations(t)
}

func TestCreate(t *testing.T) {
	setUp()

	a.On("Create", job).Return(err)

	resultErr := jm.Create(job)

	assert.Equal(t, err, resultErr)
	a.Mock.AssertExpectations(t)
}

func TestDelete(t *testing.T) {
	a.On("Delete", job.ID).Return(err)

	resultErr := jm.Delete(job)

	assert.Equal(t, err, resultErr)
	a.Mock.AssertExpectations(t)
}

func TestGetLog(t *testing.T) {
	index := 3
	jobLog := &JobLog{Index: 3}

	a.On("GetJobLog", job.ID, index).Return(jobLog, err)

	resultLog, resultErr := jm.GetLog(job, index)

	assert.Equal(t, jobLog, resultLog)
	assert.Equal(t, err, resultErr)
	a.Mock.AssertExpectations(t)
}

func TestExecuteSuccess(t *testing.T) {
	setUp()

	c.On("Create").Return(nil)
	c.On("Attach",
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer")).Return(nil)
	c.On("Start").Return(nil)
	c.On("Inspect").Return(nil)
	c.On("Remove").Return(nil)

	cf.On("NewContainer", step.Source, []string{"y=2", "x=1"}).Return(c)

	a.On("Update", job.ID, "status", "running").Return(nil)
	a.On("Update", job.ID, "completedSteps", "1").Return(nil)
	a.On("Update", job.ID, "status", "complete").Return(nil)

	resultErr := jm.Execute(job)
	time.Sleep(time.Millisecond)

	assert.Nil(t, resultErr)
	cf.Mock.AssertExpectations(t)
	c.Mock.AssertExpectations(t)
	a.Mock.AssertExpectations(t)
}

func TestExecuteContainerCreateError(t *testing.T) {
	setUp()

	c.On("Create").Return(err)

	cf.On("NewContainer", step.Source, []string{"y=2", "x=1"}).Return(c)

	a.On("Update", job.ID, "status", "running").Return(nil)
	a.On("Update", job.ID, "status", "error").Return(nil)

	resultErr := jm.Execute(job)

	if assert.Error(t, resultErr) {
		assert.Equal(t, err, resultErr)
	}

	cf.Mock.AssertExpectations(t)
	c.Mock.AssertExpectations(t)
	a.Mock.AssertExpectations(t)
}

func TestExecuteContainerStartError(t *testing.T) {
	setUp()

	c.On("Create").Return(nil)
	c.On("Attach",
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer")).Return(nil)
	c.On("Start").Return(err)
	c.On("Remove").Return(nil)

	cf.On("NewContainer", step.Source, []string{"y=2", "x=1"}).Return(c)

	a.On("Update", job.ID, "status", "running").Return(nil)
	a.On("Update", job.ID, "status", "error").Return(nil)

	resultErr := jm.Execute(job)
	time.Sleep(time.Millisecond)

	if assert.Error(t, resultErr) {
		assert.Equal(t, err, resultErr)
	}

	cf.Mock.AssertExpectations(t)
	c.Mock.AssertExpectations(t)
	a.Mock.AssertExpectations(t)
}

func TestExecuteContainerInspectError(t *testing.T) {
	setUp()

	c.On("Create").Return(nil)
	c.On("Attach",
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer")).Return(nil)
	c.On("Start").Return(nil)
	c.On("Inspect").Return(err)
	c.On("Remove").Return(nil)

	cf.On("NewContainer", step.Source, []string{"y=2", "x=1"}).Return(c)

	a.On("Update", job.ID, "status", "running").Return(nil)
	a.On("Update", job.ID, "status", "error").Return(nil)

	resultErr := jm.Execute(job)

	if assert.Error(t, resultErr) {
		assert.Equal(t, err, resultErr)
	}

	cf.Mock.AssertExpectations(t)
	c.Mock.AssertExpectations(t)
	a.Mock.AssertExpectations(t)
}

func TestExecuteOutputLogging(t *testing.T) {
	setUp()

	output := "line of output"

	c = &mockContainer{output: output}
	c.On("Create").Return(nil)
	c.On("Attach",
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer")).Return(nil)
	c.On("Start").Return(nil)
	c.On("Inspect").Return(nil)
	c.On("Remove").Return(nil)

	cf.On("NewContainer", step.Source, []string{"y=2", "x=1"}).Return(c)

	a.On("Update", job.ID, "status", "running").Return(nil)
	a.On("Update", job.ID, "completedSteps", "1").Return(nil)
	a.On("AppendLogLine", job.ID, output).Return(nil)
	a.On("Update", job.ID, "status", "complete").Return(nil)

	resultErr := jm.Execute(job)

	assert.Nil(t, resultErr)
	cf.Mock.AssertExpectations(t)
	c.Mock.AssertExpectations(t)
	a.Mock.AssertExpectations(t)
}
