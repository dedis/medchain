FROM golang:1.10-alpine

ENV server_path /go/src/github.com/talhaparacha/medChain/medChainServer
ENV src_path /go/src/
ENV PATH $PATH:$server_path
ENV PATH $PATH:$src_path

WORKDIR $src_path
COPY src/ .

WORKDIR $server_path
RUN go build .

ENTRYPOINT ./medChainServer -test

EXPOSE 8989
