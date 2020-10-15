#!/usr/bin/env bash

: "${DATA_FILE:=/home/evesite/eve-apps/systemdata.yml}"
: "${BOT_BINARY:=/home/evesite/eve-apps/route-server}"

env
exec $BOT_BINARY \
    --system-data "$DATA_FILE"