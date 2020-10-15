#!/usr/bin/env bash

# Set sane, consistent Bash shopt settings:
set -o errexit              # Exit on any bash error
set -o nounset              # Fail when unset variable is used
set -o pipefail             # Fail on any error in a pipeline

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
  DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done
HERE="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
echo ${HERE}

rm -rf ${HERE}/eve-routes
tar vCxzf ${HERE} ${HERE}/route-server-static.tar.gz
chown -R evesite:evesite ${HERE}/eve-routes
systemctl stop route-server
cp ${HERE}/route-server.service /etc/systemd/system/
systemctl daemon-reload
rm -f ${HERE}/route-server
gunzip ${HERE}/route-server.gz
systemctl start route-server