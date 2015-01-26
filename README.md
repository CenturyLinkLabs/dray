
## Building

The process of building the Dray container involves executing a Docker image 
that, when executed, compiles the Dray code and generates an extremely small 
image that contains nothing but the compiled binary.

The `build-dray.sh` script will execute the [centurylink\golang-builder](https://registry.hub.docker.com/u/centurylink/golang-builder/) image 
with the appropriate arguments -- the end-result should be a new 
`centurylink/dray` image in the `docker images` list.

### Running

Before starting Dray, start Redis and then link Dray to that Redis container:

    docker run -d --name redis redis
    docker run -d --name dray --link redis:redis \
      -v /var/run/docker.sock:/var/run/docker.sock -p 3009:3000 centurylink/dray:latest

In the example above, the `-p` flag is used to map the Dray API endpoint 
(listening on port 3000 in the container) to port 30009 on the host machine. In
situtations where you don't need a mapped port (like when linking another
container to the Dray container) the `-p` flag can be omitted.
    
