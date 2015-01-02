package job

import (
	"github.com/stretchr/testify/mock"
)

type jobAccessorAllFunc func() ([]Job, error)
type jobAccessorGetFunc func(jobID string) (*Job, error)
type jobAccessorCreateFunc func(job *Job) error
type jobAccessorDeleteFunc func(jobID string) error
type jobAccessorCompleteStepFunc func(jobID string) error
type jobAccessorGetJobLogFunc func(jobid string, index int) (*JobLog, error)
type jobAccessorAppendLogLineFunc func(jobID, logLine string) error

type testAccessor struct {
	allFunc           jobAccessorAllFunc
	getFunc           jobAccessorGetFunc
	createFunc        jobAccessorCreateFunc
	deleteFunc        jobAccessorDeleteFunc
	completeStepFunc  jobAccessorCompleteStepFunc
	getJobLogFunc     jobAccessorGetJobLogFunc
	appendLogLineFunc jobAccessorAppendLogLineFunc
}

func (a *testAccessor) All() ([]Job, error) {
	return a.allFunc()
}

func (a *testAccessor) Get(jobID string) (*Job, error) {
	return a.getFunc(jobID)
}

func (a *testAccessor) Create(job *Job) error {
	return a.createFunc(job)
}

func (a *testAccessor) Delete(jobID string) error {
	return a.deleteFunc(jobID)
}

func (a *testAccessor) CompleteStep(jobID string) error {
	return a.completeStepFunc(jobID)
}

func (a *testAccessor) GetJobLog(jobID string, index int) (*JobLog, error) {
	return a.getJobLogFunc(jobID, index)
}

func (a *testAccessor) AppendLogLine(jobID, logLine string) error {
	return a.appendLogLineFunc(jobID, logLine)
}

type mockAccessor struct {
	mock.Mock
}

func (m *mockAccessor) All() ([]Job, error) {
	args := m.Mock.Called()
	return args.Get(0).([]Job), args.Error(1)
}

func (m *mockAccessor) Get(jobID string) (*Job, error) {
	args := m.Mock.Called(jobID)
	return args.Get(0).(*Job), args.Error(1)
}

func (m *mockAccessor) Create(job *Job) error {
	args := m.Mock.Called(job)
	return args.Error(0)
}

func (m *mockAccessor) Delete(jobID string) error {
	args := m.Mock.Called(jobID)
	return args.Error(0)
}

func (m *mockAccessor) CompleteStep(jobID string) error {
	args := m.Mock.Called(jobID)
	return args.Error(0)
}

func (m *mockAccessor) GetJobLog(jobID string, index int) (*JobLog, error) {
	args := m.Mock.Called(jobID, index)
	return args.Get(0).(*JobLog), args.Error(1)
}

func (m *mockAccessor) AppendLogLine(jobID, logLine string) error {
	args := m.Mock.Called(jobID, logLine)
	return args.Error(0)
}
