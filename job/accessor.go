package job

import (
	"crypto/rand"
	"fmt"
	"net/url"
	"os"

	log "github.com/Sirupsen/logrus"
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
	redisPort := os.Getenv("REDIS_PORT")
	if len(redisPort) == 0 {
		log.Error("Missing required REDIS_PORT environment variable")
	}

	u, err := url.Parse(redisPort)
	if err != nil {
		log.Errorf("Invalid Redis URL: %s", err)
		panic(err)
	}

	pool, err := pool.NewPool("tcp", u.Host, 4)
	if err != nil {
		log.Errorf("Error instantiating Redis pool: %s", err)
		panic(err)
	}

	rp = pool
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
}

type NotFoundError string

func (s NotFoundError) Error() string {
	return fmt.Sprintf("Cannot find job with ID %s", string(s))
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
	reply := command("hgetall", jobKey(jobID))

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

func (*redisJobAccessor) Create(job *Job) error {
	job.ID = pseudoUUID()

	reply := command("rpush", jobsKey, job.ID)
	if reply.Err != nil {
		return reply.Err
	}

	totalSteps := string(len(job.Steps))
	reply = command("hmset", jobKey(job.ID), "totalSteps", totalSteps, "completedSteps", "0", "status", "")
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

func (*redisJobAccessor) Update(jobID, attr, value string) error {
	reply := command("hset", jobKey(jobID), attr, value)
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
