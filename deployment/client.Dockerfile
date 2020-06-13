FROM golang:1.13 as build

ARG BUILD_TAG=master
ARG ldflags="-s -w -X main.gitTag=unknown"
RUN go get go.dedis.ch/cothority
RUN cd /go/src/go.dedis.ch/cothority && git checkout $BUILD_TAG && GO111MODULE=on go install -ldflags="$ldflags" ./byzcoin/bcadmin && cd /go


COPY ./ /src
WORKDIR /src

# get dependencies
RUN go get -v -d ./...

# compile and install medchain binary
# CGO_ENABLED=0 in order to be able to run from alpine
RUN CGO_ENABLED=0 go build -v ./cmd/medchain-cli-client/... && \ 
CGO_ENABLED=0 go install -v ./cmd/medchain-cli-client/... 

# -------------------------------------------
FROM golang:1.13-alpine as release

# run time environment variables
ENV NODE_IDX="0" \
    MEDCHAIN_LOG_LEVEL="5" \
    MEDCHAIN_TIMEOUT_SECONDS="600"\
    MEDCHAIN_GROUP_FILE_PATH="/medchain-config/public.toml"\
    MEDCHAIN_NODES_ADDRESS="/medchain-config/public.toml"\
    MEDCHAIN_CONF_DIR="/medchain-config"



COPY --from=build /go/bin/medchain-cli-client /go/bin/
COPY --from=build /go/bin/bcadmin /usr/local/bin/

COPY deployment/docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh && \
    apk add --no-cache bash

VOLUME "$MEDCHAIN_CONF_DIR"
