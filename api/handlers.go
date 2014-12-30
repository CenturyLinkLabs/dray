package api

import (
	"encoding/json"
	"net/http"

	"github.com/CenturyLinkLabs/stevedore/job"
)

func listJobs(context context, responseWriter http.ResponseWriter, request *http.Request) {
	jobs, err := job.ListAll()
	handleErr(err)
	json.NewEncoder(responseWriter).Encode(jobs)
}

func createJob(context context, responseWriter http.ResponseWriter, request *http.Request) {
	j := job.Job{}
	err := json.NewDecoder(request.Body).Decode(&j)
	handleErr(err)

	err = j.Save()
	handleErr(err)

	go j.Execute()

	json.NewEncoder(responseWriter).Encode(j)
}

func getJob(context context, responseWriter http.ResponseWriter, request *http.Request) {
	j, err := job.GetByID(context.params["jobid"])
	handleErr(err)

	json.NewEncoder(responseWriter).Encode(j)
}

func getJobLog(context context, responseWriter http.ResponseWriter, request *http.Request) {

	jobID := context.params["jobid"]

	log, err := job.GetJobLog(jobID)
	handleErr(err)

	json.NewEncoder(responseWriter).Encode(log)
}

func deleteJob(context context, responseWriter http.ResponseWriter, request *http.Request) {
	jobID := context.params["jobid"]

	err := job.DeleteJob(jobID)
	handleErr(err)

	responseWriter.WriteHeader(http.StatusNoContent)
}
