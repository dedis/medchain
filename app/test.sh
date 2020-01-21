#!/usr/bin/env bash

DBG_TEST=1
DBG_SRV=2
export DEBUG_LVL=2
export BC_WAIT=true
export  GO111MODULE=on

# Use 3 servers, use all of them, don't leave one down.
NBR=3
NBR_SERVERS_GROUP=$NBR

. ../../../go.dedis.ch/cothority/libtest.sh

# Use a local config dir.
mc="./app -c ."

main(){
	build $APPDIR/../../../go.dedis.ch/cothority/byzcoin/bcadmin
	startTest
	export  GO111MODULE=auto
	buildConode github.com/medchain/contract

	# This must succeed before any others will work.
	run testMedchain

	stopTest
}

testMedchain(){
	##### setup phase
	rm -f *.cfg
	runCoBG 1 2 3
	runGrepSed "export BC=" "" ./bcadmin -c . create --roster public.toml --interval .5s
	eval "$SED"
	[ -z "$BC" ] && exit 1
	
	KEY=$(./app -c . key)
	echo $KEY

	./bcadmin debug counters bc*cfg key*cfg
	testOK ./bcadmin -c . darc rule -rule spawn:queryContract --identity "$KEY"
	./bcadmin debug counters bc*cfg key*cfg
	testOK ./bcadmin -c . darc rule -rule invoke:queryContract.update --identity "$KEY"
    ./bcadmin debug counters bc*cfg key*cfg
	testOK ./bcadmin -c . darc rule -rule invoke:queryContract.verifystatus --identity "$KEY"

	runGrepSed "export QUERY=" "" $mc create -sign "$KEY"
	eval "$SED"
	[ -z "$QUERY" ] && exit 1
	
	##### testing phase
	echo 'Testing Phase'
	testOK $mc query -id 'query1' -stat 'Submitted' -w 10 -sign "$KEY"
	testOK $mc query -stat 'Authorized' -w 10 -sign "$KEY"
	echo ghi | testOK $mc query -w 10 -sign "$KEY"
	seq 10 | testOK $mc query -id seq100 -w 10 -sign "$KEY"

	# testGrep "abc" $mc search -t test
	# testCountLines 13 $mc search

	# testCountLines 0 $mc search -t test -from '0s ago'
	# # The first form of relative date is for MacOS, the second for Linux.
	# testCountLines 0 $mc search -t test -from '1h ago' -to `date -v -1d +%Y-%m-%d || date -d yesterday +%Y-%m-%d`
	# testCountLines 1 $mc search -t test -to `date -v +1d +%Y-%m-%d || date -d tomorrow +%Y-%m-%d`
}

main
