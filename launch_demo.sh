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
cd medChainServer
go build
./medChainServer -conf="conf/test_conf.json" -signing_service="http://localhost:8383" > ../server.out &
sleep 2s
curl http://localhost:8989/start &
cd ../signingService
./fresh_db.sh
go build
./signingService -port="8383" > ../service.out &
cd ../medChainAdmin
go build
./medChainAdmin -port="6161" -medchain_url="http://localhost:8989" > ../signer.out &
#function called by trap
sleep infinity
