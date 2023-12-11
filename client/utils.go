package client

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	zklogger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-wsp/common"
	"net/http"
)

var UTILS_LOG_TAG = "ClientUtils"

func Connect(interfaceConn common.Connection, ctx context.Context, pool *Pool, connType common.ConnectionType, token string) error {
	err := connectInternal(ctx, interfaceConn, pool, connType, token)
	if err != nil {
		zklogger.Error(UTILS_LOG_TAG, "Unable to connect to %s : %s", pool.Target, err)
		return err
	}
	go interfaceConn.Start()
	return nil
}

// Connect to the IsolatorServer using a HTTP websocket
func connectInternal(ctx context.Context, conn common.Connection, pool *Pool, connectionType common.ConnectionType, token string) (err error) {
	if pool == nil {
		zklogger.Error(UTILS_LOG_TAG, "Aborting connection since pool is nil.")
	}

	zklogger.Debug(UTILS_LOG_TAG, "Connecting to %s and type %v", pool.Target, connectionType)

	targetConfig := pool.Target

	secretKey := token

	if err != nil {
		zklogger.Error(UTILS_LOG_TAG, "Error while getting cluster key for ws connection to server", err, connectionType)
		return err
	}

	// Create a new TCP(/TLS) connection ( no use of net.http )
	ws, response, err := pool.Client.dialer.DialContext(
		ctx,
		targetConfig.URL,
		http.Header{"X-SECRET-KEY": {secretKey}},
	)

	if response != nil && response.StatusCode == InvalidClusterKeyResponseCode {
		zklogger.Error(UTILS_LOG_TAG, "Invalid cluster key")
		return InvalidClusterKey
	}

	if err != nil {
		zklogger.Error(UTILS_LOG_TAG, "Error while establishing websocket connection to server", err, connectionType)
		return err
	}

	conn.SetWs(ws)

	zklogger.Debug(UTILS_LOG_TAG, "Connected to %s and type %v", pool.Target, connectionType)

	var serverConnType common.ConnectionType

	switch connectionType {
	case common.Read:
		serverConnType = common.Write
	case common.Write:
		serverConnType = common.Read
	default:
		zklogger.Error(UTILS_LOG_TAG, "Object is of unknown type")
	}

	// Send the greeting message with proxy id and wanted pool size.
	greeting := fmt.Sprintf(
		"%d_%d",
		pool.Client.Config.PoolIdleSize,
		serverConnType,
	)

	if err := ws.WriteMessage(websocket.TextMessage, []byte(greeting)); err != nil {
		zklogger.Error(UTILS_LOG_TAG, "greeting error :", err)
		return err
	}

	return nil
}

func RespCodeIsOk(status int) bool {
	if status > 199 && status < 300 {
		return true
	}
	return false

}
