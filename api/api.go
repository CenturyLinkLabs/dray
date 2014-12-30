package api

import (
	"net/http"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

type handler func(c context, w http.ResponseWriter, r *http.Request)

type context struct {
	params map[string]string
}

func createRouter() (*mux.Router, error) {
	router := mux.NewRouter()

	m := map[string]map[string]handler{
		"GET": {
			"/jobs":             listJobs,
			"/jobs/{jobid}":     getJob,
			"/jobs/{jobid}/log": getJobLog,
		},
		"POST": {
			"/jobs": createJob,
		},
		"DELETE": {
			"/jobs/{jobid}": deleteJob,
		},
	}

	for method, routes := range m {
		for route, fct := range routes {

			localMethod := method
			localRoute := route
			localFct := fct
			wrap := func(w http.ResponseWriter, r *http.Request) {
				c := context{params: mux.Vars(r)}

				log.Infof("%s %s", r.Method, r.RequestURI)

				if localMethod != "DELETE" {
					w.Header().Set("Content-Type", "application/json")
				}

				localFct(c, w, r)
			}

			router.Path("/v{version:[0-9.]+}" + localRoute).Methods(localMethod).HandlerFunc(wrap)
			router.Path(localRoute).Methods(localMethod).HandlerFunc(wrap)
		}
	}

	return router, nil
}

func ListenAndServe() {
	router, err := createRouter()
	handleErr(err)
	log.Infof("Server running on port 2000")
	http.ListenAndServe(":2000", router)
}

func handleErr(err error) {
	if err != nil {
		log.Errorf("error:", err)
		os.Exit(1)
	}
}
