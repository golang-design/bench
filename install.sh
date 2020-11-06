#!/bin/bash

set -e

BINPATH="$(go env GOBIN)"
if [[ -z "$BINPATH" ]]; then
    BINPATH="$(go env GOPATH)/bin"
fi
BIN="$BINPATH/bench"
if [[ ! -x "$BIN" ]]; then
    echo "bench binary $BIN does not exist." 2>&1
    echo "Please run go install golang.design/x/bench" 2>&1
    exit 1
fi

echo "Installing $BIN to /usr/bin" 1>&2
sudo install "$BIN" /usr/bin/bench

start="-b /usr/bin/bench -daemon"
starttype=
if [[ -d /etc/init ]]; then
    echo "Installing init script for Upstart" 1>&2
    sudo install -m 0644 init/upstart/bench.conf /etc/init/
    start="service bench start"
    starttype=" (using Upstart)"
fi
if [[ -d /etc/systemd ]]; then
    echo "Installing service for systemd" 1>&2
    sudo install -m 0644 init/systemd/bench.service /etc/systemd/system
    sudo systemctl enable --quiet bench.service
    start="systemctl start bench.service"
    starttype=" (using systemd)"
fi

if /usr/bin/bench -list >/dev/null 2>&1; then
    echo "Not starting bench daemon (already running)" 1>&2
else
    echo "Starting bench daemon$starttype" 1>&2
    sudo $start
fi