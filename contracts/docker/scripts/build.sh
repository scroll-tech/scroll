#!/bin/sh

latest_commit=$(git log -1 --pretty=format:%h)
tag=${latest_commit:0:8}
echo "Using Docker image tag: $tag"
echo ""

docker build -f docker/Dockerfile.gen-configs -t scrolltech/scroll-stack-contracts:gen-configs-$tag-amd64 --platform linux/amd64 .
echo
echo "built scrolltech/scroll-stack-contracts:gen-configs-$tag-amd64"
echo

docker build -f docker/Dockerfile.gen-configs -t scrolltech/scroll-stack-contracts:gen-configs-$tag-arm64 --platform linux/arm64 .
echo
echo "built scrolltech/scroll-stack-contracts:gen-configs-$tag-arm64"
echo

docker build -f docker/Dockerfile.deploy -t scrolltech/scroll-stack-contracts:deploy-$tag-amd64 --platform linux/amd64 .
echo
echo "built scrolltech/scroll-stack-contracts:deploy-$tag-amd64"
echo

docker build -f docker/Dockerfile.deploy -t scrolltech/scroll-stack-contracts:deploy-$tag-arm64 --platform linux/arm64 .
echo
echo "built scrolltech/scroll-stack-contracts:deploy-$tag-arm64"
echo
