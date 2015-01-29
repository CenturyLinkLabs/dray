package main // import "github.com/CenturyLinkLabs/dray"

import (
	"flag"
	"net/url"
	"os"
	"strings"

	"github.com/CenturyLinkLabs/dray/api"
	"github.com/CenturyLinkLabs/dray/job"
	log "github.com/Sirupsen/logrus"
)

const (
	DefaultDockerEndpoint = "unix:///var/run/docker.sock"
	DefaultLogLevel       = log.InfoLevel
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(DefaultLogLevel)
}

func main() {
	log.SetLevel(logLevel())

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

func logLevel() log.Level {
	levelString := os.Getenv("LOG_LEVEL")

	if len(levelString) == 0 {
		return DefaultLogLevel
	}

	level, err := log.ParseLevel(strings.ToLower(levelString))
	if err != nil {
		log.Errorf("Invalid log level: %s", levelString)
		return DefaultLogLevel
	}

	return level
}
