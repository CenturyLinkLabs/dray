package job

import (
	"fmt"
	"io"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

type jobStepExecutor struct {
	client *docker.Client
}

func NewExecutor(dockerEndpoint string) JobStepExecutor {
	client, err := docker.NewClient(dockerEndpoint)
	if err != nil {
		log.Errorf("Error instantiating Docker client: %s", err)
		panic(err)
	}

	return &jobStepExecutor{client: client}
}

func (e *jobStepExecutor) Start(j *Job, stdIn io.Reader, stdOut, stdErr io.WriteCloser) error {
	// Create container
	id, err := e.createContainer(j)
	if err != nil {
		return err
	}

	// Attach to container
	go func() {
		defer stdOut.Close()
		defer stdErr.Close()
		e.attachContainer(id, stdIn, stdOut, stdErr)
		log.Debugf("Container %s stopped", id)
	}()

	// Start container execution
	if err := e.startContainer(id); err != nil {
		return err
	}

	j.CurrentStep().ID = id
	return nil
}

func (e *jobStepExecutor) Inspect(j *Job) error {
	container, err := e.client.InspectContainer(j.CurrentStep().ID)

	if err != nil {
		return err
	}

	if container.State.ExitCode != 0 {
		return fmt.Errorf("Container exit code: %d", container.State.ExitCode)
	}

	return nil
}

func (e *jobStepExecutor) CleanUp(j *Job) error {
	removeOpts := docker.RemoveContainerOptions{
		ID: j.CurrentStep().ID,
	}

	err := e.client.RemoveContainer(removeOpts)

	if err == nil {
		log.Infof("Container %s removed", j.CurrentStep().ID)
	}

	return err
}

func (e *jobStepExecutor) createContainer(j *Job) (string, error) {
	step := j.CurrentStep()
	if err := e.ensureImage(step.Source); err != nil {
		return "", err
	}

	opts := docker.CreateContainerOptions{
		Config: &docker.Config{
			Image:     step.Source,
			Env:       j.CurrentStepEnvironment().Stringify(),
			OpenStdin: true,
			StdinOnce: true,
		},
	}

	if step.UsesFilePipe() {
		opts.HostConfig = &docker.HostConfig{
			Binds: []string{fmt.Sprintf("%s:%s", step.FilePipePath(), step.Output)},
		}
	}

	container, err := e.client.CreateContainer(opts)

	if err == nil {
		log.Infof("Container %s created from %s", container.ID, step.Source)
	}

	return container.ID, err
}

func (e *jobStepExecutor) attachContainer(id string, stdIn io.Reader, stdOut, stdErr io.Writer) error {
	attachOpts := docker.AttachToContainerOptions{
		Container:    id,
		InputStream:  stdIn,
		OutputStream: stdOut,
		ErrorStream:  stdErr,
		Stream:       true,
		Stdin:        true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  false,
	}

	return e.client.AttachToContainer(attachOpts)
}

func (e *jobStepExecutor) startContainer(id string) error {
	err := e.client.StartContainer(id, nil)

	if err == nil {
		log.Infof("Container %s started", id)
	}

	return err
}

func (e *jobStepExecutor) ensureImage(source string) error {
	_, err := e.client.InspectImage(source)
	if err == docker.ErrNoSuchImage {

		log.Infof("Pulling image %s", source)
		if err = e.pullImage(source); err != nil {
			return err
		}
	}

	return err
}

func (e *jobStepExecutor) pullImage(source string) error {
	opts := docker.PullImageOptions{
		Repository: source,
	}

	return e.client.PullImage(opts, docker.AuthConfiguration{})
}
