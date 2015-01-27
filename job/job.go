package job // import "github.com/CenturyLinkLabs/dray/job"

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

type JobManager interface {
	ListAll() ([]Job, error)
	GetByID(string) (*Job, error)
	Create(*Job) error
	Execute(*Job) error
	GetLog(*Job, int) (*JobLog, error)
	Delete(*Job) error
}

type jobManager struct {
	accessor         JobAccessor
	containerFactory ContainerFactory
}

func NewJobManager(a JobAccessor, cf ContainerFactory) JobManager {
	return &jobManager{
		accessor:         a,
		containerFactory: cf,
	}
}

func (jm *jobManager) ListAll() ([]Job, error) {
	return jm.accessor.All()
}

func (jm *jobManager) GetByID(jobID string) (*Job, error) {
	return jm.accessor.Get(jobID)
}

func (jm *jobManager) Create(job *Job) error {
	return jm.accessor.Create(job)
}

func (jm *jobManager) Execute(job *Job) error {
	var err error
	status := "running"
	buffer := &bytes.Buffer{}
	capture := io.Reader(buffer)

	jm.accessor.Update(job.ID, "status", status)

	for i := range job.Steps {
		capture, err = jm.executeStep(job, i, capture)

		if err != nil {
			break
		}
		jm.accessor.Update(job.ID, "completedSteps", strconv.Itoa(i+1))
	}

	if err != nil {
		status = "error"
	} else {
		status = "complete"
	}

	jm.accessor.Update(job.ID, "status", status)
	return err
}

func (jm *jobManager) GetLog(job *Job, index int) (*JobLog, error) {
	return jm.accessor.GetJobLog(job.ID, index)
}

func (jm *jobManager) Delete(job *Job) error {
	return jm.accessor.Delete(job.ID)
}

func (jm *jobManager) executeStep(job *Job, stepIndex int, stdIn io.Reader) (io.Reader, error) {
	stdOut := &bytes.Buffer{}
	stdErr := &bytes.Buffer{}
	step := job.Steps[stepIndex]

	// Each step gets its own environment, plus the job-level environment
	step.Environment = append(step.Environment, job.Environment...)
	container := jm.containerFactory.NewContainer(step.Source, stringifyEnvironment(step.Environment))

	if err := container.Create(); err != nil {
		return nil, err
	}

	defer container.Remove()

	go func() {
		container.Attach(stdIn, stdOut, stdErr)
		stdOut.Write([]byte{EOT, '\n'})
	}()

	if err := container.Start(); err != nil {
		return nil, err
	}

	output, err := jm.captureOutput(job, stdOut)
	if err != nil {
		return nil, err
	}
	log.Debugf("Container %s stopped", container)

	if err := container.Inspect(); err != nil {
		return nil, err
	}

	return output, nil
}

func (jm *jobManager) captureOutput(job *Job, r io.Reader) (io.Reader, error) {
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
			jm.accessor.AppendLogLine(job.ID, s)

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
