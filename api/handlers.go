package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/CenturyLinkLabs/dray/job"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

func listJobs(jm job.JobManager, r *http.Request, w http.ResponseWriter) {
	jobs, err := jm.ListAll()
	if err != nil {
		handleErr(err, w)
		return
	}

	json.NewEncoder(w).Encode(jobs)
}

func createJob(jm job.JobManager, r *http.Request, w http.ResponseWriter) {
	j := &job.Job{}
	err := json.NewDecoder(r.Body).Decode(j)
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

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(j)
}

func getJob(jm job.JobManager, r *http.Request, w http.ResponseWriter) {
	jobID := mux.Vars(r)["jobid"]
	j, err := jm.GetByID(jobID)

	if err != nil {
		handleErr(err, w)
		return
	}

	json.NewEncoder(w).Encode(j)
}

func getJobLog(jm job.JobManager, r *http.Request, w http.ResponseWriter) {
	jobID := mux.Vars(r)["jobid"]

	indexQuery := querystringValue(r, "index")
	index, err := strconv.Atoi(indexQuery)
	if err != nil {
		index = 0
	}

	j, err := jm.GetByID(jobID)
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

func deleteJob(jm job.JobManager, r *http.Request, w http.ResponseWriter) {
	jobID := mux.Vars(r)["jobid"]

	j, err := jm.GetByID(jobID)
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

func querystringValue(r *http.Request, key string) string {
	v := r.URL.Query()[key]

	if len(v) == 0 {
		return ""
	}

	return v[0]
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
