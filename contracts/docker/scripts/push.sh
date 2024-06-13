#!/bin/sh

latest_commit=$(git log -1 --pretty=format:%h)
tag=${latest_commit:0:8}
echo "Using Docker image tag: $tag"
echo ""

docker push scrolltech/scroll-stack-contracts:gen-configs-$tag-amd64
docker push scrolltech/scroll-stack-contracts:gen-configs-$tag-arm64

docker manifest create scrolltech/scroll-stack-contracts:gen-configs-$tag \
    --amend scrolltech/scroll-stack-contracts:gen-configs-$tag-amd64 \
    --amend scrolltech/scroll-stack-contracts:gen-configs-$tag-arm64

docker manifest push scrolltech/scroll-stack-contracts:gen-configs-$tag

docker push scrolltech/scroll-stack-contracts:deploy-$tag-amd64
docker push scrolltech/scroll-stack-contracts:deploy-$tag-arm64

docker manifest create scrolltech/scroll-stack-contracts:deploy-$tag \
    --amend scrolltech/scroll-stack-contracts:deploy-$tag-amd64 \
    --amend scrolltech/scroll-stack-contracts:deploy-$tag-arm64

docker manifest push scrolltech/scroll-stack-contracts:deploy-$tag
