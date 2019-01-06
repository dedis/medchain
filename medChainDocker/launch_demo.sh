#!/bin/bash
do_this_on_ctrl_c(){
    echo "Exiting the loop"
    pkill -f signingService
    pkill -f medChainAdmin
    pkill -f medChainServer
    date
    exit 0
}
trap 'do_this_on_ctrl_c' INT
cd ..
cd medChainServer
go build
./medChainServer -conf="conf/test_conf.json" -signing_service="http://localhost:8383" > ../medChainDocker/medchain_server.out &
sleep 2s
curl http://localhost:8989/start &
cd ../signingService/data/
./fresh_db.sh
cd ..
go build
./signingService -port="8383" > ../medChainDocker/signing_service.out &
cd ../medChainAdmin
go build
./medChainAdmin -port="6161" -medchain_url="http://localhost:8989" > ../medChainDocker/local_signer.out &
sleep infinity
