package common

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/zerok-ai/zk-wsp"
)

// ReadConnection handle a single websocket (HTTP/TCP) connection to an Server
type ReadConnection struct {
	pool      *ConnectionPool
	ws        *websocket.Conn
	idleSince time.Time
	Status    int
	lock      sync.RWMutex
}

func (connection *ReadConnection) GetStatus() int {
	return connection.Status
}

func (connection *ReadConnection) GetLock() *sync.RWMutex {
	return &connection.lock
}

// NewReadConnection create a ReadConnection object
func NewReadConnection(pool ConnectionPool, status int) *ReadConnection {
	c := new(ReadConnection)
	c.pool = &pool
	c.Status = status
	return c
}

// the main loop it :
//   - wait to receive HTTP requests
//   - execute HTTP requests
//   - send HTTP response back
//
// There is no buffering of HTTP request/response body
// If any error occurs the connection is closed/throwed
func (connection *ReadConnection) Start() {
	defer func() {
		fmt.Println("Read connection ending.")
		connection.Close()
	}()

	fmt.Println("Read connection starting.")

	// Keep connection alive
	go func() {
		for {
			time.Sleep(30 * time.Second)
			err := connection.ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second))
			if err != nil {
				connection.Close()
			}
		}
	}()

	for {
		// Read request
		connection.Status = IDLE
		_, jsonRequest, err := connection.ws.ReadMessage()
		fmt.Println(" Received request.")
		if err != nil {
			log.Println("Unable to read request", err)
			break
		}

		connection.Status = BUSY

		// Deserialize request
		httpRequest := new(wsp.HTTPRequest)
		err = json.Unmarshal(jsonRequest, httpRequest)
		if err != nil {
			connection.error(fmt.Sprintf("Unable to deserialize json http request : %s\n", err))
			break
		}
		fmt.Println("Created http request.")

		req, err := wsp.UnserializeHTTPRequest(httpRequest)
		if err != nil {
			connection.error(fmt.Sprintf("Unable to deserialize http request : %v\n", err))
			break
		}

		log.Printf("[%s] %s", req.Method, req.URL.String())

		// Pipe request body
		_, bodyReader, err := connection.ws.NextReader()
		if err != nil {
			log.Printf("Unable to get response body reader : %v", err)
			break
		}
		req.Body = io.NopCloser(bodyReader)
		fmt.Println("Received request body.")

		// Execute request
		httpClient := (*connection.pool).GetHttpClient()
		resp, err := httpClient.Do(req)
		if err != nil {
			err = connection.error(fmt.Sprintf("Unable to execute request : %v\n", err))
			if err != nil {
				break
			}
			continue
		}
		fmt.Println("Done executing request.")

		// Serialize response
		jsonResponse, err := json.Marshal(wsp.SerializeHTTPResponse(resp))
		if err != nil {
			err = connection.error(fmt.Sprintf("Unable to serialize response : %v\n", err))
			if err != nil {
				break
			}
			continue
		}

		fmt.Println("Writing response.")
		// Write response
		err = connection.ws.WriteMessage(websocket.TextMessage, jsonResponse)
		if err != nil {
			log.Printf("Unable to write response : %v", err)
			break
		}

		fmt.Println("Writing response body.")
		// Pipe response body
		bodyWriter, err := connection.ws.NextWriter(websocket.BinaryMessage)
		if err != nil {
			log.Printf("Unable to get response body writer : %v", err)
			break
		}
		_, err = io.Copy(bodyWriter, resp.Body)
		if err != nil {
			log.Printf("Unable to get pipe response body : %v", err)
			break
		}
		err = bodyWriter.Close()
		if err != nil {
			fmt.Println("Error while closing bodyWriter in read connection : ", err)
		}
	}
}

func (connection *ReadConnection) error(msg string) (err error) {
	resp := wsp.NewHTTPResponse()
	resp.StatusCode = 527

	log.Println(msg)

	resp.ContentLength = int64(len(msg))

	// Serialize response
	jsonResponse, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Unable to serialize response : %v", err)
		return
	}

	// Write response
	err = connection.ws.WriteMessage(websocket.TextMessage, jsonResponse)
	if err != nil {
		log.Printf("Unable to write response : %v", err)
		return
	}

	// Write response body
	err = connection.ws.WriteMessage(websocket.BinaryMessage, []byte(msg))
	if err != nil {
		log.Printf("Unable to write response body : %v", err)
		return
	}

	return
}

// Close the ws/tcp connection and remove it from the pool
func (connection *ReadConnection) Close() {

	lock := (*connection.pool).GetLock()
	lock.Lock()
	defer lock.Unlock()
	connection.CloseWithOutLock()
}

func (connection *ReadConnection) CloseWithOutLock() {
	if connection.Status == CLOSED {
		return
	}
	defer func() { connection.Status = CLOSED }()
	(*connection.pool).Remove(connection)
	err := connection.ws.Close()
	if err != nil {
		fmt.Println("Error while closing read connection : ", err)
	}
}

func (connection *ReadConnection) GetWs() *websocket.Conn {
	return connection.ws
}

func (connection *ReadConnection) SetWs(conn *websocket.Conn) {
	connection.ws = conn
}

func (connection *ReadConnection) IdleSince() time.Time {
	return connection.idleSince
}