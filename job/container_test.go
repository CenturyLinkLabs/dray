package job

import (
	"io"

	"github.com/stretchr/testify/mock"
)

type mockContainerFactory struct {
	mock.Mock
}

func (m *mockContainerFactory) NewContainer(source string, env []string) Container {
	args := m.Mock.Called(source, env)
	return args.Get(0).(Container)
}

type mockContainer struct {
	mock.Mock
	output string
}

func (m *mockContainer) Create() error {
	args := m.Mock.Called()
	return args.Error(0)
}

func (m *mockContainer) Attach(stdIn io.Reader, stdOut, stdErr io.Writer) error {

	if len(m.output) > 0 {
		stdOut.Write([]byte(m.output))
		stdOut.Write([]byte{'\n'})
	}
	args := m.Mock.Called(stdIn, stdOut, stdErr)
	return args.Error(0)
}

func (m *mockContainer) Start() error {
	args := m.Mock.Called()
	return args.Error(0)
}

func (m *mockContainer) Inspect() error {
	args := m.Mock.Called()
	return args.Error(0)
}

func (m *mockContainer) Remove() error {
	args := m.Mock.Called()
	return args.Error(0)
}
