#!/bin/bash

THIS_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
export ROOT_DIR="$(dirname "$THIS_DIR")"

if [ -z "$1" ]; then
    echo "Error: Missing variable"
    echo "Usage: ./build.sh <variable>"
    echo "Possible values for <variable>: server, client"
    exit 1
fi

# Retrieve the variable from the argument
variable=$1

# Check the value of the variable
if [ "$variable" == "server" ]; then
    echo "Installing wsp server"
    kubectl apply -f "$ROOT_DIR"/config/client/configmap.yaml
    kubectl apply -k "$ROOT_DIR"/k8s/client
elif [ "$variable" == "client" ]; then
    echo "installing wsp client"
    kubectl apply -f "$ROOT_DIR"/config/server/configmap.yaml
    kubectl apply -k "$ROOT_DIR"/k8s/server
else
    echo "Variable is neither server nor client"
fi