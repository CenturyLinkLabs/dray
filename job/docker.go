package job

import (
	"io"
	"os"

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
		log.Errorf("error:", err)
		os.Exit(1)
	}
	dockerClient = client
}

func createContainer(image string) (*docker.Container, error) {
	opts := docker.CreateContainerOptions{
		Config: &docker.Config{
			Image:     image,
			OpenStdin: true,
			StdinOnce: true,
		},
	}

	return dockerClient.CreateContainer(opts)
}

func attachContainer(containerID string, stdIn io.Reader, stdOut io.Writer, stdErr io.Writer) error {
	attachOpts := docker.AttachToContainerOptions{
		Container:    containerID,
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

func startContainer(containerID string) error {
	return dockerClient.StartContainer(containerID, nil)
}

func removeContainer(containerID string) error {
	removeOpts := docker.RemoveContainerOptions{
		ID: containerID,
	}

	return dockerClient.RemoveContainer(removeOpts)
}
