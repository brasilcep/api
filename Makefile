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

build:
	@echo "Building version $(VERSION) (commit: $(COMMIT))"
	go build -ldflags "-X '$(REPO).api.Version=$(VERSION)' -X '$(REPO).api.Commit=$(COMMIT)' -X '$(REPO).Repo=$(REPO)' -X '$(REPO).Compiler=$(GO_VERSION)'" -o wserver main.go
test:
	go test -v ./......
lint:
	golangci-lint run --timeout 5m

docker-build:
	tar -czf data.tar.gz data/
	docker build -t brasilcep/api:dev .

run-docker: 
	docker run --cpus="0.5" --memory="512m" \
	-p 8080:8080 \
	brasilcep/api:dev