#!/usr/bin/env bash

docker run -it --rm -p 7770-7771:7770-7771 --name conode1 -v ~/conode_data:/conode_data dedis/conode:latest ./conode setup
docker run -it --rm -p 7772-7773:7772-7773 --name conode2 -v ~/conode_data:/conode_data dedis/conode:latest ./conode setup
docker run -it --rm -p 7774-7775:7774-7775 --name conode3 -v ~/conode_data:/conode_data dedis/conode:latest ./conode setup