
## Running

As currently implemented, stevedore needs access to Redis (port 6379) and Docker (port 2375) running on the localhost (connection strings are currently hard-coded).

### Redis
Simplest way to get Redis running is to run it in-container and map port 6379:

    docker run -d -p 6379:6379 redis
    
Make sure that port 6379 is also mapped in VirtualBox so that it is accessible from your host machine.

### Docker
You'll also need to bind the Docker daemon to a TCP port so that a client running outside the VM can interact with it.

To do this in CoreOS (or any other platform where the Docker daemon is being managed by systemd) you'll need to create a file named `/etc/systemd/system/docker-tcp.socket` with the following contents

	[Unit]
	Description=Docker Socket for the API
	
	[Socket]
	ListenStream=2375
	BindIPv6Only=both
	Service=docker.service
	
	[Install]
	WantedBy=sockets.target

Then enable the socket binding by issuing the following commands:

	sudo systemctl enable docker-tcp.socket
	sudo systemctl stop docker
	sudo systemctl start docker-tcp.socket
	sudo systemctl start docker
	
Note: When executing the steps above you may find that the docker service starts again immediately after the `systemctl stop docker` command. If this happens, the attempt to start the docker-tcp.socket may fail. If you experience this, try stopping any running containers you have before executing these steps.

There are [instructions](https://coreos.com/docs/launching-containers/building/customizing-docker/#cloud-config) on the CoreOS site for doing this set-up automatically via cloud-config in case you don't want to manually set it up every time you start a new CoreOS instance.

Similar to the Redis set-up above, make sure that port 2375 is mapped in VirtualBox so that it is accessible from your host.

### Stevedore
After completing the Redis and Docker configuration above you should be able to start stevedore by navigating to the root project directory an executing the following:

    go run main.go


   
