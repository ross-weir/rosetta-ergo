.PHONY: deps build run test shorten-lines

APP=github.com/ross-weir/rosetta-ergo
APP_VER=$(shell cat ./VERSION)
LDFLAGS=-ldflags "-X '${APP}/pkg/config.Version=${APP_VER}'"
TEST_SCRIPT=go test pkg

GOLINES_INSTALL=go install github.com/segmentio/golines@latest
GOLINES_CMD=golines

deps:
	go get ./...

build:
	go build ${LDFLAGS}

run:
	go run ${LDFLAGS} main.go run

test:
	${TEST_SCRIPT}

lint:
	golangci-lint run

shorten-lines:
	${GOLINES_INSTALL}
	${GOLINES_CMD} -w --shorten-comments pkg cmd
