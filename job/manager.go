package job

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
)

const (
	fieldStatus         = "status"
	fieldCompletedSteps = "completedSteps"
	fieldCreatedAt      = "createdAt"
	fieldFinishedIn     = "finishedIn"

	statusRunning  = "running"
	statusError    = "error"
	statusComplete = "complete"
)

type jobManager struct {
	repository JobRepository
	executor   JobStepExecutor
}

// NewJobManager returns a JobManager instance with connections to the
// specified JobRepository and JobStepExecutor.
func NewJobManager(r JobRepository, e JobStepExecutor) JobManager {
	return &jobManager{
		repository: r,
		executor:   e,
	}
}

func (jm *jobManager) ListAll() ([]Job, error) {
	return jm.repository.All()
}

func (jm *jobManager) GetByID(jobID string) (*Job, error) {
	return jm.repository.Get(jobID)
}

func (jm *jobManager) Create(job *Job) error {
	return jm.repository.Create(job)
}

func (jm *jobManager) Execute(job *Job) error {
	var capture io.Reader
	var err error
	status := statusRunning
	createdAt := time.Now()

	jm.repository.Update(job.ID, fieldStatus, status)
	jm.repository.Update(job.ID, fieldCreatedAt, createdAt.String())

	for i := range job.Steps {
		capture, err = jm.executeStep(job, capture)

		if err != nil {
			break
		}

		job.StepsCompleted++
		jm.repository.Update(job.ID, fieldCompletedSteps, strconv.Itoa(i+1))
	}

	if err != nil {
		status = statusError
	} else {
		status = statusComplete
	}

	jm.repository.Update(job.ID, fieldStatus, status)
	finishedIn := float32(time.Since(createdAt)) / float32(time.Second)
	jm.repository.Update(job.ID, fieldFinishedIn, fmt.Sprintf("%f", finishedIn))
	return err
}

func (jm *jobManager) GetLog(job *Job, index int) (*JobLog, error) {
	return jm.repository.GetJobLog(job.ID, index)
}

func (jm *jobManager) Delete(job *Job) error {
	return jm.repository.Delete(job.ID)
}

func (jm *jobManager) executeStep(job *Job, stdIn io.Reader) (io.Reader, error) {
	var wg sync.WaitGroup
	var outBuffer, errBuffer io.Writer
	var stepOutput io.Reader

	step := job.currentStep()
	stdOutReader, stdOutWriter := io.Pipe()
	stdErrReader, stdErrWriter := io.Pipe()

	if step.usesFilePipe() {
		f, err := os.Create(step.filePipePath())
		if err != nil {
			return nil, err
		}

		f.Close()
		defer os.Remove(step.filePipePath())
	} else {
		buffer := &bytes.Buffer{}
		stepOutput = buffer

		if step.usesStdOutPipe() {
			outBuffer = buffer
		} else if step.usesStdErrPipe() {
			errBuffer = buffer
		}
	}

	err := jm.executor.Start(job, stdIn, stdOutWriter, stdErrWriter)
	if err != nil {
		return nil, err
	}
	defer jm.executor.CleanUp(job)

	wg.Add(2)

	go func() {
		defer wg.Done()
		jm.capture(job, stdOutReader, outBuffer)
	}()

	go func() {
		defer wg.Done()
		jm.capture(job, stdErrReader, errBuffer)
	}()

	wg.Wait()

	if err := jm.executor.Inspect(job); err != nil {
		return nil, err
	}

	if step.usesFilePipe() {
		// Grab data written to pipe file
		b, err := ioutil.ReadFile(step.filePipePath())
		if err != nil {
			return nil, err
		}

		stepOutput = bytes.NewBuffer(b)
	}

	return stepOutput, nil
}

func (jm *jobManager) capture(job *Job, r io.Reader, w io.Writer) {
	step := job.currentStep()
	scanner := bufio.NewScanner(r)
	capture := !step.usesDelimitedOutput()

	for scanner.Scan() {
		line := scanner.Text()

		log.Debugf(line)
		jm.repository.AppendLogLine(job.ID, line)

		if w != nil {
			if step.usesDelimitedOutput() && line == step.EndDelimiter {
				capture = false
			}

			if capture {
				w.Write(append([]byte(line), '\n'))
			}

			if step.usesDelimitedOutput() && line == step.BeginDelimiter {
				capture = true
			}
		}
	}
}
