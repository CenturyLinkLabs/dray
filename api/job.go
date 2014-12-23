package api

import (
	"bufio"
	"bytes"
	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"io"
	"strings"
)

const (
	DockerEndpoint = "tcp://localhost:2375"
	BeginDelimiter = "----BEGIN PANAMAX DATA----"
	EndDelimiter   = "----END PANAMAX DATA----"
	EOT            = byte('\u0003')
)

var (
	dockerClient *docker.Client
)

func init() {
	client, err := docker.NewClient(DockerEndpoint)
	handleErr(err)
	dockerClient = client
}

type Job struct {
	ID    string    `json:"id,omitempty"`
	Name  string    `json:"name,omitempty"`
	Steps []JobStep `json:"steps,omitempty"`
}

type JobStep struct {
	Name   string `json:"name,omitempty"`
	Source string `json:"source,omitempty"`
}

func ExecuteJob(job *Job) error {
	var capture io.Reader

	for _, step := range job.Steps {
		capture, _ = executeJobStep(job.ID, &step, capture)
	}
	return nil
}

func executeJobStep(jobID string, step *JobStep, stdIn io.Reader) (io.Reader, error) {

	stdOut := &bytes.Buffer{}
	stdErr := &bytes.Buffer{}

	container, err := createContainer(step.Source)
	if err != nil {
		return nil, err
	}
	log.Debugf("Container %s created", container.ID[0:12])

	go func() {
		err = attachContainer(container.ID, stdIn, stdOut, stdErr)
		handleErr(err)
		stdOut.Write([]byte{EOT, '\n'})
	}()

	err = startContainer(container.ID)
	if err != nil {
		return nil, err
	}
	log.Debugf("Container %s started", container.ID[0:12])

	key := "job:" + jobID + ":log"
	output, err := captureContainerOutput(stdOut, key)
	log.Debugf("Container %s stopped", container.ID[0:12])

	removeContainer(container.ID)
	if err != nil {
		return nil, err
	}
	log.Debugf("Container %s removed", container.ID[0:12])

	return output, nil
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

func captureContainerOutput(r io.Reader, key string) (io.Reader, error) {
	redis, _ := redisPool.Get()
	defer redisPool.Put(redis)

	reader := bufio.NewReader(r)
	buffer := &bytes.Buffer{}
	capture := false

	for {
		line, _ := reader.ReadBytes('\n')

		if len(line) > 0 {
			if line[0] == EOT {
				break
			}
			s := strings.TrimSpace(string(line))
			log.Debugf(s)
			redis.Cmd("rpush", key, s)

			if s == EndDelimiter {
				capture = false
			}

			if capture {
				buffer.WriteString(s)
				buffer.WriteString("\n")
			}

			if s == BeginDelimiter {
				capture = true
			}
		}
	}

	return buffer, nil
}
