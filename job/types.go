package job // import "github.com/CenturyLinkLabs/dray/job"

import (
	"crypto/md5"
	"fmt"
	"io"
	"strings"
)

type JobManager interface {
	ListAll() ([]Job, error)
	GetByID(string) (*Job, error)
	Create(*Job) error
	Execute(*Job) error
	GetLog(*Job, int) (*JobLog, error)
	Delete(*Job) error
}

type JobRepository interface {
	All() ([]Job, error)
	Get(jobID string) (*Job, error)
	Create(job *Job) error
	Delete(jobID string) error
	Update(jobID, attr, value string) error
	GetJobLog(jobID string, index int) (*JobLog, error)
	AppendLogLine(jobID, logLine string) error
}

type JobStepExecutor interface {
	Start(js *Job, stdIn io.Reader, stdOut, stdErr io.WriteCloser) error
	Inspect(js *Job) error
	CleanUp(js *Job) error
}

type Job struct {
	ID             string      `json:"id,omitempty"`
	Name           string      `json:"name,omitempty"`
	Steps          []JobStep   `json:"steps,omitempty"`
	Environment    Environment `json:"environment,omitempty"`
	StepsCompleted int         `json:"stepsCompleted,omitempty"`
	Status         string      `json:"status,omitempty"`
}

func (j Job) CurrentStep() *JobStep {
	return &j.Steps[j.StepsCompleted]
}

func (j Job) CurrentStepEnvironment() Environment {
	return append(j.Environment, j.CurrentStep().Environment...)
}

type JobStep struct {
	id             string
	Name           string      `json:"name,omitempty"`
	Source         string      `json:"source,omitempty"`
	Environment    Environment `json:"environment,omitempty"`
	Output         string      `json:"output,omitempty"`
	BeginDelimiter string      `json:"beginDelimiter,omitempty"`
	EndDelimiter   string      `json:"endDelimiter,omitempty"`
	Refresh        bool        `json:"refresh,omitempty"`
}

type Environment []EnvVar

type EnvVar struct {
	Variable string `json:"variable"`
	Value    string `json:"value"`
}

type JobLog struct {
	Index int      `json:"index,omitempty"`
	Lines []string `json:"lines"`
}

func (js JobStep) UsesStdOutPipe() bool {
	return js.Output == "stdout" || js.Output == ""
}

func (js JobStep) UsesStdErrPipe() bool {
	return js.Output == "stderr"
}

func (js JobStep) UsesFilePipe() bool {
	return strings.HasPrefix(js.Output, "/")
}

func (js JobStep) FilePipePath() string {
	return fmt.Sprintf("/tmp/%x", md5.Sum([]byte(js.Source)))
}

func (js JobStep) UsesDelimitedOutput() bool {
	return len(js.BeginDelimiter) > 0 && len(js.EndDelimiter) > 0
}

func (e Environment) Stringify() []string {
	envStrings := make([]string, len(e))

	for i, v := range e {
		envStrings[i] = v.String()
	}

	return envStrings
}

func (e EnvVar) String() string {
	return fmt.Sprintf("%s=%s", e.Variable, e.Value)
}
