package job

import (
	"github.com/stretchr/testify/mock"
)

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
