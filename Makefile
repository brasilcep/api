COMMIT := $(shell git rev-parse --short HEAD)
VERSION := $(shell \
	TAG=$$(git describe --tags --exact-match 2>/dev/null || true); \
	if [ -n "$$TAG" ]; then \
		echo "$$TAG"; \
	elif [ "$$(git rev-parse --abbrev-ref HEAD 2>/dev/null)" != "HEAD" ]; then \
		git rev-parse --abbrev-ref HEAD; \
	else \
		git describe --tags --always --dirty; \
	fi \
)
GO_VERSION := $(shell go version | awk '{print $$3}')

REPO := github.com/brasilcep/api
IMAGE_NAME ?= brasilcep/api
IMAGE_TAG ?= dev
DOCKER_IMAGE := $(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: build test lint docker-build docker-push docker-publish run-docker benchmark-docker

build:
	@echo "Building version $(VERSION) (commit: $(COMMIT))"
	go build -ldflags "\
		-X 'main.Version=$(VERSION)' \
		-X 'main.Commit=$(COMMIT)' \
		-X 'main.Repo=$(REPO)' \
		-X 'main.Compiler=$(GO_VERSION)' \
	" -o wserver main.go

test:
	go test -v ./......
lint:
	golangci-lint run --timeout 5m

docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE)"
	docker build \
		--build-arg VERSION="$(VERSION)" \
		--build-arg COMMIT="$(COMMIT)" \
		--build-arg REPO="$(REPO)" \
		-t "$(DOCKER_IMAGE)" .

docker-push:
	docker push "$(DOCKER_IMAGE)"

docker-publish: docker-build
	docker push "$(DOCKER_IMAGE)"

run-docker:
	docker run --cpus="0.5" --memory="512m" \
	-p 8080:8080 \
	"$(DOCKER_IMAGE)"

benchmark-docker:
	docker compose up brasilcep-api -d
	@echo "Waiting for the API to start and load data..."
	sleep 120
	docker compose up
