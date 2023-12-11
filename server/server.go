package server

import (
	"fmt"
	zklogger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-wsp/common"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zerok-ai/zk-wsp"
)

var SERVER_LOG_TAG = "WspServer"

// Server is a Reverse HTTP Proxy over WebSocket
// This is the Server part, Clients will offer websocket writeConnections,
// those will be pooled to transfer HTTP Request and response
type Server struct {
	Config *Config

	upgrader websocket.Upgrader

	// In pools, keep writeConnections with WebSocket peers.
	pools  map[string]*Pool
	lock   sync.RWMutex
	done   chan struct{}
	server *http.Server
}

// NewServer return a new Server instance
func NewServer(config *Config) (server *Server) {
	rand.Seed(time.Now().Unix())

	server = new(Server)
	server.Config = config
	server.upgrader = websocket.Upgrader{}
	server.pools = make(map[string]*Pool)
	server.done = make(chan struct{})
	return server
}

// Start Server HTTP server
func (s *Server) Start() {
	go func() {
	L:
		for {
			select {
			case <-s.done:
				zklogger.Debug(SERVER_LOG_TAG, "Breaking the cleanup loop.")
				break L
			case <-time.After(15 * time.Second):
				s.clean()
			}
		}
	}()

	connectionStatusHandler := NewClusterConnectionHandler(s.Config, s)

	r := http.NewServeMux()
	// but it is tightly coupled to the internal state of the Server.
	//TODO: Validate the request method here.
	r.HandleFunc("/register", s.Register)
	r.HandleFunc("/request", s.Request)
	r.HandleFunc("/status", s.status)

	s.server = &http.Server{
		Addr:    s.Config.GetAddr(),
		Handler: r,
	}
	connectionStatusHandler.StartPeriodicSync()
	go func() { log.Fatal(s.server.ListenAndServe()) }()
}

// clean removes empty Pools which has no connection.
// It is invoked every 5 sesconds and at shutdown.

func (s *Server) Request(w http.ResponseWriter, r *http.Request) {
	// [1]: Receive requests to be proxied
	// Parse destination URL
	URL, err := common.GetDestinationUrl(w, r)
	if err != nil {
		return
	}
	r.URL = URL

	clientId, err := common.GetClientId(w, r)
	if err != nil {
		wsp.ProxyErrorf(w, "Missing clientId value")
		return
	}

	zklogger.Debug(SERVER_LOG_TAG, "[%s] %s", r.Method, r.URL.String())

	pool := s.pools[clientId]

	if pool == nil {
		wsp.ProxyErrorf(w, "No pool available for the target client.")
		return
	}

	connection := pool.GetIdleWriteConnection()
	if connection == nil {
		// It means that dispatcher has set `nil` which is a system error case that is
		// not expected in the normal flow.
		wsp.ProxyErrorf(w, "Unable to get a proxy connection")
		return
	}

	// [3]: Send the request to the peer through the WebSocket connection.
	if _, err := connection.ProxyRequest(w, r); err != nil {
		// An error occurred throw the connection away
		zklogger.Error(SERVER_LOG_TAG, "Error while proxying request ", err)
		connection.Close()
		pool.Remove(connection)

		// Try to return an error to the httpClient
		// This might fail if response headers have already been sent
		wsp.ProxyError(w, err)
	}
}

// Request receives the WebSocket upgrade handshake request from wsp_client.
func (s *Server) Register(w http.ResponseWriter, r *http.Request) {

	//This secret Key is the access token.
	secretKey := r.Header.Get("X-SECRET-KEY")
	baseURL := "http://" + s.Config.ZkCloud.Host + ":" + s.Config.ZkCloud.Port + s.Config.ZkCloud.LoginPath
	zklogger.Debug(SERVER_LOG_TAG, "Base url for login is ", baseURL)
	zklogger.Debug(SERVER_LOG_TAG, "Secret key is ", secretKey)
	response, err := common.ValidateKeyWithZkCloud(secretKey, baseURL)
	if err != nil {
		wsp.ProxyErrorf(w, "Error while getting clientId : %v", err)
		return
	}
	if !response.Payload.IsValid {
		wsp.InvalidClusterErrorf(w, "Secret key is invalid or killed.")
		return
	}

	clientId := response.Payload.ClusterId

	// 1. Upgrade a received HTTP request to a WebSocket connection
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		wsp.ProxyErrorf(w, "HTTP upgrade error : %v", err)
		return
	}

	// 2. Wait a greeting message from the peer and parse it
	// The first message should contain the remote client id, idleSize and connectionType
	_, greeting, err := ws.ReadMessage()
	if err != nil {
		wsp.ProxyErrorf(w, "Unable to read greeting message : %s", err)
		ws.Close()
		return
	}

	// Parse the greeting message
	split := strings.Split(string(greeting), "_")

	idleSize, err := strconv.Atoi(split[0])
	if err != nil {
		wsp.ProxyErrorf(w, "Unable to parse greeting message : %s", err)
		ws.Close()
		return
	}
	connectionType, err := strconv.Atoi(split[1])
	if err != nil {
		wsp.ProxyErrorf(w, "Unable to parse greeting message : %s", err)
		ws.Close()
		return
	}

	zklogger.Debug(SERVER_LOG_TAG, "Clientid is ", clientId, "idleSize is ", idleSize, "connectionType is ", connectionType)

	// 3. Register the connection into server pools.
	// s.lock is for exclusive control of pools operation.
	s.lock.Lock()
	defer s.lock.Unlock()

	zklogger.Debug(SERVER_LOG_TAG, "Executing next line after lock.")

	var pool *Pool
	// There is no need to create a new pool,
	// if it is already registered in current pools.
	for _, p := range s.pools {
		if p.clientId == clientId {
			pool = p
			break
		}
	}
	if pool == nil {
		pool = NewPool(s, clientId)
		s.pools[clientId] = pool
	}

	// update pool idleSize
	pool.idleSize = idleSize

	// Add the WebSocket connection to the pool
	pool.AddConnection(ws, common.ConnectionType(connectionType))
	zklogger.Debug(SERVER_LOG_TAG, "Adding connection done.")
}

func (s *Server) getConnectionStatus(clientId string) (bool, error) {
	zklogger.Debug(SERVER_LOG_TAG, "Getting connection status for client ", clientId)
	pool := s.pools[clientId]
	if pool == nil {
		zklogger.Debug(SERVER_LOG_TAG, "No pool available for the target client.")
		return false, fmt.Errorf("no pool available for the target client")
	}

	busyConnections := pool.GetAllBusyWriteConnections()
	connection := pool.GetIdleWriteConnection()
	zklogger.Debug(SERVER_LOG_TAG, "Number of Busy connections are ", len(busyConnections))
	if connection == nil {
		if len(busyConnections) > 0 {
			// It means that there is no idle connection. Assuming here that the cluster is connected since there are some busy connections.
			zklogger.Debug(SERVER_LOG_TAG, "No idle write connection.")
			return true, nil
		} else {
			// It means that there is no idle connection and no busy connections.
			zklogger.Debug(SERVER_LOG_TAG, "No idle write connection and busy connections.")
			return false, fmt.Errorf("no idle or busy connection available for the target client")
		}
	}
	err := connection.SendPingMessage()
	if err != nil {
		zklogger.Error(SERVER_LOG_TAG, "Error while sending ping message.")
		return false, fmt.Errorf("error while sending ping message")
	}
	return true, nil
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func (s *Server) clean() {
	zklogger.Debug(SERVER_LOG_TAG, "Cleaning empty connections.")
	s.lock.Lock()
	defer s.lock.Unlock()

	if len(s.pools) == 0 {
		return
	}

	pools := make(map[string]*Pool)
	for id, pool := range s.pools {
		if pool.IsEmpty() {
			zklogger.Debug(SERVER_LOG_TAG, "Removing empty connection pool : %s", pool.clientId)
			pool.Shutdown()
		} else {
			pools[id] = pool
		}
	}

	s.pools = pools
	//zklogger.Debug(SERVER_LOG_TAG,"Done with cleaning empty connections.")
}

// Shutdown stop the Server
func (s *Server) Shutdown() {
	close(s.done)
	for _, pool := range s.pools {
		pool.Shutdown()
	}
	s.pools = make(map[string]*Pool)
	//Removing this as it is not needed.
	//s.clean()
}
