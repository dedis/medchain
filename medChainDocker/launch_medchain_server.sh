rm -rf $GOPATH/src/github.com/dedis/cothority/
cp -rf $GOPATH/src/github.com/DPPH/cothority/ $GOPATH/src/github.com/dedis/cothority/
cp $GOPATH/src/github.com/DPPH/MedChain/medChainDocker/Dockerfile $GOPATH/MedChainDockerfile
cp $GOPATH/src/github.com/DPPH/MedChain/medChainDocker/.dockerignore $GOPATH/.dockerignore
cd $GOPATH
docker build -f MedChainDockerfile -t medchainserver .
docker run -d --name=medchainserver1 --rm -p 8989:8989 medchainserver
sleep 5s
echo "Starting the server"
curl http://localhost:8989/start &
docker attach medchainserver1
