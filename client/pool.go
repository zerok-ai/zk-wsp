package client

import (
	"context"
	zklogger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-wsp/common"
	"net/http"
	"sync"
	"time"
)

var POOL_LOG_TAG = "ClientPool"

// Pool manage a pool of connection to a remote Server
type Pool struct {
	Client           *Client
	Target           *TargetConfig
	secretKey        string
	httpClient       *http.Client
	readConnections  []*common.ReadConnection
	writeConnections []*common.WriteConnection
	lock             sync.RWMutex
	idle             chan *common.WriteConnection
	ticker           *time.Ticker
	done             chan struct{}
	retryInterval    time.Duration
}

// NewPool creates a new Pool
func NewPool(client *Client, target *TargetConfig) (pool *Pool) {
	pool = new(Pool)
	pool.Client = client
	pool.httpClient = client.client
	pool.Target = target
	pool.readConnections = make([]*common.ReadConnection, 0)
	pool.writeConnections = make([]*common.WriteConnection, 0)
	pool.idle = make(chan *common.WriteConnection, client.Config.PoolMaxSize)
	pool.done = make(chan struct{})
	pool.ticker = time.NewTicker(time.Second * time.Duration(client.Config.DefaultRetryInterval))
	pool.retryInterval = time.Second * time.Duration(client.Config.DefaultRetryInterval)
	return
}

// Start connect to the remote Server
func (pool *Pool) Start(ctx context.Context) {
	pool.startInternal(ctx)
	go func() {
		defer pool.ticker.Stop()
	L:
		for {
			select {
			case <-pool.done:
				break L
			case <-pool.ticker.C:
				pool.startInternal(ctx)
			}
		}
	}()
}

func (pool *Pool) startInternal(ctx context.Context) {
	zklogger.Debug(POOL_LOG_TAG, "Executing start internal method.")
	err := pool.connector(ctx)
	if err != nil {
		if err == InvalidClusterKey {
			zklogger.Error(POOL_LOG_TAG, "Invalid cluster key. Shutting down client.")
			pool.Client.Shutdown()
			pool.Client.killed = true
			pool.Client.ready = true
			return
		}
		pool.retryInterval = pool.retryInterval * 2
		maxRetryInterval := time.Second * time.Duration(pool.Client.Config.MaxRetryInterval)
		if pool.retryInterval > maxRetryInterval {
			pool.retryInterval = maxRetryInterval
		}
	} else {
		pool.retryInterval = time.Second * time.Duration(pool.Client.Config.DefaultRetryInterval)
	}
	pool.ticker.Reset(pool.retryInterval)
	if pool.Client.ready == false && len(pool.idle) == pool.Client.Config.PoolIdleSize {
		pool.Client.ready = true
	}
}

// Add new pool connections if needed.
func (pool *Pool) connector(ctx context.Context) error {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	readPoolSize, writePoolSize := pool.Size()

	toCreateRead := pool.connectionsToCreate(readPoolSize)

	if toCreateRead > 0 {
		zklogger.Debug(POOL_LOG_TAG, "Creating %v read connections.\n", toCreateRead)
	}

	err := pool.createConnections(ctx, toCreateRead, common.Read)

	if err != nil {
		zklogger.Error(POOL_LOG_TAG, "Error creating read connections: %v", err)
		return err
	}

	toCreateWrite := pool.connectionsToCreate(writePoolSize)

	if toCreateWrite > 0 {
		zklogger.Debug(POOL_LOG_TAG, "Creating %v write connections.\n", toCreateWrite)
	}

	err = pool.createConnections(ctx, toCreateWrite, common.Write)

	if err != nil {
		zklogger.Error(POOL_LOG_TAG, "Error creating write connections: %v", err)
		return err
	}
	return nil
}

func (pool *Pool) connectionsToCreate(poolSize *PoolSize) int {
	// Create enough connection to fill the pool
	toCreate := pool.Client.Config.PoolIdleSize - poolSize.idle

	// Ensure to open at most PoolMaxSize readConnections
	if poolSize.total+toCreate > pool.Client.Config.PoolMaxSize {
		toCreate = pool.Client.Config.PoolMaxSize - poolSize.total
	}
	return toCreate
}

func (pool *Pool) createConnections(ctx context.Context, toCreate int, connType common.ConnectionType) error {
	var interfaceConn common.Connection
	for i := 0; i < toCreate; i++ {
		switch connType {
		case common.Read:
			conn := common.NewReadConnection(pool, common.CONNECTING)
			pool.readConnections = append(pool.readConnections, conn)
			interfaceConn = conn
		case common.Write:
			conn := common.NewWriteConnection(pool, common.CONNECTING)
			pool.writeConnections = append(pool.writeConnections, conn)
			interfaceConn = conn
		default:
			zklogger.Error(POOL_LOG_TAG, "Object is of unknown type")
		}
		err := Connect(interfaceConn, ctx, pool, connType, pool.Client.wspLogin.GetAuthToken(), pool.Client.wspLogin.GetClusterId())
		if err != nil {
			zklogger.Error(POOL_LOG_TAG, "Error while creating connection type ", connType, " error is ", err)
			interfaceConn.Close()
			pool.RemoveWithoutLock(interfaceConn)
			return err
		}
	}
	return nil
}

func (pool *Pool) add(conn common.Connection) {
	switch c := (conn).(type) {
	case *common.ReadConnection:
		pool.readConnections = append(pool.readConnections, c)
	case *common.WriteConnection:
		pool.writeConnections = append(pool.writeConnections, c)
	default:
		zklogger.Error(POOL_LOG_TAG, "Object is of unknown type")
	}
}

// Offer offers an idle connection to the server.
func (pool *Pool) Offer(connection *common.WriteConnection) {
	pool.idle <- connection
	zklogger.Debug(POOL_LOG_TAG, "Idle channel length is ", len(pool.idle))
}

func (pool *Pool) RemoveAllConnections() {
	pool.readConnections = make([]*common.ReadConnection, 0)
	pool.writeConnections = make([]*common.WriteConnection, 0)
}

func (pool *Pool) RemoveWithoutLock(conn common.Connection) {
	switch c := (conn).(type) {
	case *common.ReadConnection:
		zklogger.Debug(POOL_LOG_TAG, "Removing read connection from pool")
		filtered := make([]*common.ReadConnection, 0)
		for _, i := range pool.readConnections {
			if c != i {
				filtered = append(filtered, i)
			}
		}
		pool.readConnections = filtered
		zklogger.Debug(POOL_LOG_TAG, "Read connections length in client is ", len(pool.readConnections))
	case *common.WriteConnection:
		zklogger.Debug(POOL_LOG_TAG, "Removing write connection from pool")
		filtered := make([]*common.WriteConnection, 0)
		for _, i := range pool.writeConnections {
			if c != i {
				filtered = append(filtered, i)
			}
		}
		pool.writeConnections = filtered
		zklogger.Debug(POOL_LOG_TAG, "Write connections length in client is ", len(pool.writeConnections))
	default:
		zklogger.Error(POOL_LOG_TAG, "Object is of unknown type")
	}
}

// Remove a connection from the pool
func (pool *Pool) Remove(conn common.Connection) {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	pool.RemoveWithoutLock(conn)
}

// Shutdown close all connection in the pool
func (pool *Pool) Shutdown() {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	close(pool.done)
	for _, conn := range pool.readConnections {
		conn.Close()
	}

	for _, conn := range pool.writeConnections {
		conn.Close()
	}
	pool.RemoveAllConnections()
}

// PoolSize represent the total number of connections and idle connections.
type PoolSize struct {
	idle  int
	total int
}

// Size return the current state of the pool
func (pool *Pool) Size() (*PoolSize, *PoolSize) {
	clientPoolSize := new(PoolSize)
	clientPoolSize.total = len(pool.readConnections)
	for _, connection := range pool.readConnections {
		updateIdleConnCount(connection, clientPoolSize)
	}

	serverPoolSize := new(PoolSize)
	serverPoolSize.total = len(pool.writeConnections)
	for _, connection := range pool.writeConnections {
		updateIdleConnCount(connection, serverPoolSize)
	}

	return clientPoolSize, serverPoolSize
}

func updateIdleConnCount(connection common.Connection, poolSize *PoolSize) {
	switch connection.GetStatus() {
	case common.IDLE:
		poolSize.idle++
	case common.CONNECTING:
		poolSize.idle++
	}
}

func (pool *Pool) GetHttpClient() *http.Client {
	return pool.httpClient
}

func (pool *Pool) GetLock() *sync.RWMutex {
	return &pool.lock
}

func (pool *Pool) GetIdleWriteConnection() *common.WriteConnection {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	connection, err := common.GetValueWithTimeout(pool.idle, pool.Client.Config.GetTimeout())

	if err == nil && connection.Take() {
		return connection
	}
	return nil
}
