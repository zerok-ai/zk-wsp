package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	zklogger "github.com/zerok-ai/zk-utils-go/logs"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/zerok-ai/zk-wsp"
)

// WriteConnection manages a single websocket connection from the peer.
// wsp supports multiple connections from a single peer at the same time.
type WriteConnection struct {
	pool      *ConnectionPool
	ws        *websocket.Conn
	Status    int
	idleSince time.Time
	lock      sync.RWMutex
	// nextResponse is the channel of channel to wait an HTTP response.
	//
	// In advance, the `read` function waits to receive the HTTP response as a separate thread "reader".
	// (See https://github.com/hgsgtk/wsp/blob/29cc73bbd67de18f1df295809166a7a5ef52e9fa/server/connection.go#L56 )
	//
	// When a "server" thread proxies, it sends the HTTP request to the peer over the WebSocket,
	// and sends the channel of the io.Reader interface (chan io.Reader) that can read the HTTP response to the field `nextResponse`,
	// then waits until the value is written in the channel (chan io.Reader) by another thread "reader".
	//
	// After the thread "reader" detects that the HTTP response from the peer of the WebSocket connection has been written,
	// it sends the value to the channel (chan io.Reader),
	// and the "server" thread can proceed to process the rest procedures.
	nextResponse chan chan io.Reader
}

var WRITE_LOG_TAG = "WriteConnection"

func (connection *WriteConnection) GetStatus() int {
	return connection.Status
}

// NewWriteConnection returns a new WriteConnection.
func NewWriteConnection(pool ConnectionPool, status int) *WriteConnection {
	// Initialize a new WriteConnection
	c := new(WriteConnection)
	c.pool = &pool
	c.nextResponse = make(chan chan io.Reader)
	c.Status = status

	return c
}

// read the incoming message of the connection
func (connection *WriteConnection) Start() {
	// Mark that this connection is ready to use for relay
	connection.Release()
	defer func() {
		if r := recover(); r != nil {
			zklogger.Error(LOG_TAG, "Write connection crash recovered : %s", r)
		}
		zklogger.Debug(LOG_TAG, "Write connection ending.")
		(*connection.pool).Remove(connection)
		connection.Close()
	}()

	for {
		if connection.Status == CLOSED {
			break
		}

		// https://godoc.org/github.com/gorilla/websocket#hdr-Control_Messages
		//
		// We need to ensure :
		//  - no concurrent calls to ws.NextReader() / ws.ReadMessage()
		//  - only one reader exists at a time
		//  - wait for reader to be consumed before requesting the next one
		//  - always be reading on the socket to be able to process control messages ( ping / pong / close )

		// We will block here until a message is received or the ws is closed
		_, reader, err := connection.ws.NextReader()
		if err != nil {
			break
		}

		if connection.Status != BUSY {
			// We received a wild unexpected message
			break
		}

		// When it gets here, it is expected to be either a HttpResponse or a HttpResponseBody has been returned.
		//
		// Next, it waits to receive the value from the WriteConnection.proxyRequest function that is invoked in the "server" thread.
		// https://github.com/hgsgtk/wsp/blob/29cc73bbd67de18f1df295809166a7a5ef52e9fa/server/connection.go#L157
		c := <-connection.nextResponse
		if c == nil {
			// We have been unlocked by Close()
			break
		}

		// Send the reader back to WriteConnection.proxyRequest
		c <- reader

		// Wait for proxyRequest to close the channel
		// this notify that it is done with the reader
		<-c
	}
}

func (connection *WriteConnection) SendPingMessage() error {
	zklogger.Debug(WRITE_LOG_TAG, "Sending ping message to peer.")

	// Create a context with a timeout
	//TODO: Think and decide on a timeout value
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer func() {
		cancel()
		connection.Release()
	}()

	// Set the pong handler
	connection.ws.SetPongHandler(func(appData string) error {
		cancel() // Cancel the context on receiving a pong
		return nil
	})

	err := connection.ws.WriteMessage(websocket.PingMessage, []byte{})
	if err != nil {
		return err
	}

	// Wait for the pong message or context timeout
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			zklogger.Debug("Pong wait timeout")
			return fmt.Errorf("pong wait timeout")
		} else {
			// Pong received before timeout
			zklogger.Debug("Pong received")
			return nil
		}
	}
}

// Proxy a HTTP request through the Proxy over the websocket connection
func (connection *WriteConnection) ProxyRequest(w http.ResponseWriter, r *http.Request) (err error) {

	// [1]: Serialize HTTP request
	jsonReq, err := json.Marshal(wsp.SerializeHTTPRequest(r))
	if err != nil {
		return fmt.Errorf("unable to serialize request : %w", err)
	}
	// i.e.
	// {
	// 		"Method":"GET",
	// 		"URL":"http://localhost:8081/hello",
	// 		"Header":{"Accept":["*/*"],"User-Agent":["curl/7.77.0"],"X-Proxy-Destination":["http://localhost:8081/hello"]},
	//		"ContentLength":0
	// }

	// [2]: Send the HTTP request to the peer
	// Send the serialized HTTP request to the peer
	zklogger.Debug(LOG_TAG, "Sending the http request to peer.")
	if err := connection.ws.WriteMessage(websocket.TextMessage, jsonReq); err != nil {
		return fmt.Errorf("unable to write request : %w", err)
	}

	// Pipe the HTTP request body to the peer
	bodyWriter, err := connection.ws.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return fmt.Errorf("unable to get request body writer : %w", err)
	}
	if _, err := io.Copy(bodyWriter, r.Body); err != nil {
		return fmt.Errorf("unable to pipe request body : %w", err)
	}
	if err := bodyWriter.Close(); err != nil {
		return fmt.Errorf("unable to pipe request body (close) : %w", err)
	}

	// [3]: Wait the HTTP response is ready
	zklogger.Debug(LOG_TAG, "Waiting for http response.")
	responseChannel := make(chan (io.Reader))
	connection.nextResponse <- responseChannel
	responseReader, ok := <-responseChannel
	if responseReader == nil {
		if ok {
			// The value of ok is false, the channel is closed and empty.
			// See the Receiver operator in https://go.dev/ref/spec for more information.
			close(responseChannel)
		}
		return fmt.Errorf("unable to get http response reader : %w", err)
	}

	// [4]: Read the HTTP response from the peer
	// Get the serialized HTTP Response from the peer
	jsonResponse, err := io.ReadAll(responseReader)
	zklogger.Debug(LOG_TAG, "Response received is ", jsonResponse)
	if err != nil {
		close(responseChannel)
		return fmt.Errorf("unable to read http response : %w", err)
	}

	// Notify the read() goroutine that we are done reading the response
	close(responseChannel)

	// Deserialize the HTTP Response
	httpResponse := new(wsp.HTTPResponse)
	if err := json.Unmarshal(jsonResponse, httpResponse); err != nil {
		return fmt.Errorf("unable to unserialize http response : %w", err)
	}

	// Write response headers back to the client
	for header, values := range httpResponse.Header {
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}
	w.WriteHeader(httpResponse.StatusCode)

	// [5]: Wait the HTTP response body is ready
	// Get the HTTP Response body from the the peer
	// To do so send a new channel to the read() goroutine
	// to get the next message reader
	zklogger.Debug(LOG_TAG, "Waiting for http body.")
	responseBodyChannel := make(chan (io.Reader))
	connection.nextResponse <- responseBodyChannel
	responseBodyReader, ok := <-responseBodyChannel
	if responseBodyReader == nil {
		if ok {
			// If more is false the channel is already closed
			close(responseChannel)
		}
		return fmt.Errorf("unable to get http response body reader : %w", err)
	}

	// [6]: Read the HTTP response body from the peer
	// Pipe the HTTP response body right from the remote Proxy to the client
	zklogger.Debug(LOG_TAG, "Received response body.")
	if _, err := io.Copy(w, responseBodyReader); err != nil {
		close(responseBodyChannel)
		return fmt.Errorf("unable to pipe response body : %w", err)
	}

	// Notify read() that we are done reading the response body
	close(responseBodyChannel)

	connection.Release()

	return
}

// Take notifies that this connection is going to be used
func (connection *WriteConnection) Take() bool {
	connection.lock.Lock()
	defer connection.lock.Unlock()

	if connection.Status == CLOSED {
		return false
	}

	if connection.Status == BUSY {
		return false
	}

	connection.Status = BUSY
	return true
}

// Release notifies that this connection is ready to use again
func (connection *WriteConnection) Release() {
	connection.lock.Lock()
	defer connection.lock.Unlock()

	if connection.Status == CLOSED {
		return
	}

	connection.idleSince = time.Now()
	connection.Status = IDLE
	(*connection.pool).Offer(connection)
}

// Close the connection
func (connection *WriteConnection) Close() {
	connection.lock.Lock()
	defer connection.lock.Unlock()
	connection.CloseWithOutLock()
}

// Close the connection ( without lock )
func (connection *WriteConnection) CloseWithOutLock() {
	if connection.Status == CLOSED {
		return
	}

	// This one will be executed *before* lock.Unlock()
	defer func() { connection.Status = CLOSED }()

	// Unlock a possible read() wild message
	close(connection.nextResponse)
	if connection.ws != nil {
		err := connection.ws.Close()
		if err != nil {
			zklogger.Error(LOG_TAG, "Error while closing read connection : ", err)
		}
	}
}

func (connection *WriteConnection) GetWs() *websocket.Conn {
	return connection.ws
}

func (connection *WriteConnection) SetWs(conn *websocket.Conn) {
	connection.ws = conn
}

func (connection *WriteConnection) GetLock() *sync.RWMutex {
	return &connection.lock
}

func (connection *WriteConnection) IdleSince() time.Time {
	return connection.idleSince
}
