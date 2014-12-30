package job

import (
	"crypto/rand"
	"fmt"

	"github.com/fzzy/radix/extra/pool"
	"github.com/fzzy/radix/redis"
)

const (
	jobsKey = "jobs"
)

var (
	rp *pool.Pool
)

func init() {
	rp, _ = pool.NewPool("tcp", "127.0.0.1:6379", 4)
}

type JobAccessor interface {
	All() ([]Job, error)
	Get(jobID string) (*Job, error)
	Create(job *Job) error
	Delete(jobID string) error
	CompleteStep(jobID string) error
	GetJobLog(jobID string, index int) (*JobLog, error)
	AppendLogLine(jobID, logLine string) error
}

type redisJobAccessor struct {
}

func (*redisJobAccessor) All() ([]Job, error) {
	jobs := []Job{}

	jobIDs, err := command("lrange", jobsKey, 0, -1).List()
	if err != nil {
		return nil, err
	}

	for _, jobID := range jobIDs {
		jobs = append(jobs, Job{ID: jobID})
	}

	return jobs, nil
}

func (*redisJobAccessor) Get(jobID string) (*Job, error) {
	job := Job{ID: jobID}
	status, err := command("hgetall", jobKey(jobID)).Hash()
	if err != nil {
		return nil, err
	}

	job.StepsCompleted = status["completedSteps"]
	return &job, nil
}

func (*redisJobAccessor) Create(job *Job) error {
	job.ID = pseudoUUID()

	reply := command("rpush", jobsKey, job.ID)
	if reply.Err != nil {
		return reply.Err
	}

	totalSteps := string(len(job.Steps))
	reply = command("hmset", jobKey(job.ID), "totalSteps", totalSteps, "completedSteps", "0")
	return reply.Err
}

func (*redisJobAccessor) Delete(jobID string) error {
	reply := command("lrem", jobsKey, 0, jobID)
	if reply.Err != nil {
		return reply.Err
	}

	reply = command("del", jobKey(jobID))
	if reply.Err != nil {
		return reply.Err
	}

	reply = command("del", jobLogKey(jobID))
	return reply.Err
}

func (*redisJobAccessor) CompleteStep(jobID string) error {
	reply := command("hincrby", jobKey(jobID), "completedSteps", 1)
	return reply.Err
}

func (*redisJobAccessor) GetJobLog(jobID string, index int) (*JobLog, error) {
	lines, err := command("lrange", jobLogKey(jobID), index, -1).List()
	if err != nil {
		return nil, err
	}

	return &JobLog{Lines: lines}, nil
}

func (*redisJobAccessor) AppendLogLine(jobID, logLine string) error {
	reply := command("rpush", jobLogKey(jobID), logLine)
	return reply.Err
}

func command(cmd string, args ...interface{}) *redis.Reply {
	client, err := rp.Get()
	if err != nil {
		return &redis.Reply{Err: err}
	}
	defer rp.Put(client)

	return client.Cmd(cmd, args...)
}

func jobKey(jobID string) string {
	return fmt.Sprintf("%s:%s", jobsKey, jobID)
}

func jobLogKey(jobID string) string {
	return fmt.Sprintf("%s:%s:log", jobsKey, jobID)
}

func pseudoUUID() (uuid string) {
	b := make([]byte, 16)
	rand.Read(b)

	return fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
