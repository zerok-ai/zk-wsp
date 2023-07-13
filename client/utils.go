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
		log.Printf("Unable to connect to %s : %s", pool.Target, err)
		//Removing the connection from pool since there is a connection error.
		pool.Remove(interfaceConn)
		return err
	}
	go interfaceConn.Start()
	return nil
}

// Connect to the IsolatorServer using a HTTP websocket
func connectInternal(ctx context.Context, conn common.Connection, pool *Pool, connectionType common.ConnectionType) (err error) {
	if pool == nil {
		log.Printf("Aborting connection since pool is nil.")
	}

	log.Printf("Connecting to %s and type %v", pool.Target, connectionType)

	targetConfig := pool.Target

	secretKey := targetConfig.SecretKey

	if err != nil {
		fmt.Println("Error while getting cluster key for ws connection to server", err, connectionType)
		return err
	}

	// Create a new TCP(/TLS) connection ( no use of net.http )
	ws, response, err := pool.client.dialer.DialContext(
		ctx,
		targetConfig.URL,
		http.Header{"X-SECRET-KEY": {secretKey}},
	)

	if response != nil && response.StatusCode == InvalidClusterKeyResponseCode {
		fmt.Println("Invalid cluster key")
		return InvalidClusterKey
	}

	if err != nil {
		fmt.Println("Error while establishing websocket connection to server", err, connectionType)
		return err
	}

	conn.SetWs(ws)

	log.Printf("Connected to %s and type %v", pool.Target, connectionType)

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
		"%d_%d",
		pool.client.Config.PoolIdleSize,
		serverConnType,
	)

	if err := ws.WriteMessage(websocket.TextMessage, []byte(greeting)); err != nil {
		log.Println("greeting error :", err)
		return err
	}

	return nil
}
