#!/usr/bin/env bash

docker run --restart always -d -p 7770-7771:7770-7771 --name conode1 -v ~/conode_data:/conode_data dedis/conode:latest
docker run --restart always -d -p 7772-7773:7770-7771 --name conode2 -v ~/conode_data:/conode_data dedis/conode:latest
docker run --restart always -d -p 7774-7775:7770-7771 --name conode3 -v ~/conode_data:/conode_data dedis/conode:latest
