package job

import (
	"bufio"
	"bytes"
	"io"
	"strings"

	log "github.com/Sirupsen/logrus"
)

const (
	BeginDelimiter = "----BEGIN PANAMAX DATA----"
	EndDelimiter   = "----END PANAMAX DATA----"
	EOT            = byte('\u0003')
)

var (
	accessor JobAccessor
)

func init() {
	accessor = &redisJobAccessor{}
}

type Job struct {
	ID             string    `json:"id,omitempty"`
	Name           string    `json:"name,omitempty"`
	Steps          []JobStep `json:"steps,omitempty"`
	StepsCompleted string    `json:"stepsCompleted,omitempty"`
}

type JobStep struct {
	Name   string `json:"name,omitempty"`
	Source string `json:"source,omitempty"`
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

func DeleteJob(jobID string) error {
	return accessor.Delete(jobID)
}

func GetJobLog(jobID string) (*JobLog, error) {
	return accessor.GetJobLog(jobID, 0)
}

func (job *Job) Save() error {
	return accessor.Create(job)
}

func (job *Job) Execute() error {
	var capture io.Reader

	for i := range job.Steps {
		capture, _ = job.executeStep(i, capture)
		accessor.CompleteStep(job.ID)
	}
	return nil
}

func (job *Job) executeStep(stepIndex int, stdIn io.Reader) (io.Reader, error) {

	step := job.Steps[stepIndex]
	stdOut := &bytes.Buffer{}
	stdErr := &bytes.Buffer{}

	container, err := createContainer(step.Source)
	if err != nil {
		return nil, err
	}
	log.Debugf("Container %s created", container.ID[0:12])

	go func() {
		attachContainer(container.ID, stdIn, stdOut, stdErr)
		stdOut.Write([]byte{EOT, '\n'})
	}()

	err = startContainer(container.ID)
	if err != nil {
		return nil, err
	}
	log.Debugf("Container %s started", container.ID[0:12])

	output, err := job.captureOutput(stdOut)
	log.Debugf("Container %s stopped", container.ID[0:12])

	removeContainer(container.ID)
	if err != nil {
		return nil, err
	}
	log.Debugf("Container %s removed", container.ID[0:12])

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
