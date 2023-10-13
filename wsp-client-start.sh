#!/bin/sh

arch=$(uname -m)

if [ "$arch" = "x86_64" ] || [ "$arch" = "amd64" ]; then
    echo "This is an AMD64 system."
    ./myapp-amd64 -c /opt/wsp-config.yaml
elif [ "$arch" = "aarch64" ] || [ "$arch" = "arm64" ]; then
    echo "This is an ARM64 system."
    ./myapp-arm64 -c /opt/wsp-config.yaml
else
    echo "Unsupported Architecture: $arch"
fi