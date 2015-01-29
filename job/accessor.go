package job

import (
	"crypto/rand"
	"errors"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/fzzy/radix/extra/pool"
	"github.com/fzzy/radix/redis"
)

const (
	jobsKey = "jobs"
)

type NotFoundError string

func (s NotFoundError) Error() string {
	return fmt.Sprintf("Cannot find job with ID %s", string(s))
}

type JobAccessor interface {
	All() ([]Job, error)
	Get(jobID string) (*Job, error)
	Create(job *Job) error
	Delete(jobID string) error
	Update(jobID, attr, value string) error
	GetJobLog(jobID string, index int) (*JobLog, error)
	AppendLogLine(jobID, logLine string) error
}

type redisJobAccessor struct {
	pool *pool.Pool
}

func NewJobAccessor(host string) JobAccessor {
	pool, err := pool.NewPool("tcp", host, 4)
	if err != nil {
		log.Errorf("Error instantiating Redis pool: %s", err)
		panic(err)
	}

	return &redisJobAccessor{pool: pool}
}

func (a *redisJobAccessor) All() ([]Job, error) {
	jobs := []Job{}

	jobIDs, err := a.command("lrange", jobsKey, 0, -1).List()
	if err != nil {
		return nil, err
	}

	for _, jobID := range jobIDs {
		jobs = append(jobs, Job{ID: jobID})
	}

	return jobs, nil
}

func (a *redisJobAccessor) Get(jobID string) (*Job, error) {
	job := Job{ID: jobID}
	reply := a.command("hgetall", jobKey(jobID))

	if len(reply.Elems) == 0 {
		return nil, NotFoundError(jobID)
	}

	status, err := reply.Hash()
	if err != nil {
		return nil, err
	}

	job.StepsCompleted = status["completedSteps"]
	job.Status = status["status"]
	return &job, nil
}

func (a *redisJobAccessor) Create(job *Job) error {
	job.ID = pseudoUUID()

	reply := a.command("rpush", jobsKey, job.ID)
	if reply.Err != nil {
		fmt.Println(reply.Err)
		return reply.Err
	}

	totalSteps := string(len(job.Steps))
	reply = a.command("hmset", jobKey(job.ID), "totalSteps", totalSteps, "completedSteps", "0", "status", "")
	return reply.Err
}

func (a *redisJobAccessor) Delete(jobID string) error {
	reply := a.command("lrem", jobsKey, 0, jobID)
	if reply.Err != nil {
		return reply.Err
	}

	reply = a.command("del", jobKey(jobID))
	if reply.Err != nil {
		return reply.Err
	}

	reply = a.command("del", jobLogKey(jobID))
	return reply.Err
}

func (a *redisJobAccessor) Update(jobID, attr, value string) error {
	reply := a.command("hset", jobKey(jobID), attr, value)
	return reply.Err
}

func (a *redisJobAccessor) GetJobLog(jobID string, index int) (*JobLog, error) {
	lines, err := a.command("lrange", jobLogKey(jobID), index, -1).List()
	if err != nil {
		return nil, err
	}

	return &JobLog{Lines: lines}, nil
}

func (a *redisJobAccessor) AppendLogLine(jobID, logLine string) error {
	reply := a.command("rpush", jobLogKey(jobID), logLine)
	return reply.Err
}

func (a *redisJobAccessor) command(cmd string, args ...interface{}) *redis.Reply {
	client, err := a.pool.Get()
	if err != nil {
		return &redis.Reply{Err: err}
	}
	defer a.pool.Put(client)

	reply := client.Cmd(cmd, args...)

	// Use a more friendly error message for connection problems
	if reply.Err != nil {
		if _, ok := reply.Err.(*redis.CmdError); !ok {
			reply.Err = errors.New("Redis connection error")
		}
	}

	return reply
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
