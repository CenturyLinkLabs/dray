package main // import "github.com/CenturyLinkLabs/dray"

import (
	"flag"
	"net/url"
	"os"

	"github.com/CenturyLinkLabs/dray/api"
	"github.com/CenturyLinkLabs/dray/job"
	log "github.com/Sirupsen/logrus"
)

const (
	DefaultDockerEndpoint = "unix:///var/run/docker.sock"
)

func main() {
	port := flag.Int("p", 3000, "port on which the server will run")
	flag.Parse()

	a := job.NewJobAccessor(redisHost())
	cf := job.NewContainerFactory(dockerEndpoint())
	jm := job.NewJobManager(a, cf)

	s := api.NewServer(jm)
	s.Start(*port)
}

func redisHost() string {
	redisPort := os.Getenv("REDIS_PORT")

	if len(redisPort) == 0 {
		log.Error("Missing required REDIS_PORT environment variable")
	}

	u, err := url.Parse(redisPort)
	if err != nil {
		log.Errorf("Invalid Redis URL: %s", err)
		panic(err)
	}

	return u.Host
}

func dockerEndpoint() string {
	endpoint := os.Getenv("DOCKER_HOST")

	if len(endpoint) == 0 {
		endpoint = DefaultDockerEndpoint
	}

	return endpoint
}

// For streaming responses
//type flushWriter struct {
//flusher http.Flusher
//writer  io.Writer
//}

//func newFlushWriter(writer io.Writer) flushWriter {
//fw := flushWriter{writer: writer}
//if flusher, ok := writer.(http.Flusher); ok {
//fw.flusher = flusher
//}

//return fw
//}

//func (fw *flushWriter) Write(p []byte) (n int, err error) {
//n, err = fw.writer.Write(p)
//if fw.flusher != nil {
//fw.flusher.Flush()
//}
//return
//}
