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

type containerCommandFunc func() error
type containerAttachFunc func(stdIn io.Reader, stdOut, stdErr io.Writer) error

type testContainer struct {
	createFunc containerCommandFunc
	attachFunc containerAttachFunc
	startFunc  containerCommandFunc
	removeFunc containerCommandFunc
}

func (c *testContainer) Create() error {
	return c.createFunc()
}

func (c *testContainer) Attach(stdIn io.Reader, stdOut, stdErr io.Writer) error {
	return c.attachFunc(stdIn, stdOut, stdErr)
}

func (c *testContainer) Start() error {
	return c.startFunc()
}

func (c *testContainer) Remove() error {
	return c.removeFunc()
}

type mockContainer struct {
	mock.Mock
}

func (m *mockContainer) Create() error {
	args := m.Mock.Called()
	return args.Error(0)
}

func (m *mockContainer) Attach(stdIn io.Reader, stdOut, stdErr io.Writer) error {
	args := m.Mock.Called(stdIn, stdOut, stdErr)
	return args.Error(0)
}

func (m *mockContainer) Start() error {
	args := m.Mock.Called()
	return args.Error(0)
}

func (m *mockContainer) Remove() error {
	args := m.Mock.Called()
	return args.Error(0)
}
