FROM golang:1.13 as build
ENV GO111MODULE=on

ARG BUILD_TAG=master
ARG ldflags="-s -w -X main.gitTag=unknown"
RUN GO111MODULE=off go get go.dedis.ch/cothority
RUN cd /go/src/go.dedis.ch/cothority && git checkout $BUILD_TAG && GO111MODULE=on go install -ldflags="$ldflags" ./byzcoin/bcadmin && cd /go

COPY ./ /src
WORKDIR /src 

# get dependencies
RUN go get -v -d ./...
RUN go get -v go.dedis.ch/cothority/v3/byzcoin/bcadmin

# compile and install medchain binary
# CGO_ENABLED=0 in order to be able to run from alpine
RUN CGO_ENABLED=0 go build -v ./cmd/medchain-server/... && \ 
CGO_ENABLED=0 go install -v ./cmd/medchain-server/... 

# -------------------------------------------
FROM golang:1.13 as release

# run time environment variables
ENV NODE_IDX="1" \
    MEDCHAIN_LOG_LEVEL="5" \
    MEDCHAIN_TIMEOUT_SECONDS="600"\
    MEDCHAIN_GROUP_FILE_PATH="/medchain-config/group.toml"\
    MEDCHAIN_NODES_ADDRESS="/medchain-config/group.toml"\
    MEDCHAIN_CONF_DIR="/medchain-config"



COPY --from=build /go/bin/medchain-server /go/bin/
COPY --from=build /go/bin/bcadmin  /go/bin/


COPY deployment/docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh 

VOLUME "$MEDCHAIN_CONF_DIR"
ENTRYPOINT docker-entrypoint.sh medchain-server