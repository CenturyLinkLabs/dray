package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/CenturyLinkLabs/dray/job"
	log "github.com/Sirupsen/logrus"
)

func listJobs(jm job.JobManager, r requestHelper, w http.ResponseWriter) {
	jobs, err := jm.ListAll()
	if err != nil {
		handleErr(err, w)
		return
	}

	json.NewEncoder(w).Encode(jobs)
}

func createJob(jm job.JobManager, r requestHelper, w http.ResponseWriter) {
	j := &job.Job{}
	err := json.NewDecoder(r.Body()).Decode(j)
	if err != nil {
		handleErr(err, w)
		return
	}

	err = jm.Create(j)
	if err != nil {
		handleErr(err, w)
		return
	}

	go func() {
		if err := jm.Execute(j); err != nil {
			log.Error(err)
		}
	}()

	json.NewEncoder(w).Encode(j)
}

func getJob(jm job.JobManager, r requestHelper, w http.ResponseWriter) {
	j, err := jm.GetByID(r.Param("jobid"))

	if err != nil {
		handleErr(err, w)
		return
	}

	json.NewEncoder(w).Encode(j)
}

func getJobLog(jm job.JobManager, r requestHelper, w http.ResponseWriter) {
	index, err := strconv.Atoi(r.Query("index"))
	if err != nil {
		index = 0
	}

	j, err := jm.GetByID(r.Param("jobid"))
	if err != nil {
		handleErr(err, w)
		return
	}

	log, err := jm.GetLog(j, index)
	if err != nil {
		handleErr(err, w)
		return
	}

	json.NewEncoder(w).Encode(log)
}

func deleteJob(jm job.JobManager, r requestHelper, w http.ResponseWriter) {
	j, err := jm.GetByID(r.Param("jobid"))
	if err != nil {
		handleErr(err, w)
		return
	}

	err = jm.Delete(j)
	if err != nil {
		handleErr(err, w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleErr(err error, w http.ResponseWriter) {
	log.Error(err)
	w.Header().Del("Content-Type")

	if _, ok := err.(job.NotFoundError); ok {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
