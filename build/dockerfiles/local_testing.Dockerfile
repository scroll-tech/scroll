# Start from the latest golang base image
FROM golang:1.19

# Install Docker
RUN apt-get update && apt-get install -y docker.io

# Set the working directory
WORKDIR /go/src/app

# This container will be executable 
ENTRYPOINT [ "/bin/bash" ]
