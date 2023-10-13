#!/bin/sh

ARCH=$(uname -m)
if [ "$ARCH" = "aarch64" ]; then
    ./myapp-arm64 -c /opt/wsp-config.yaml
else
    ./myapp-amd64 -c /opt/wsp-config.yaml
fi