package job

import (
	"fmt"
	"io"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

const (
	DockerEndpoint = "tcp://localhost:2375"
)

var (
	dockerClient *docker.Client
)

func init() {
	client, err := docker.NewClient(DockerEndpoint)
	if err != nil {
		log.Errorf("Error instantiating Docker client: %s", err)
		panic(err)
	}
	dockerClient = client
}

type Container interface {
	Create() error
	Attach(stdIn io.Reader, stdOut, stdErr io.Writer) error
	Start() error
	Inspect() error
	Remove() error
}

type ContainerFactory interface {
	NewContainer(source string, env []string) Container
}

type dockerContainer struct {
	ID     string
	Source string
	Env    []string
}

type dockerContainerFactory struct {
}

func (*dockerContainerFactory) NewContainer(source string, env []string) Container {
	return &dockerContainer{Source: source, Env: env}
}

func (c *dockerContainer) Create() error {
	opts := docker.CreateContainerOptions{
		Config: &docker.Config{
			Image:     c.Source,
			Env:       c.Env,
			OpenStdin: true,
			StdinOnce: true,
		},
	}

	container, err := dockerClient.CreateContainer(opts)

	if err == nil {
		c.ID = container.ID
	}

	return err
}

func (c *dockerContainer) Attach(stdIn io.Reader, stdOut, stdErr io.Writer) error {
	attachOpts := docker.AttachToContainerOptions{
		Container:    c.ID,
		InputStream:  stdIn,
		OutputStream: stdOut,
		ErrorStream:  stdErr,
		Stream:       true,
		Stdin:        true,
		Stdout:       true,
		Stderr:       true,
		//RawTerminal:  true,
	}

	return dockerClient.AttachToContainer(attachOpts)
}

func (c *dockerContainer) Start() error {
	return dockerClient.StartContainer(c.ID, nil)
}

func (c *dockerContainer) Inspect() error {
	container, err := dockerClient.InspectContainer(c.ID)

	if err != nil {
		return err
	}

	if container.State.ExitCode != 0 {
		return fmt.Errorf("Container exit code: %d", container.State.ExitCode)
	}

	return nil
}

func (c *dockerContainer) Remove() error {
	removeOpts := docker.RemoveContainerOptions{
		ID: c.ID,
	}

	return dockerClient.RemoveContainer(removeOpts)
}

func (c *dockerContainer) String() string {
	return c.ID[0:12]
}
