.PHONY: deps build run test

APP=github.com/ross-weir/rosetta-ergo
APP_VER=$(shell cat ./VERSION)
LDFLAGS=-ldflags "-X '${APP}/configuration.Version=${APP_VER}'"
TEST_SCRIPT=go test ${GO_PACKAGES}

deps:
	go get ./...

build:
	go build ${LDFLAGS}

run:
	go run ${LDFLAGS} main.go run

test:
	${TEST_SCRIPT}
