# zerok websocket proxy server
### TODO: Add more description here.
The websocket proxy server maintains a pool of connections between server and client. The proxy server supports full duplex communications.

### Running on the cluster

Run this command will build the server and push the updated image to docker.

```
make build-server
```

Run this command to build the client and push the updated image to docker.

```
make build-client
```

Run this command will install the server to the cluster in the current context.

```
make install-server
```

Run this command will install the client to the cluster in the current context.

```
make install-client
```

Run this commands to uninstall the client.

```
make uninstall-client
```


Run this commands to uninstall the server.

```
make uninstall-server
```


### Usage

Refer to the example below sending request to server from the client cluster. Please note that the client id here is the clusterId on zerok.

```
curl -H 'X-PROXY-DESTINATION: http://localhost:8091/hello' -H 'X-CLIENT-ID: 9e7f784c-b453-4ef4-61f4-dc389dd0bbe1' http://127.0.0.1:8989/request
```

For sending request to client from the server cluster:

```
curl -H 'X-PROXY-DESTINATION: http://localhost:8091/hello' http://127.0.0.1:8987/request
```