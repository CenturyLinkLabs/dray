package job

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
)

const (
	BeginDelimiter = "----BEGIN PANAMAX DATA----"
	EndDelimiter   = "----END PANAMAX DATA----"
	EOT            = byte('\u0003')
)

var (
	accessor         JobAccessor
	containerFactory ContainerFactory
)

func init() {
	accessor = &redisJobAccessor{}
	containerFactory = &dockerContainerFactory{}
}

type Job struct {
	ID             string    `json:"id,omitempty"`
	Name           string    `json:"name,omitempty"`
	Steps          []JobStep `json:"steps,omitempty"`
	Environment    []EnvVar  `json:"environment,omitempty"`
	StepsCompleted string    `json:"stepsCompleted,omitempty"`
	Status         string    `json:"status,omitempty"`
}

type JobStep struct {
	Name        string   `json:"name,omitempty"`
	Source      string   `json:"source,omitempty"`
	Environment []EnvVar `json:"environment,omitempty"`
}

type EnvVar struct {
	Variable string `json:"variable"`
	Value    string `json:"value"`
}

type JobLog struct {
	Index int      `json:"index,omitempty"`
	Lines []string `json:"lines"`
}

func ListAll() ([]Job, error) {
	return accessor.All()
}

func GetByID(jobID string) (*Job, error) {
	return accessor.Get(jobID)
}

func (job *Job) Create() error {
	return accessor.Create(job)
}

func (job *Job) Delete() error {
	return accessor.Delete(job.ID)
}

func (job *Job) GetLog(index int) (*JobLog, error) {
	return accessor.GetJobLog(job.ID, index)
}

func (job *Job) Execute() error {
	var err error
	status := "running"
	buffer := &bytes.Buffer{}
	capture := io.Reader(buffer)

	accessor.Update(job.ID, "status", status)

	for i := range job.Steps {
		capture, err = job.executeStep(i, capture)

		if err != nil {
			break
		}
		accessor.Update(job.ID, "completedSteps", strconv.Itoa(i+1))
	}

	if err != nil {
		status = "error"
	} else {
		status = "complete"
	}

	accessor.Update(job.ID, "status", status)
	return err
}

func (job *Job) executeStep(stepIndex int, stdIn io.Reader) (io.Reader, error) {
	stdOut := &bytes.Buffer{}
	stdErr := &bytes.Buffer{}
	step := job.Steps[stepIndex]

	// Each step gets its own environment, plus the job-level environment
	step.Environment = append(step.Environment, job.Environment...)
	container := containerFactory.NewContainer(step.Source, stringifyEnvironment(step.Environment))

	if err := container.Create(); err != nil {
		return nil, err
	}
	log.Debugf("Container %s created", container)

	defer func() {
		var msg string

		if err := container.Remove(); err != nil {
			msg = fmt.Sprintf("Container %s NOT removed", container)
		} else {
			msg = fmt.Sprintf("Container %s removed", container)
		}
		log.Debug(msg)
	}()

	go func() {
		container.Attach(stdIn, stdOut, stdErr)
		stdOut.Write([]byte{EOT, '\n'})
	}()

	if err := container.Start(); err != nil {
		return nil, err
	}
	log.Debugf("Container %s started", container)

	output, err := job.captureOutput(stdOut)
	if err != nil {
		return nil, err
	}
	log.Debugf("Container %s stopped", container)

	if err := container.Inspect(); err != nil {
		return nil, err
	}

	return output, nil
}

func (job *Job) captureOutput(r io.Reader) (io.Reader, error) {
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
			accessor.AppendLogLine(job.ID, s)

			if s == EndDelimiter {
				capture = false
			}

			if capture {
				buffer.WriteString(s + "\n")
			}

			if s == BeginDelimiter {
				capture = true
			}
		}
	}

	return buffer, nil
}

func stringifyEnvironment(env []EnvVar) []string {
	envStrings := []string{}

	for _, v := range env {
		s := fmt.Sprintf("%s=%s", v.Variable, v.Value)
		envStrings = append(envStrings, s)
	}

	return envStrings
}
