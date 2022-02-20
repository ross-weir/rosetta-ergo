# Rosetta requirements for docker images: https://www.rosetta-api.org/docs/node_deployment.html#dockerfile-expectations
# openjdk/sbt images are debian based, we must use ubuntu according to the rosetta docker requirements.

ARG JAVA_VERSION=11.0.12-open

FROM ubuntu:20.04 as node-builder

ARG JAVA_VERSION

RUN mkdir -p /app \
    && chown -R nobody:nogroup /app
WORKDIR /app

RUN apt-get update \
    && apt-get install -y curl git unzip zip \
    && curl -s "https://get.sdkman.io" | bash

RUN git clone https://github.com/ergoplatform/ergo.git

# So we can run `source` command
RUN rm /bin/sh && ln -s /bin/bash /bin/sh
RUN cd ergo \
    && git checkout v4.0.22 \
    && source "$HOME/.sdkman/bin/sdkman-init.sh" \
    && sdk install java "$JAVA_VERSION" \
    && sdk install sbt 1.5.2 \
    && sbt update \
    && sbt assembly \
    && mv `find target/scala-*/stripped/ -name ergo-*.jar` ergo.jar

FROM ubuntu:20.04 as rosetta-builder

RUN mkdir -p /app \
    && chown -R nobody:nogroup /app
WORKDIR /app

RUN apt-get update && apt-get install -y curl make gcc g++
ENV GOLANG_VERSION 1.17.6
ENV GOLANG_DOWNLOAD_SHA256 231654bbf2dab3d86c1619ce799e77b03d96f9b50770297c8f4dff8836fc8ca2
ENV GOLANG_DOWNLOAD_URL https://go.dev/dl/go$GOLANG_VERSION.linux-amd64.tar.gz

RUN curl -fsSL "$GOLANG_DOWNLOAD_URL" -o golang.tar.gz \
    && echo "$GOLANG_DOWNLOAD_SHA256  golang.tar.gz" | sha256sum -c - \
    && tar -C /usr/local -xzf golang.tar.gz \
    && rm golang.tar.gz

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

COPY . src 
RUN cd src \
    && make build \
    && cd .. \
    && mv src/rosetta-ergo /app/rosetta-ergo \
    && rm -rf src 

FROM ubuntu:20.04

ARG JAVA_VERSION

ENV ERGO_ROSETTA_DATA_DIR /data
ENV ERGO_CONFIG_DIR /app/configs
ENV ERGO_ROSETTA_PORT 8080

RUN mkdir -p /app \
    && chown -R nobody:nogroup /app \
    && mkdir -p ${ERGO_ROSETTA_DATA_DIR} \
    && chown -R nobody:nogroup ${ERGO_ROSETTA_DATA_DIR}

# So we can run `source` command
RUN rm /bin/sh && ln -s /bin/bash /bin/sh
RUN apt-get update \
    && apt-get install -y curl git unzip zip \
    && curl -s "https://get.sdkman.io" | bash \
    && source "$HOME/.sdkman/bin/sdkman-init.sh" \
    && sdk install java "$JAVA_VERSION"

ENV JAVA_HOME /root/.sdkman/candidates/java/current

WORKDIR /app

COPY --from=node-builder /app/ergo/ergo.jar /app/ergo.jar
COPY --from=rosetta-builder /app/* /app/
COPY configs /app/configs

# mainnet node network
EXPOSE 9030
# testnet node network
EXPOSE 9020
# rosetta webserver
EXPOSE 8080

RUN chmod -R 755 /app/*

ENTRYPOINT ["/app/rosetta-ergo"]
CMD ["run", "--start-node"]
