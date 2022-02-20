.PHONY: deps build run test shorten-lines

APP=rosetta-ergo
PACKAGE=github.com/ross-weir/${APP}
PACKAGE_VER=$(shell cat ./VERSION)
LDFLAGS=-ldflags "-X '${PACKAGE}/pkg/config.Version=${PACKAGE_VER}'"
TEST_SCRIPT=go test ./pkg/...

GOLINES_INSTALL=go install github.com/segmentio/golines@latest
GOLINES_CMD=golines

ERGO_NETWORK=testnet

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

docker-build:
	docker build \
		-t ${APP}:${PACKAGE_VER} \
		.

docker-run:
	docker run -it \
		-v ${CURDIR}/data/${ERGO_NETWORK}:/data \
		-e ERGO_ROSETTA_PORT=8080 \
		-e ERGO_NETWORK=${ERGO_NETWORK} \
		-e ERGO_ROSETTA_MODE=ONLINE \
		-p 8080:8080 \
		-p 9030:9030 \
		-p 9020:9020 \
		-p 9052:9052 \
		${APP}:${PACKAGE_VER}
