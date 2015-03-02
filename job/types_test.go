package job

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJobCurrentStep(t *testing.T) {
	job := Job{
		Steps:          []JobStep{{Name: "step1"}, {Name: "step2"}},
		StepsCompleted: 1,
	}

	assert.Equal(t, &job.Steps[1], job.CurrentStep())
}

func TestJobCurrentStepEnvironment(t *testing.T) {
	var1 := EnvVar{Variable: "foo", Value: "bar"}
	var2 := EnvVar{Variable: "fiz", Value: "bin"}
	job := Job{
		Environment:    Environment{var1},
		Steps:          []JobStep{{Environment: Environment{var2}}},
		StepsCompleted: 0,
	}

	env := job.CurrentStepEnvironment()
	assert.Len(t, env, 2)
	assert.Contains(t, env, var1)
	assert.Contains(t, env, var2)
}

func TestJobStepUsesStdOutPipe(t *testing.T) {
	js := JobStep{}
	assert.True(t, js.UsesStdOutPipe())

	js = JobStep{Output: "stdout"}
	assert.True(t, js.UsesStdOutPipe())

	js = JobStep{Output: "foo"}
	assert.False(t, js.UsesStdOutPipe())
}

func TestJobStepUsesStdErrPipe(t *testing.T) {
	js := JobStep{Output: "stderr"}
	assert.True(t, js.UsesStdErrPipe())

	js = JobStep{}
	assert.False(t, js.UsesStdErrPipe())

	js = JobStep{Output: "foo"}
	assert.False(t, js.UsesStdErrPipe())
}

func TestJobStepUsesFilePipe(t *testing.T) {
	js := JobStep{Output: "/foo"}
	assert.True(t, js.UsesFilePipe())

	js = JobStep{}
	assert.False(t, js.UsesFilePipe())

	js = JobStep{Output: "foo"}
	assert.False(t, js.UsesFilePipe())
}

func TestJobStepFilePipePath(t *testing.T) {
	js := JobStep{Source: "foo"}

	// Using hard-coded md5 hash of the string "foo"
	assert.Equal(t, "/tmp/acbd18db4cc2f85cedef654fccc4a4d8", js.FilePipePath())
}

func TestEnvVarString(t *testing.T) {
	e := EnvVar{Variable: "foo", Value: "bar"}
	assert.Equal(t, "foo=bar", e.String())
}

func TestEnvironmentStringfy(t *testing.T) {
	e := Environment{
		{Variable: "foo", Value: "bar"},
		{Variable: "fizz", Value: "bin"},
	}

	s := e.Stringify()
	assert.Len(t, s, len(e))
	assert.Contains(t, s, "foo=bar")
	assert.Contains(t, s, "fizz=bin")
}
