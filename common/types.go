package common

import (
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
	"time"
)

type Connection interface {
	GetWs() *websocket.Conn
	SetWs(conn *websocket.Conn)
	Close()
	GetLock() *sync.RWMutex
	Start()
	IdleSince() time.Time
	CloseWithOutLock()
	GetStatus() int
}

type ConnectionType int

const (
	Read ConnectionType = iota
	Write
)

// Status of a Connection
const (
	CONNECTING = iota
	IDLE
	BUSY
	CLOSED
)

type ConnectionPool interface {
	Offer(connection *WriteConnection)
	Remove(conn Connection)
	GetHttpClient() *http.Client
	GetLock() *sync.RWMutex
}
