FROM golang:1.13 as build

COPY ./ /src
WORKDIR /src

# get dependencies
RUN go get -v -d ./...

# compile and install medchain binary
# CGO_ENABLED=0 in order to be able to run from alpine
RUN CGO_ENABLED=0 go build -v ./... && \
    CGO_ENABLED=0 go install -v ./... && \
    mv /go/bin/app /go/bin/medchain

# -------------------------------------------
FROM golang:1.13-alpine as release

# run time environment variables
ENV MEDCHAIN_NODE_IDX="0" \
    MEDCHAIN_LOG_LEVEL="1" \
    MEDCHAIN_TIMEOUT_SECONDS="600"\
    MEDCHAIN_GROUP_FILE_PATH="/medchain-config/public.toml"\
    MEDCHAIN_NODES_ADDRESS="/medchain-config/public.toml"\
    CONF_DIR="/medchain-config"



COPY --from=build /go/bin/medchain /go/bin/
COPY --from=builder /go/bin/conode .
COPY --from=builder /go/bin/bcadmin /go/bin/el /go/bin/scmgr /go/bin/mc /usr/local/bin/

COPY deployment/docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh && \
    apk add --no-cache bash

VOLUME "$CONF_DIR"
ENTRYPOINT ["docker-entrypoint.sh"]