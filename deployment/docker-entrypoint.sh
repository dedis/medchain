#!/usr/bin/env bash
set -Eeuo pipefail

# export environment variables
export  MEDCHAIN_KEY_FILE_PATH="$MEDCHAIN_CONF_DIR/srv$NODE_IDX-private.toml" \
       

# run medchain
if [[ "$1" = "medchain-server" ]]; then
if [[ $# -eq 1 ]]; then
    ARGS="-d $MEDCHAIN_LOG_LEVEL server -c $MEDCHAIN_KEY_FILE_PATH"
else
    ARGS=$@
fi

exec medchain-server ${ARGS}
fi

if [[ "$1" = "medchain-cli-client" ]]; then
if [[ $# -eq 1 ]]; then
    ARGS="-h"
else
    ARGS=$@
fi

exec /bin/bash 
fi
