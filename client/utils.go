package client

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/zerok-ai/zk-wsp/common"
	"log"
	"net/http"
)

func Connect(interfaceConn common.Connection, ctx context.Context, pool *Pool, connType common.ConnectionType) error {
	err := connectInternal(ctx, interfaceConn, pool, connType)
	if err != nil {
		log.Printf("Unable to connect to %s : %s", pool.target, err)
		pool.lock.Lock()
		defer pool.lock.Unlock()
		//Removing the connection from pool since there is a connection error.
		pool.Remove(interfaceConn)
		return err
	}
	go interfaceConn.Start()
	return nil
}

// Connect to the IsolatorServer using a HTTP websocket
func connectInternal(ctx context.Context, conn common.Connection, pool *Pool, connectionType common.ConnectionType) (err error) {
	log.Printf("Connecting to %s", pool.target)

	// Create a new TCP(/TLS) connection ( no use of net.http )
	ws, _, err := pool.client.dialer.DialContext(
		ctx,
		pool.target,
		http.Header{"X-SECRET-KEY": {pool.secretKey}},
	)

	if err != nil {
		return err
	}

	conn.SetWs(ws)

	log.Printf("Connected to %s", pool.target)

	var serverConnType common.ConnectionType

	switch connectionType {
	case common.Read:
		serverConnType = common.Write
	case common.Write:
		serverConnType = common.Read
	default:
		fmt.Println("Object is of unknown type")
	}

	// Send the greeting message with proxy id and wanted pool size.
	greeting := fmt.Sprintf(
		"%s_%d_%d",
		pool.client.Config.ID,
		pool.client.Config.PoolIdleSize,
		serverConnType,
	)

	if err := ws.WriteMessage(websocket.TextMessage, []byte(greeting)); err != nil {
		log.Println("greeting error :", err)
		conn.Close()
		return err
	}

	return nil
}
