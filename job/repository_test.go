package job

import (
	"github.com/stretchr/testify/mock"
)

type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) All() ([]Job, error) {
	args := m.Mock.Called()
	return args.Get(0).([]Job), args.Error(1)
}

func (m *mockRepository) Get(jobID string) (*Job, error) {
	args := m.Mock.Called(jobID)
	return args.Get(0).(*Job), args.Error(1)
}

func (m *mockRepository) Create(job *Job) error {
	args := m.Mock.Called(job)
	return args.Error(0)
}

func (m *mockRepository) Delete(jobID string) error {
	args := m.Mock.Called(jobID)
	return args.Error(0)
}

func (m *mockRepository) Update(jobID, attr, value string) error {
	args := m.Mock.Called(jobID, attr, value)
	return args.Error(0)
}

func (m *mockRepository) GetJobLog(jobID string, index int) (*JobLog, error) {
	args := m.Mock.Called(jobID, index)
	return args.Get(0).(*JobLog), args.Error(1)
}

func (m *mockRepository) AppendLogLine(jobID, logLine string) error {
	args := m.Mock.Called(jobID, logLine)
	return args.Error(0)
}
