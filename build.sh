#!/bin/bash

docker run --rm -v $GOPATH:/go -v $PWD:/tmp eawsy/aws-lambda-go
