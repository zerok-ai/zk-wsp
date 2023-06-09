#!/bin/bash


LOCATION="us-west1"
PROJECT_ID="zerok-dev"
REPOSITORY="stage"

SERVER_VERSION="0.0.2"
SERVER_IMAGE="zk-wsp-server"
SERVER_ART_Repo_URI="$LOCATION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY/$SERVER_IMAGE"
SERVER_IMG="$SERVER_ART_Repo_URI:$SERVER_VERSION"

CLIENT_VERSION="0.0.2"
CLIENT_IMAGE="zk-wsp-client"
CLIENT_ART_Repo_URI="$LOCATION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY/$CLIENT_IMAGE"
CLIENT_IMG="$CLIENT_ART_Repo_URI:$CLIENT_VERSION"

# Check if the variable is provided as an argument
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
    echo "Building wsp server"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o wsp_server cmd/wsp_server/main.go
    docker build . -t "$SERVER_IMG" --build-arg APP_FILE=wsp_server
    docker push "$SERVER_IMG"
elif [ "$variable" == "client" ]; then
    echo "Building wsp client"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o wsp_client cmd/wsp_client/main.go
    docker build . -t "$CLIENT_IMG" --build-arg APP_FILE=wsp_client
    docker push "$CLIENT_IMG"
else
    echo "Variable is neither server nor client"
fi


