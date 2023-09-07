REPO_SERVER=019120760881.dkr.ecr.us-east-1.amazonaws.com

GIT_DIRTY := $(shell git diff --quiet || echo '-dirty')
GIT_SHA := $(shell git rev-parse --short HEAD)
GIT_TAG := ${GIT_SHA}${GIT_DIRTY}

docker:
	docker build -t "${REPO_SERVER}/probelab:boomo-${GIT_TAG}" .

docker-push: docker
	docker push "${REPO_SERVER}/probelab:boomo-${GIT_TAG}"
	docker rmi "${REPO_SERVER}/probelab:boomo-${GIT_TAG}"

format:
	gofumpt -w -l .

.PHONY: all clean build test format tools models migrate-up migrate-down