package job

import (
	"fmt"
	"io"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

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

	client *docker.Client
}

type dockerContainerFactory struct {
	client *docker.Client
}

func NewContainerFactory(dockerEndpoint string) ContainerFactory {
	client, err := docker.NewClient(dockerEndpoint)
	if err != nil {
		log.Errorf("Error instantiating Docker client: %s", err)
		panic(err)
	}

	return &dockerContainerFactory{client: client}
}

func (cf *dockerContainerFactory) NewContainer(source string, env []string) Container {
	return &dockerContainer{Source: source, Env: env, client: cf.client}
}

func (c *dockerContainer) Create() error {
	if err := c.ensureImage(); err != nil {
		return err
	}

	opts := docker.CreateContainerOptions{
		Config: &docker.Config{
			Image:     c.Source,
			Env:       c.Env,
			OpenStdin: true,
			StdinOnce: true,
		},
	}

	container, err := c.client.CreateContainer(opts)

	if err == nil {
		c.ID = container.ID
		log.Infof("Container %s created from %s", c, c.Source)
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

	return c.client.AttachToContainer(attachOpts)
}

func (c *dockerContainer) Start() error {
	err := c.client.StartContainer(c.ID, nil)

	if err == nil {
		log.Infof("Container %s started", c)
	}

	return err
}

func (c *dockerContainer) Inspect() error {
	container, err := c.client.InspectContainer(c.ID)

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

	err := c.client.RemoveContainer(removeOpts)

	if err == nil {
		log.Infof("Container %s removed", c)
	}

	return err
}

func (c *dockerContainer) ensureImage() error {
	_, err := c.client.InspectImage(c.Source)
	if err == docker.ErrNoSuchImage {

		log.Infof("Pulling image %s", c.Source)
		if err = c.pullImage(); err != nil {
			return err
		}
	}

	return err
}

func (c *dockerContainer) pullImage() error {
	opts := docker.PullImageOptions{
		Repository: c.Source,
	}

	return c.client.PullImage(opts, docker.AuthConfiguration{})
}

func (c *dockerContainer) String() string {
	return c.ID[0:12]
}
