package server

import (
	"fmt"
	"github.com/zerok-ai/zk-wsp/common"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Pool handles all writeConnections from the peer.
type Pool struct {
	server   *Server
	clientId string

	idleSize   int
	httpClient *http.Client

	writeConnections []*common.WriteConnection
	readConnections  []*common.ReadConnection
	idle             chan *common.WriteConnection

	done bool
	lock sync.RWMutex
}

// NewPool creates a new Pool
func NewPool(server *Server, id string) *Pool {
	p := new(Pool)
	p.server = server
	p.clientId = id
	p.readConnections = make([]*common.ReadConnection, 0)
	p.writeConnections = make([]*common.WriteConnection, 0)
	p.idle = make(chan *common.WriteConnection, server.Config.PoolMaxSize)
	p.httpClient = &http.Client{}
	return p
}

// AddConnection adds a new connection to the pool
func (pool *Pool) AddConnection(ws *websocket.Conn, connectionType common.ConnectionType) {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	// Ensure we never add a connection to a pool we have garbage collected
	if pool.done {
		return
	}

	log.Printf("Adding new connection to pool from %s and type %d.\n", pool.clientId, connectionType)
	switch connectionType {
	case common.Read:
		connection := common.NewReadConnection(pool, common.IDLE)
		connection.SetWs(ws)
		pool.readConnections = append(pool.readConnections, connection)
		go connection.Start()
	case common.Write:
		connection := common.NewWriteConnection(pool, common.IDLE)
		connection.SetWs(ws)
		pool.writeConnections = append(pool.writeConnections, connection)
		go connection.Start()
	default:
		fmt.Println("Object is of unknown type")
	}
}

// Offer offers an idle connection to the server.
func (pool *Pool) Offer(connection *common.WriteConnection) {
	pool.idle <- connection
	fmt.Println("Idle channel length is ", len(pool.idle))
}

// Clean removes dead connection from the pool
// Look for dead connection in the pool
// This MUST be surrounded by pool.lock.Lock()
func (pool *Pool) Clean() {
	idle := 0
	for _, connection := range pool.readConnections {
		idle = pool.CleanConnection(connection, idle)
	}

	idle = 0
	for _, connection := range pool.writeConnections {
		idle = pool.CleanConnection(connection, idle)
	}
}

func (pool *Pool) CleanConnection(connection common.Connection, idle int) int {
	lock := connection.GetLock()
	lock.Lock()
	defer lock.Unlock()
	if connection.GetStatus() == common.IDLE {
		idle++
		if idle > pool.idleSize {
			if int(time.Now().Sub(connection.IdleSince()).Seconds()) > pool.server.Config.IdleTimeout {
				switch connection.(type) {
				case *common.ReadConnection:
					fmt.Println("Closing connection due to timeout and connType read")
				case *common.WriteConnection:
					fmt.Println("Closing connection due to timeout and connType write")
				}
				connection.CloseWithOutLock()
				pool.RemoveWithoutLock(connection)
			}
		}
	}
	return idle
}

// IsEmpty clean the pool and return true if the pool is empty
func (pool *Pool) IsEmpty() bool {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	pool.Clean()
	return len(pool.writeConnections) == 0 && len(pool.readConnections) == 0
}

// Shutdown closes every writeConnections in the pool and cleans it
func (pool *Pool) Shutdown() {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	pool.done = true

	for _, connection := range pool.writeConnections {
		connection.Close()
	}

	for _, connection := range pool.readConnections {
		connection.Close()
	}

	pool.RemoveAllConnections()

}

func (pool *Pool) GetHttpClient() *http.Client {
	return pool.httpClient
}

func (pool *Pool) GetLock() *sync.RWMutex {
	return &pool.lock
}

func (pool *Pool) RemoveWithoutLock(conn common.Connection) {
	switch c := (conn).(type) {
	case *common.ReadConnection:
		fmt.Println("Removing read connection from pool")
		filtered := make([]*common.ReadConnection, 0)
		for _, i := range pool.readConnections {
			if c != i {
				filtered = append(filtered, i)
			}
		}
		pool.readConnections = filtered
		fmt.Println("Read connections length in server is ", len(pool.readConnections))
	case *common.WriteConnection:
		fmt.Println("Removing write connection from pool")
		filtered := make([]*common.WriteConnection, 0)
		for _, i := range pool.writeConnections {
			if c != i {
				filtered = append(filtered, i)
			}
		}
		pool.writeConnections = filtered
		fmt.Println("Write connections length in server is ", len(pool.writeConnections))
	default:
		fmt.Println("Object is of unknown type")
	}
}

// Remove a connection from the pool
func (pool *Pool) Remove(conn common.Connection) {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	pool.RemoveWithoutLock(conn)
}

func (pool *Pool) RemoveAllConnections() {
	pool.readConnections = make([]*common.ReadConnection, 0)
	pool.writeConnections = make([]*common.WriteConnection, 0)
}

func (pool *Pool) GetIdleWriteConnection() *common.WriteConnection {
	connection, err := common.GetValueWithTimeout(pool.idle, pool.server.Config.GetTimeout())

	if err == nil && connection.Take() {
		return connection
	}
	fmt.Println("Error getting idle connection ", err)
	return nil
}
