# dray

![Dray Logo](http://www.centurylinklabs.com/wp-content/uploads/2015/03/dray-600x360.jpg)

[![Circle CI](https://circleci.com/gh/CenturyLinkLabs/dray.svg?style=svg)](https://circleci.com/gh/CenturyLinkLabs/dray)
[![GoDoc](http://godoc.org/github.com/CenturyLinkLabs/dray?status.png)](http://godoc.org/github.com/CenturyLinkLabs/dray)
[![Docker Hub](https://img.shields.io/badge/docker-ready-blue.svg)](https://registry.hub.docker.com/u/centurylink/dray/)
[![](https://badge.imagelayers.io/centurylink/dray.svg)](https://imagelayers.io/?images=centurylink/dray:latest 'Get your own badge on imagelayers.io')
[![Analytics](https://ga-beacon.appspot.com/UA-49491413-7/dray/README?pixel)](https://github.com/CenturyLinkLabs/dray)

An engine for managing the execution of container-based workflows.

Most common Docker use cases involve using containers for hosting long-running
services. These are things like a web application, database or message queue -- services
that are running continuously, waiting to service requests.

Another interesting use case for Docker is to wrap short-lived, single-purpose tasks.
Perhaps it's a Ruby app that needs to be execute periodically or a set of bash scripts 
that need to be executed in sequence. Much like the services described above, these things
can be wrapped in a Docker container to provide an isolated execution environment. The only
real difference is that the task containers exit when they've finished their work while the
service containers run until they are explicitly stopped.

Once you start using task containers, you may find it useful to execute a set of these containers
together in sequence. Maybe you want to string together a set of tasks and have
the output of one container feed the input of the next container. Something like unix pipes:

    cat customers.txt | sort | uniq | wc -l

This is the service that Dray provides. Dray allows you to define a serial workflow, or job, as a 
list of Docker containers with each container encapsulating a step in the workflow. Dray 
will ensure that each step of the workflow (each container) is started in the correct 
order and handles the work of marshaling data between the different steps.

## NOTE

This repo is no longer being maintained. Users are welcome to fork it, but we make no warranty of its functionality.

## Overview
Dray is a Go application that provides a RESTful API for managing jobs. A job is simply a list of Docker containers to be executed in sequence that is posted to Dray as a JSON document:

	{  
	  "name":"Word Job",
	  "steps":[  
	    {  
	      "source":"centurylink/randword"
	    },
	    {  
	      "source":"centurylink/upper"
	    },
	    {  
	      "source":"centurylink/reverse"
	    }
	  ]
	}

The JSON above describes a job named "Word Job" which consists of three steps. Each step references the name of a Docker image to be executed.

When receiving this job description, Dray will immediately return a response containing an ID for the job and then execute the "centurylink/randword" image . As the container is executing Dray will capture any data written to the container's *stdout* stream so that it can be passed along to the next step in the list (there are other output channels you can use, but *stdout* is the default).

Once the "randword" container exits, Dray will inspect the exit code for the container. If, and only if, the exit code is zero, Dray will start the "centurylink/upper" container and pass any data captured in the previous step to that container's *stdin* stream.

Dray will continue executing each of the steps in this manner, marshalling the *stdout* of one step to the *stdin* of the next step, until all of the steps have been completed (or until one of the steps exits with a non-zero exit code).

That status of a running job can be queried at any point by hitting Dray's `/jobs/(id)` endpoint. Additionally, any output generated by the job can be viewed by querying the `/jobs/(id)/log` endpoint.

Note that the example above is a working job description that you can execute on your own Dray installation -- each of the referenced images can be found on the Docker Hub.

## Running

Dray is packaged as a small Docker image and can easily be executed with the Docker *run* command.

Dray relies on [Redis](http://redis.io/) for persisting information about jobs so you'll first need to start one of the [numerous](https://registry.hub.docker.com/search?q=redis&searchfield=) Redis Docker images. In the example below we're simply using the [official Redis image](https://registry.hub.docker.com/_/redis/):

    docker run -d --name redis redis
    
Once Redis is running, you can start the Dray container with the following:

    docker run -d --name dray \
      --link redis:redis \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -p 3000:3000 \
      centurylink/dray:latest

The Dray container must be linked to the Redis container using the `--link` flag so that Dray can find the correct Redis endpoint. The Redis container can be named anything you like, but the alias used in the `--link` flag must be "redis".

Since Dray interacts with the Docker API in order launch containers it needs access to the Docker API socket. When starting the container, the `-v` flag needs to be used to make the Docker socket available inside the container.

In the example above, the `-p` flag is used to map the Dray API endpoint
(listening on port 3000 in the container) to port 3000 on the host machine. In
situations where you don't need a mapped port (like when linking another
container to the Dray container) the `-p` flag can be omitted.

If you'd like to use [Docker Compose](https://docs.docker.com/compose/) to start Dray, the following `docker-compose.yml` is equivalent to the steps shown above

	dray:                                                                                                                   
	  image: centurylink/dray
	  links:
	   - redis
	  volumes:
	   - /var/run/docker.sock:/var/run/docker.sock
	  ports:
	   - "3000:3000"
	redis:
	  image: redis
	  
With this `docker-compose.yml` file you can start Redis and Dray by simply issuing a `docker-compose up -d` command.

### Configuration
The Dray service can be configured by injecting environment variables into the container when it is started. At this time, Dray supports the following configuration variables:

* `LOG_LEVEL` - Valid values are "panic", "fatal", "error", "warn", "info" and "debug". By default, Dray writes messages at and above the "info" level. To increase the amount of logging, set the log level to "debug".

Environment variables can be passed to the Dray container by using the `-e` flag as part of the Docker *run* command:

    docker run -d --name dray \
      --link redis:redis \
      -e LOG_LEVEL=debug \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -p 3000:3000 \
      centurylink/dray:latest
      
## Example
Below is an actual Dray job description that is being used as part of the [Panamax](http://panamax.io/) project. The goal of this job is to provision a cluster of servers on AWS and then install some software on those servers.

	{  
	  "name":"aws=fleet",
	  "environment":[  
	    { "variable":"AWS_ACCESS_KEY_ID", "value":"xxxxxx" },
	    { "variable":"AWS_SECRET_ACCESS_KEY", "value":"xxxxxxx" },
	    { "variable":"REGION", "value":"us-west-2a" },
	    { "variable":"NODE_COUNT", "value":"2" },
	    { "variable":"VM_SIZE", "value":"t2.small" },
	    { "variable":"REMOTE_TARGET_NAME", "value":"AWS - Fleet-CoreOS" }
	  ],
	  "steps":[  
	    {  
	      "name":"Step 1",
	      "source":"centurylink/cluster-deploy:aws.fleet"
	    },
	    {  
	      "name":"Step 2",
	      "source":"centurylink/cluster-deploy:agent"
	    },
	    {  
	      "name":"Step 3",
	      "source":"centurylink/remote-agent-install:latest"
	    }
	  ]
	}

This job uses environment variables to pass a bunch of configuration data into the different steps. Things like the AWS credentials and node count can be passed-in at run-time instead of being hard-coded into the images themselves.

This job uses Dray's data marshalling to pass information between the different steps. Step 1 provisions a cluster of virtual serves and the IP addresses of those servers are needed in step 2. The first step simply writes those IP addresses to the *stdout* stream where they are captured by Dray and passed to the *stdin* stream of the second step.

The way this job is structured, job templates can be created for different cloud providers by simply swapping-out the provider-specific steps and changing some environment variables.

## API
Dray jobs are created and monitored using the API endpoints described below.

### Create Job

    POST /jobs
    
Submits a new job for execution. The execution of the job happens asynchronous to the API call -- the API will respond immediately while execution happens in the background. 

The response body will echo back the submitted job description including the ID assigned to the job. The returned job ID can be used to retrieve information about the job using either the `/jobs/(id)` or `/jobs/(id)/log` endpoints.

**Input:**

*job*

* `name` (`string`) - **Optional.** Name of job.
* `environment` (`array` of `envVar`) - **Optional.** List of environment variables. Environment variables specified at the job level will be injected into **all** job steps.
* `steps` (`array` of `step`) - **Required.** List of job steps.

*envVar*

* `variable` (`string`) - **Required.** Name of the environment variable.
* `value` (`string`) - **Required.** Value of the environment variable.

*step*

* `name` (`string`) - **Optional.** Name of step.
* `environment` (`array` of `envVar`) - **Optional.** List of environment variables to be injected into this step's container.
* `source` (`string`) - **Required.** Name of the Docker image to be executed for this step. If the tag is omitted from the image name, will default to "latest".
* `output` (`string`) - **Optional.** Output channel to be captured and passed to the next step in the job. Valid values are "stdout", "stderr" or any absolute file path. Defaults to "stdout" if not specified. See the "Output Channels" section below for more details.
* `refresh` (`boolean`) - **Optional.** Flag indicating whether or not the image identified by the *source* attribute should be refreshed before it is executed. A *true* value will force Dray to do a `docker pull` before the job step is started. A *false* value (the default) indicates that a `docker pull` should be done only if the image doesn't already exist in the local image cache.

**Example Request:**

    POST /jobs HTTP/1.1
    Content-Type: application/json
    
	{  
	  "name":"Demo Job",
	  "steps":[  
	    {  
	      "name":"random-word",
	      "source":"centurylink/randword",
	      "environment":[  
	        { "variable":"WORD_COUNT", "value":"10" }
	      ]
	    },
	    {  
	      "name":"uppercase",
	      "source":"centurylink/upper"
	    },
	    {  
	      "name":"reverse",
	      "source":"centurylink/reverse"
	    }	    
	  ]
	}
    
**Example Response:**

	HTTP/1.1 201 Created
	Content-Type: application/json
	
	{  
	  "id":"51E0E756-A6B4-9CC7-67BD-364970C2268C",
	  "name":"Demo Job",
	  "steps":[  
	    {  
	      "name":"random-word",
	      "source":"centurylink/randword",
	      "environment":[  
	        { "variable":"WORD_COUNT", "value":"10" }
	      ]
	    },
	    {  
	      "name":"uppercase",
	      "source":"centurylink/upper"
	    },
	    {  
	      "name":"reverse",
	      "source":"centurylink/reverse"
	    },	    
	  ]
	}
	
**Status Codes:**

* **201** - no error
* **500** - server error
	  
### List Jobs

    GET /jobs
    
Returns a list of all the job IDs. Every time that a job is started, it is assigned a unique ID and some basic information is persisted. This call will return the IDs of all the persisted jobs.

**Example Request:**

    GET /jobs HTTP/1.1
    
**Example Response:**

	HTTP/1.1 200 OK
	Content-Type: application/json
	
	[  
	  {  
	    "id":"E2C7017E-449D-B4AA-1BEB-F85224DFC0E1"
	  },
	  {  
	    "id":"26C4A46D-C615-E978-521F-A0D8FDD80801"
	  },
	  {  
	    "id":"51E0E756-A6B4-9CC7-67BD-364970C2268C"
	  }
	]
	
**Status Codes:**

* **200** - no error
* **500** - server error

### Get Job

    GET /jobs/(id)
    
Returns the state of the specified job. The response will include the number of steps which have been completed and an overall status for the job. 

The status will be one of "running", "complete", or "error". The "error" status indicates that one of the steps exited with a non-zero exit code.

**Exampel Request:**

    GET /jobs/51E0E756-A6B4-9CC7-67BD-364970C2268C HTTP/1.1
    
**Example Response:**

    HTTP/1.1 200 OK
    Content-Type: application/json
    
	{
  	  "id": "51E0E756-A6B4-9CC7-67BD-364970C2268C",
	  "stepsCompleted": 2,
	  "status": "complete"
	}
	
**Status Codes:**

* **200** - no error
* **404** - no such job
* **500** - server error

### Get Job Log

    GET /jobs/(id)/log
    
Retrieves the log output of the specified job. While a job is executing any data written to the *stdout* or *stderr* streams (by any of the steps) is persisted and made available via this API endpoint.

**Querystring Params:**

* `index` (`number`) - **Optional.** The starting index for the log output. The response will contain all the log lines starting with the specified index. This can be useful if you are trying to monitor a job while it is still executing. If your first call responds with 10 lines of log output, you can pass `index=10` on your next request and you'll only receive log entries which have been added since your first call. Defaults to 0 if the index is not specified.

**Example Request:**

    GET /jobs/51E0E756-A6B4-9CC7-67BD-364970C2268C/log?index=0 HTTP/1.1
    
**Example Response:**

    HTTP/1.1 200 OK
    Content-Type: application/json
    
    {
      "lines": [
        "Standard output line 1",
		"Standard output line 2",
		"Standard output line 3",
		"Standard error line 1",				
      ]
    }
      
**Status Codes:**

* **200** - no error
* **404** - no such job
* **500** - server error
      
### Delete Job

    DELETE /jobs/(id)
   
Deletes all the information persisted for a given job ID. Note that this will **not** stop a running job, it merely removes all the information persisted for the job in Redis.

**Example Request:**

    DELETE /jobs/51E0E756-A6B4-9CC7-67BD-364970C2268C HTTP/1.1
    
**Example Response:**

    HTTP/1.1 204 No Content
    
**Status Codes:**

* **204** - no error
* **404** - no such job
* **500** - server error

## Output Channels
One of the key features that Dray provides is the ability to marshal data between the different steps (containers) in a job. By default, Dray will capture anything written to the container's *stdout* stream and automatically feed that into the next container's *stdin* stream. However, different output channels can be configured on a step-by-step basis.

### stderr
It is common for tasks/services running in Docker containers to use the *stdout* stream for log output. If you're already using *stdout* for log output and want to use a different channel for data that should be passed to the next job step you can opt to use the *stderr* stream instead. 

To configure Dray to monitor *stderr* for a particular job step you simply use the `output` field for that step in the job description:

	{  
	  "steps":[  
	    {  
	      "source":"jdoe/foo",
	      "output":"stderr"
	    },
	    {  
	      "source":"jdoe/bar"
	    }
	  ]
	}

When creating the "foo" image for use in this job you just need to make sure that any data you want passed to the next step in the job is written to the *stderr* stream.

Here are some examples of writing to *stderr* in different languages:

* Bash - `echo "hello world" >&2`
* Ruby - `STDERR.puts 'hello world'`
* Python - `print("hello world", file=sys.stderr)`

### Custom File
If you don't want to use either the *stdout* or *stderr* streams for passing data, you also have the option of using a regular file. When configured in this way, Dray will volume mount a file into the Docker container at start-up time and then read the contents of the file when the container stops.

To configure Dray to monitor a custom file you need to specify the fully-qualified path of the file (relative to the root of the container) in the `output` field for that step in the job description:

	{  
	  "steps":[  
	    {  
	      "source":"jdoe/foo",
	      "output":"/output.txt"
	    },
	    {  
	      "source":"jdoe/bar"
	    }
	  ]
	}

The `output` value specified **must** begin with a `/` character. The specified file doesn't necessarily need to exist in the image already, Dray will create a temporary file and than volume mount it into the container at the specified location at start-up time.

From within your container, you'll simply need to open the specified file and write to it any data that you would like to have passed to the next step in the job.

There is one other bit of configuration that is also required when using a custom file as an output channel. Since Dray, Docker and your job container all need access to this file, we need a common directory to which all three have access. To enable this you'll need to specify an additional volume mount flag when starting Dray that exposes the host's `/tmp` directory to the Dray container.

    docker run -d --name dray \
      --link redis:redis \
      -v /tmp:/tmp \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -p 3000:3000 \
      centurylink/dray:latest
      
Note the addition of the `-v /tmp:/tmp` flag in the Docker `run` command above. This setting is required **only** if you intend to use custom files as a data-passing mechanism and can be omitted otherwise.

## Building

To facilitate the creation of small Docker image, Dray is compiled into a statically linked binary that can be run with no external dependencies.

The `build.sh` script included in the Dray repository will compile the executable and create the Docker image by leveraging the [centurylink\golang-builder](https://registry.hub.docker.com/u/centurylink/golang-builder/) image. The resulting image is tagged as `centurylink/dray:latest`.
