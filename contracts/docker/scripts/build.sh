#!/bin/sh

latest_commit=$(git log -1 --pretty=format:%h)
tag=${latest_commit:0:8}
echo "Using Docker image tag: $tag"

docker build -f docker/Dockerfile.gen-configs -t scrolltech/scroll-stack-contracts:gen-configs-$tag --platform linux/amd64 .

docker build -f docker/Dockerfile.deploy -t scrolltech/scroll-stack-contracts:deploy-$tag --platform linux/amd64 .