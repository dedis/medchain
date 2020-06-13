#!/usr/bin/env bash
set -e

# A POSIX variable
OPTIND=1         # Reset in case getopts has been used previously in the shell.

# Initialize our own variables:
verbose=0
nbr_nodes=3
base_port=7770
base_ip=localhost
data_dir=.
show_all="true"
show_time="false"
single=""
while getopts "h?v:n:p:i:d:qftsc" opt; do
    case "$opt" in
    h|\?)
        echo "Allowed arguments:

        -h help
        -v verbosity level: none (0) - full (5)
        -t show timestamps on logging
        -c show logs in color
        -n number of nodes (3)
        -p port base in case of new configuration (7000)
        -i IP in case of new configuration (localhost)
        -d data dir to store private keys, databases and logs (.)
        -q quiet all non-leader nodes
        -s don't start failing nodes again
        -f flush databases and start from scratch"
        exit 0
        ;;
    v)  verbose=$OPTARG
        ;;
    n)  nbr_nodes=$OPTARG
        ;;
    p)  base_port=$OPTARG
        ;;
    i)  base_ip=$OPTARG
        ;;
    d)  data_dir=$OPTARG
        ;;
    q)  show_all=""
        ;;
    f)  flush="yes"
        ;;
    t)  DEBUG_TIME="true"
        export DEBUG_TIME
        ;;
    s)  single="true"
        ;;
    c)  export DEBUG_COLOR=true
        ;;
    esac
done

shift $((OPTIND-1))

[ "${1:-}" = "--" ] && shift

MEDCHAIN_BIN="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"/medchain-server
if [ ! -x $MEDCHAIN_BIN ]; then
	echo "No medchain-server executable found. Use \"go build\" to make it."
	exit 1
fi

mkdir -p $data_dir
cd $data_dir
export DEBUG_TIME=true
if [ "$flush" ]; then
  echo "Flushing databases"
  rm -f *db
fi

rm -f group.toml
mkdir -p log
touch running
for n in $( seq $nbr_nodes -1 1 ); do
  mc=mc$n
  PORT=$(($base_port + 2 * n - 2))
  if [ ! -d $mc ]; then
    echo -e "$base_ip:$PORT\nMedchain_$n\n$mc" | $MEDCHAIN_BIN server setup
  fi
  (
    LOG=log/medchain_${mc}_$PORT
    SHOW=$( [ "$n" -eq 1 -o "$show_all" ] && echo "showing" || echo "" )
    export MEDCHAIN_SERVICE_PATH=$(pwd)
    while [[ -f running ]]; do
      echo "Starting medchain-server $LOG"
      if [[ "$SHOW" ]]; then
        $MEDCHAIN_BIN -d $verbose  server -c $mc/private.toml 2>&1 | tee $LOG-$(date +%y%m%d-%H%M).log
      else
        $MEDCHAIN_BIN -d $verbose  server -c $mc/private.toml > $LOG-$(date +%y%m%d-%H%M).log 2>&1
      fi
      if [[ "$single" ]]; then
        echo "Will not restart medchain-server in single mode."
        exit
      fi
      sleep 1
    done
  ) &
  cat $mc/public.toml >> group.toml
  # Wait for LOG to be initialized
  sleep 1
done

trap ctrl_c INT

function ctrl_c() {
  rm running
  pkill medchain-server
}

while true; do
  sleep 1;
done
