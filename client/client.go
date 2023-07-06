package client

import (
	"context"
	"github.com/gorilla/websocket"
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

	//This map contains pool for each target Id.
	pools      map[string]*Pool
	lock       sync.RWMutex
	done       chan struct{}
	httpServer *http.Server
}

// NewClient creates a new Client.
func NewClient(config *Config) (c *Client) {
	c = new(Client)
	c.Config = config
	c.client = &http.Client{}
	c.dialer = &websocket.Dialer{}
	c.pools = make(map[string]*Pool)
	c.done = make(chan struct{})
	return
}

// Start the Proxy
func (c *Client) Start(ctx context.Context) {
	for _, target := range c.Config.Targets {
		pool := NewPool(c, target, c.Config.SecretKey)
		c.pools[target.URL] = pool
		go pool.Start(ctx)
	}
	r := http.NewServeMux()
	r.HandleFunc("/request", c.Request)

	c.httpServer = &http.Server{
		Addr:    c.Config.GetAddr(),
		Handler: r,
	}
	go func() { log.Fatal(c.httpServer.ListenAndServe()) }()
}

func (c *Client) Request(w http.ResponseWriter, r *http.Request) {
	// [1]: Receive requests to be proxied
	// Parse destination URL
	URL, err := common.GetDestinationUrl(w, r)
	if err != nil {
		return
	}
	r.URL = URL

	targetId := r.Header.Get("X-TARGET-ID")
	if targetId == "" {
		//Scenario where there is only one target.
		if len(c.pools) == 1 {
			for key := range c.pools {
				targetId = key
				break
			}
		} else {
			wsp.ProxyErrorf(w, "Missing X-TARGET-ID header")
			return
		}
	}

	log.Printf("[%s] %s %s", r.Method, r.URL.String(), targetId)

	pool := c.pools[targetId]

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
	if err := connection.ProxyRequest(w, r); err != nil {
		// An error occurred throw the connection away
		log.Println(err)
		connection.Close()

		// Try to return an error to the client
		// This might fail if response headers have already been sent
		wsp.ProxyError(w, err)
	}
}

// Shutdown the Proxy
func (c *Client) Shutdown() {
	for _, pool := range c.pools {
		pool.Shutdown()
	}
}
