# zerok websocket proxy server
The websocket proxy server maintains a pool of connections between server and client. The proxy server supports full duplex communications. 

# Usage

For sending request to server:

curl -H 'X-PROXY-DESTINATION: http://localhost:8091/hello' -H 'X-CLIENT-ID: 9e7f784c-b453-4ef4-61f4-dc389dd0bbe1' http://127.0.0.1:8989/request

For sending request to client

curl -H 'X-PROXY-DESTINATION: http://localhost:8091/hello' http://127.0.0.1:8987/request