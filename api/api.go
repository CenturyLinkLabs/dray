package api

import (
	log "github.com/Sirupsen/logrus"
	"github.com/fzzy/radix/extra/pool"
	"github.com/fzzy/radix/redis"
	"github.com/gorilla/mux"
	"net/http"
	"os"
)

var (
	redisPool *pool.Pool
)

func init() {
	log.SetLevel(log.DebugLevel)
	redisPool, _ = pool.NewPool("tcp", "127.0.0.1:6379", 4)
}

type handler func(c context, w http.ResponseWriter, r *http.Request)

type context struct {
	redis  *redis.Client
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
	}

	for method, routes := range m {
		for route, fct := range routes {

			localMethod := method
			localRoute := route
			localFct := fct
			wrap := func(w http.ResponseWriter, r *http.Request) {
				redis, _ := redisPool.Get()
				defer redisPool.Put(redis)
				c := context{
					redis:  redis,
					params: mux.Vars(r),
				}

				log.Infof("%s %s", r.Method, r.RequestURI)

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
