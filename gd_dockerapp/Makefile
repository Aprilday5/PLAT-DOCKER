.PHONY: build test clean prepare update docker

GO = CGO_ENABLED=0 GO111MODULE=on go
GOCGO = CGO_ENABLED=1 GO111MODULE=on go

MICROSERVICES=cmd/gddockerapp

.PHONY: $(MICROSERVICES)

DOCKERS=docker_gddockerapp_go
.PHONY: $(DOCKERS)

VERSION=$(shell cat ./VERSION 2>/dev/null || echo 0.0.0)
GIT_SHA=$(shell git rev-parse HEAD)
GOFLAGS=-ldflags "-X gddockerapp.Version=$(VERSION)"

build: $(MICROSERVICES)

cmd/gddockerapp:
	go mod tidy
	$(GOCGO) build $(GOFLAGS) -o $@ ./cmd


clean:
	rm -f $(MICROSERVICES)

docker: $(DOCKERS)

docker_gddockerapp_go:
	docker build \
		--label "git_sha=$(GIT_SHA)" \
		-t gddockerapp:$(VERSION)\
		.
