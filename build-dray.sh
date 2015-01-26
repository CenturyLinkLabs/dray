#!/bin/sh

docker run --rm \
  -v $(pwd):/src \
  -v /var/run/docker.sock:/var/run/docker.sock \
  centurylink/golang-builder:latest \
  github.com/CenturyLinkLabs/dray \
  centurylink/dray:latest
