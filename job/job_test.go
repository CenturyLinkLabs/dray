package job

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListAll(t *testing.T) {
	jobs := []Job{Job{Name: "foo"}}
	err := errors.New("oops")

	accessor = &testAccessor{
		allFunc: func() ([]Job, error) {
			return jobs, err
		},
	}

	resultJobs, resultErr := ListAll()
	assert.Equal(t, jobs, resultJobs)
	assert.Equal(t, err, resultErr)
}

func TestGetByID(t *testing.T) {
	id := "123"
	job := Job{Name: "foo"}
	err := errors.New("oops")

	accessor = &testAccessor{
		getFunc: func(jobID string) (*Job, error) {
			assert.Equal(t, id, jobID)
			return &job, err
		},
	}

	resultJob, resultErr := GetByID(id)
	assert.Equal(t, &job, resultJob)
	assert.Equal(t, err, resultErr)
}

func TestCreate(t *testing.T) {
	job := &Job{Name: "foo"}
	err := errors.New("oops")

	accessor = &testAccessor{
		createFunc: func(j *Job) error {
			assert.Equal(t, j, job)
			return err
		},
	}

	resultErr := job.Create()
	assert.Equal(t, err, resultErr)
}

func TestDelete(t *testing.T) {
	job := &Job{ID: "123"}
	err := errors.New("oops")

	accessor = &testAccessor{
		deleteFunc: func(jobID string) error {
			assert.Equal(t, job.ID, jobID)
			return err
		},
	}

	resultErr := job.Delete()
	assert.Equal(t, err, resultErr)
}

func TestGetLog(t *testing.T) {
	index := 3
	job := &Job{ID: "123"}
	jobLog := &JobLog{Index: 3}
	err := errors.New("oops")

	accessor = &testAccessor{
		getJobLogFunc: func(jobID string, logIndex int) (*JobLog, error) {
			assert.Equal(t, job.ID, jobID)
			assert.Equal(t, index, logIndex)
			return jobLog, err
		},
	}

	resultLog, resultErr := job.GetLog(index)
	assert.Equal(t, jobLog, resultLog)
	assert.Equal(t, err, resultErr)
}

func TestExecuteSuccess(t *testing.T) {
	jobStep := JobStep{
		Name:        "Step1",
		Source:      "foo/bar",
		Environment: []EnvVar{},
	}

	job := &Job{
		ID:    "123",
		Steps: []JobStep{jobStep},
	}

	container := &mockContainer{}
	container.On("Create").Return(nil)
	container.On("Attach",
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer")).Return(nil)
	container.On("Start").Return(nil)
	container.On("Remove").Return(nil)

	mockFactory := &mockContainerFactory{}
	mockFactory.On("NewContainer", jobStep.Source, []string{}).Return(container)
	containerFactory = mockFactory

	acc := &mockAccessor{}
	acc.On("CompleteStep", job.ID).Return(nil)
	accessor = acc

	resultErr := job.Execute()

	assert.Nil(t, resultErr)
	container.Mock.AssertExpectations(t)
}

func TestExecuteContainerCreateError(t *testing.T) {
	err := errors.New("oops")
	jobStep := JobStep{
		Name:        "Step1",
		Source:      "foo/bar",
		Environment: []EnvVar{},
	}

	job := &Job{
		Steps: []JobStep{jobStep},
	}

	container := &mockContainer{}
	container.On("Create").Return(err)

	mockFactory := &mockContainerFactory{}
	mockFactory.On("NewContainer", jobStep.Source, []string{}).Return(container)
	containerFactory = mockFactory

	resultErr := job.Execute()

	if assert.Error(t, resultErr) {
		assert.Equal(t, err, resultErr)
	}

	container.Mock.AssertExpectations(t)
}

func TestExecuteContainerStartError(t *testing.T) {
	err := errors.New("oops")
	jobStep := JobStep{
		Name:        "Step1",
		Source:      "foo/bar",
		Environment: []EnvVar{},
	}

	job := &Job{
		Steps: []JobStep{jobStep},
	}

	container := &mockContainer{}
	container.On("Create").Return(nil)
	container.On("Attach",
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer")).Return(nil)
	container.On("Start").Return(err)

	mockFactory := &mockContainerFactory{}
	mockFactory.On("NewContainer", jobStep.Source, []string{}).Return(container)
	containerFactory = mockFactory

	resultErr := job.Execute()

	if assert.Error(t, resultErr) {
		assert.Equal(t, err, resultErr)
	}

	time.Sleep(time.Millisecond)
	container.Mock.AssertExpectations(t)
}

func TestExecuteContainerDeleteError(t *testing.T) {
	err := errors.New("oops")
	jobStep := JobStep{
		Name:        "Step1",
		Source:      "foo/bar",
		Environment: []EnvVar{},
	}

	job := &Job{
		Steps: []JobStep{jobStep},
	}

	container := &mockContainer{}
	container.On("Create").Return(nil)
	container.On("Attach",
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer"),
		mock.AnythingOfType("*bytes.Buffer")).Return(nil)
	container.On("Start").Return(nil)
	container.On("Remove").Return(err)

	mockFactory := &mockContainerFactory{}
	mockFactory.On("NewContainer", jobStep.Source, []string{}).Return(container)
	containerFactory = mockFactory

	resultErr := job.Execute()

	if assert.Error(t, resultErr) {
		assert.Equal(t, err, resultErr)
	}

	container.Mock.AssertExpectations(t)
}
