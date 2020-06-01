#!/usr/bin/env bash

DBG_TEST=1
DBG_SRV=2
export DEBUG_LVL=2
export BC_WAIT=true
export  GO111MODULE=on

# Use 3 servers, use all of them, don't leave one down.
NBR=3
NBR_SERVERS_GROUP=$NBR

./libtest.sh

# Use a local config dir.
mccl="./mcadmin -c ."

main(){
	build $APPDIR/../../../go.dedis.ch/cothority/byzcoin/bcadmin
	startTest
	export  GO111MODULE=auto
	buildConode github.com/medchain/services/medchain-cli-client

	# This must succeed before any others will work.
	run testMedChain

	stopTest
}

testMedChain(){
	##### setup phase
	rm -f *.cfg

	echo $PWD
	# run medchain nodes in background (a subshell)
	runCoBG 1 2 3
	runGrepSed "export BC=" "" ./bcadmin -c . create --roster public.toml --interval .5s
	eval "$SED"
	[ -z "$BC" ] && exit 1
	
	KEY=$(./mcAdmin -c . key)
	echo "Key is" $KEY
	sleep 5
	runGrepSed "export MC=" "" $mc create -sign "$KEY"
	eval "$SED"
	[ -z "$MC" ] && exit 1

	
 	./bcadmin debug counters bc*cfg key*cfg
	testOK ./mc -c . darc rule -rule spawn:queryContract -identity "$KEY"
	./bcadmin debug counters bc*cfg key*cfg
	testOK ./mc -c . darc rule -rule invoke:queryContract.update -identity "$KEY"
    ./bcadmin debug counters bc*cfg key*cfg
	testOK ./mc -c . darc rule -rule invoke:queryContract.verifystatus -identity "$KEY"
	./bcadmin debug counters bc*cfg key*cfg
	testOK ./mc -c . darc rule -rule _name:queryContract -identity "$KEY"
	testOK ./mc -c . darc add -name "A" -identity "$KEY" -out_id "out_id.txt" -out_key "out_key.txt"
	
	
	##### testing phase
	echo 'Testing Phase'

	testOK $mcAdmin query -id 'wsdf65k80h:A:patient_list' -stat 'Submitted' -w 10 -sign "$KEY" 
	echo $KEY
	testOK $mcAdmin query -id 'wqdesf547z:B:patient_list' -stat 'Submitted' -w 10 -sign "$KEY"
	echo ghi | testOK $mc query -w 10 -sign "$KEY"
	seq 10 | testOK $mcAdmin query -id seq100 -w 10 -sign "$KEY"

}

main