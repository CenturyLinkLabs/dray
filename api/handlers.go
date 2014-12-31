package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/CenturyLinkLabs/stevedore/job"
)

func listJobs(context context, responseWriter http.ResponseWriter) {
	jobs, err := job.ListAll()
	if err != nil {
		handleErr(err, responseWriter)
		return
	}

	json.NewEncoder(responseWriter).Encode(jobs)
}

func createJob(context context, responseWriter http.ResponseWriter) {
	j := job.Job{}
	err := json.NewDecoder(context.Body()).Decode(&j)
	if err != nil {
		handleErr(err, responseWriter)
		return
	}

	err = j.Create()
	if err != nil {
		handleErr(err, responseWriter)
		return
	}

	go j.Execute()

	json.NewEncoder(responseWriter).Encode(j)
}

func getJob(context context, responseWriter http.ResponseWriter) {
	j, err := job.GetByID(context.Params("jobid"))

	if err != nil {
		handleErr(err, responseWriter)
		return
	}

	json.NewEncoder(responseWriter).Encode(j)
}

func getJobLog(context context, responseWriter http.ResponseWriter) {
	index, err := strconv.Atoi(context.Query("index"))
	if err != nil {
		index = 0
	}

	j, err := job.GetByID(context.Params("jobid"))
	if err != nil {
		handleErr(err, responseWriter)
		return
	}

	log, err := j.GetLog(index)
	if err != nil {
		handleErr(err, responseWriter)
		return
	}

	json.NewEncoder(responseWriter).Encode(log)
}

func deleteJob(context context, responseWriter http.ResponseWriter) {
	j, err := job.GetByID(context.Params("jobid"))
	if err != nil {
		handleErr(err, responseWriter)
		return
	}

	err = j.Delete()
	if err != nil {
		handleErr(err, responseWriter)
		return
	}

	responseWriter.WriteHeader(http.StatusNoContent)
}

func handleErr(err error, w http.ResponseWriter) {
	w.Header().Del("Content-Type")

	if _, ok := err.(job.NotFoundError); ok {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
