FROM golang:1.15 as builder

WORKDIR /app/

COPY . .
RUN go build -o /conode ./conode/

FROM debian:stretch-slim
RUN apt update; apt install -y procps ca-certificates; apt clean

WORKDIR /root/

RUN mkdir /conode_data
RUN mkdir -p .local/share .config
RUN ln -s /conode_data .local/share/conode
RUN ln -s /conode_data .config/conode

COPY --from=builder /conode .
COPY conode/run_nodes.sh .

EXPOSE 7770 7771

CMD ["/root/conode", "-d", "2", "server"]
