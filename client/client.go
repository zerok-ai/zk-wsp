package client

import (
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	zklogger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-wsp"
	"github.com/zerok-ai/zk-wsp/common"
	"log"
	"net/http"
	"sync"
)

// Client connects to one or more Server using HTTP websockets.
// The Server can then send HTTP requests to execute.
type Client struct {
	Config *Config
	client *http.Client
	dialer *websocket.Dialer

	pool                *Pool
	lock                sync.RWMutex
	done                chan struct{}
	httpServer          *http.Server
	ready               bool
	killed              bool
	clusterTokenHandler *ClusterTokenHandler
}

var ZK_LOG_TAG = "WspClient"

// NewClient creates a new Client.
func NewClient(config *Config) (c *Client) {
	c = new(Client)
	c.Config = config
	c.client = &http.Client{}
	c.dialer = &websocket.Dialer{}
	c.done = make(chan struct{})
	c.ready = false
	wspLogin := CreateWspLogin(config)
	validateKey := CreateValidateKey(config)
	c.clusterTokenHandler = NewClusterTokenHandler(config, wspLogin, validateKey)
	return
}

// Start the Proxy
func (c *Client) Start(ctx context.Context) {
	target := c.Config.Target
	pool := NewPool(c, target, c.Config.SecretKey)
	c.pool = pool
	pool.Start(ctx)

	r := http.NewServeMux()
	//TODO: Validate the request method here.
	r.HandleFunc("/request", c.Request)
	r.HandleFunc("/healthz", c.Status)

	c.httpServer = &http.Server{
		Addr:    c.Config.GetAddr(),
		Handler: r,
	}
	c.clusterTokenHandler.StartPeriodicSync()
	go func() { log.Fatal(c.httpServer.ListenAndServe()) }()
}

func (c *Client) Status(w http.ResponseWriter, r *http.Request) {
	if c.ready {
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	} else {
		w.WriteHeader(503)
		w.Write([]byte("Wsp Client Not Ready."))
	}
}

func (c *Client) SendKillResponse(w http.ResponseWriter) {
	responseObj := ClusterKillResponseObj{Killed: true}
	resp := ClusterKillResponse{Payload: responseObj}
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)
	w.WriteHeader(200)
}

func (c *Client) Request(w http.ResponseWriter, r *http.Request) {
	// [0]: Check if the client is killed.
	if c.killed {
		c.SendKillResponse(w)
		return
	}

	// [1]: Receive requests to be proxied
	// Parse destination URL
	URL, err := common.GetDestinationUrl(w, r)
	if err != nil {
		return
	}
	r.URL = URL

	if c.pool == nil {
		wsp.ProxyErrorf(w, "No pool available for the target client.")
		return
	}

	connection := c.pool.GetIdleWriteConnection()

	if connection == nil {
		// It means that dispatcher has set `nil` which is a system error case that is
		// not expected in the normal flow.
		wsp.ProxyErrorf(w, "Unable to get a proxy connection")
		return
	}

	// [3]: Send the request to the peer through the WebSocket connection.
	if err := connection.ProxyRequest(w, r); err != nil {
		// An error occurred throw the connection away
		zklogger.Error(ZK_LOG_TAG, "Error while proxying request: %s", err.Error())
		connection.Close()
		c.pool.Remove(connection)

		// Try to return an error to the client
		// This might fail if response headers have already been sent
		wsp.ProxyError(w, err)
	}
}

// Shutdown the Proxy
func (c *Client) Shutdown() {
	c.pool.Shutdown()
}
