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

	assert.Equal(t, &job.Steps[1], job.currentStep())
}

func TestJobCurrentStepEnvironment(t *testing.T) {
	var1 := EnvVar{Variable: "foo", Value: "bar"}
	var2 := EnvVar{Variable: "fiz", Value: "bin"}

	job := Job{
		Name:           "foo",
		Environment:    Environment{var1},
		Steps:          []JobStep{{Environment: Environment{var2}, Name: "bar"}},
		StepsCompleted: 0,
	}

	var3 := EnvVar{Variable: "DRAY_JOB_ID", Value: job.ID}
	var4 := EnvVar{Variable: "DRAY_JOB_NAME", Value: "foo"}
	var5 := EnvVar{Variable: "DRAY_CURRENT_STEP_INDEX", Value: "0"}
	var6 := EnvVar{Variable: "DRAY_CURRENT_STEP_NAME", Value: "bar"}

	env := job.currentStepEnvironment()
	assert.Len(t, env, 6)
	assert.Contains(t, env, var1)
	assert.Contains(t, env, var2)
	assert.Contains(t, env, var3)
	assert.Contains(t, env, var4)
	assert.Contains(t, env, var5)
	assert.Contains(t, env, var6)
}

func TestJobStepUsesStdOutPipe(t *testing.T) {
	js := JobStep{}
	assert.True(t, js.usesStdOutPipe())

	js = JobStep{Output: "stdout"}
	assert.True(t, js.usesStdOutPipe())

	js = JobStep{Output: "foo"}
	assert.False(t, js.usesStdOutPipe())
}

func TestJobStepUsesStdErrPipe(t *testing.T) {
	js := JobStep{Output: "stderr"}
	assert.True(t, js.usesStdErrPipe())

	js = JobStep{}
	assert.False(t, js.usesStdErrPipe())

	js = JobStep{Output: "foo"}
	assert.False(t, js.usesStdErrPipe())
}

func TestJobStepUsesFilePipe(t *testing.T) {
	js := JobStep{Output: "/foo"}
	assert.True(t, js.usesFilePipe())

	js = JobStep{}
	assert.False(t, js.usesFilePipe())

	js = JobStep{Output: "foo"}
	assert.False(t, js.usesFilePipe())
}

func TestJobStepFilePipePath(t *testing.T) {
	js := JobStep{Source: "foo"}

	// Using hard-coded md5 hash of the string "foo"
	assert.Equal(t, "/tmp/acbd18db4cc2f85cedef654fccc4a4d8", js.filePipePath())
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

	s := e.stringify()
	assert.Len(t, s, len(e))
	assert.Contains(t, s, "foo=bar")
	assert.Contains(t, s, "fizz=bin")
}
