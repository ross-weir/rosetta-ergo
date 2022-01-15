.PHONY: deps build run test shorten-lines

APP=github.com/ross-weir/rosetta-ergo
APP_VER=$(shell cat ./VERSION)
LDFLAGS=-ldflags "-X '${APP}/configuration.Version=${APP_VER}'"
TEST_SCRIPT=go test ${GO_PACKAGES}

GO_PACKAGES=./cmd/... ./configuration/... ./ergo/... ./services/... ./indexer/...
GO_FOLDERS=$(shell echo ${GO_PACKAGES} | sed -e "s/\.\///g" | sed -e "s/\/\.\.\.//g")

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

shorten-lines:
	${GOLINES_INSTALL}
	${GOLINES_CMD} -w --shorten-comments ${GO_FOLDERS}
