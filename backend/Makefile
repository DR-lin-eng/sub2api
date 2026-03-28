.PHONY: build generate test test-unit test-integration test-e2e test-unit-docker test-integration-docker

VERSION ?= $(shell tr -d '\r\n' < ./cmd/server/VERSION)
RELEASE_REPO ?= $(shell git remote get-url origin 2>/dev/null | sed -E 's#(git@github.com:|https://github.com/)##; s#\.git$$##')
ifeq ($(strip $(RELEASE_REPO)),)
RELEASE_REPO := DR-lin-eng/sub2api
endif
LDFLAGS ?= -s -w -X main.Version=$(VERSION) -X main.ReleaseRepo=$(RELEASE_REPO)

build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -trimpath -o bin/server ./cmd/server

generate:
	go generate ./ent
	go generate ./cmd/server

test:
	go test ./...
	golangci-lint run ./...

test-unit:
	go test -tags=unit ./...

test-integration:
	go test -tags=integration ./...

test-unit-docker:
	../tools/docker/test-backend.sh --unit

test-integration-docker:
	../tools/docker/test-backend.sh --integration

test-e2e:
	./scripts/e2e-test.sh

test-e2e-local:
	go test -tags=e2e -v -timeout=300s ./internal/integration/...
