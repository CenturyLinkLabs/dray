package api

import (
	"io"
	"net/http"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

type handler func(c context, w http.ResponseWriter)

type context struct {
	request *http.Request
}

func (c *context) Params(key string) string {
	return mux.Vars(c.request)[key]
}

func (c *context) Query(key string) string {
	v := c.request.URL.Query()[key]

	if len(v) == 0 {
		return ""
	}

	return v[0]
}

func (c *context) Body() io.ReadCloser {
	return c.request.Body
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
				c := context{request: r}

				log.Infof("%s %s", r.Method, r.RequestURI)

				if localMethod != "DELETE" {
					w.Header().Set("Content-Type", "application/json")
				}

				localFct(c, w)
			}

			router.Path("/v{version:[0-9.]+}" + localRoute).Methods(localMethod).HandlerFunc(wrap)
			router.Path(localRoute).Methods(localMethod).HandlerFunc(wrap)
		}
	}

	return router, nil
}

func ListenAndServe() {
	router, err := createRouter()
	if err != nil {
		log.Errorf("error:", err)
		os.Exit(1)
	}

	log.Infof("Server running on port 2000")
	http.ListenAndServe(":2000", router)
}
