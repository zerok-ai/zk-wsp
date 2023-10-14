#!/bin/sh

while [[ $# -gt 0 ]]; do
    case "$1" in
        -amd64)
            amd64="$2"
            shift 2
            ;;
        -arm64)
            arm64="$2"
            shift 2
            ;;
        -c)
            config="$2"
            shift 2
            ;;
    esac
done

arch=$(uname -m)

ls

if [ "$arch" = "x86_64" ] || [ "$arch" = "amd64" ]; then
    echo "This is an AMD64 system."
    ./$amd64 -c $config
elif [ "$arch" = "aarch64" ] || [ "$arch" = "arm64" ]; then
    echo "This is an ARM64 system."
    ./$arm64 -c $config
else
    echo "Unsupported Architecture: $arch"
fi