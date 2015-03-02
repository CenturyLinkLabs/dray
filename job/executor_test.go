package job

import (
	"io"

	"github.com/stretchr/testify/mock"
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
