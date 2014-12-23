package api

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
)

func listJobs(context context, responseWriter http.ResponseWriter, request *http.Request) {
	var jobs []string
	jobs, err := context.redis.Cmd("lrange", "jobs", 0, -1).List()
	handleErr(err)

	json.NewEncoder(responseWriter).Encode(jobs)
}

func createJob(context context, responseWriter http.ResponseWriter, request *http.Request) {
	job := Job{ID: pseudoUUID()}
	err := json.NewDecoder(request.Body).Decode(&job)
	handleErr(err)

	context.redis.Cmd("rpush", "jobs", job.ID)

	go ExecuteJob(&job)
	json.NewEncoder(responseWriter).Encode(job)
}

func getJob(context context, responseWriter http.ResponseWriter, request *http.Request) {

	job := Job{}
	job.ID = context.params["jobid"]

	json.NewEncoder(responseWriter).Encode(job)
}

type jobLog struct {
	Index int
	Lines []string
}

func getJobLog(context context, responseWriter http.ResponseWriter, request *http.Request) {

	jobID := context.params["jobid"]

	lines, err := context.redis.Cmd("lrange", "job:"+jobID+":log", 0, -1).List()
	handleErr(err)

	log := jobLog{Lines: lines}
	json.NewEncoder(responseWriter).Encode(log)
}

func pseudoUUID() (uuid string) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	handleErr(err)

	return fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
