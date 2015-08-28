#!/bin/bash
docker rm -f ec2metaproxy
docker run -d \
	--net=host \
	-v /var/run/docker.sock:/var/run/docker.sock \
	--name=ec2metaproxy \
	flyinprogrammer/ec2metaproxy:latest "$@"
