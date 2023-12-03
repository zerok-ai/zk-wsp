package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	serviceModel "github.com/zerok-ai/zk-utils-go/clusterMetadata/model"
	zklogger "github.com/zerok-ai/zk-utils-go/logs"
	zktick "github.com/zerok-ai/zk-utils-go/ticker"
	"net/http"
	"time"
)

var ClusterConnectionHandlerTag = "ClusterConnectionHandler"

type ClusterConnectionHandler struct {
	config *Config
	server *Server
	ticker *zktick.TickerTask
}

func NewClusterConnectionHandler(config *Config, server *Server) *ClusterConnectionHandler {
	ch := &ClusterConnectionHandler{
		config: config,
		server: server,
	}
	var duration = time.Duration(config.ZkCloud.ConnectionSyncInterval) * time.Second
	ch.ticker = zktick.GetNewTickerTask("ClusterHeathSync", duration, ch.PeriodicSync)
	return ch
}
func (ch *ClusterConnectionHandler) StartPeriodicSync() {
	ch.PeriodicSync()
	ch.ticker.Start()
}

func (ch *ClusterConnectionHandler) PeriodicSync() {
	allClientConnectionStatus := make(map[string]serviceModel.ClusterConnection)
	for clientId, _ := range ch.server.pools {
		state, err := ch.server.getConnectionStatus(clientId)
		if err != nil {
			zklogger.Debug(SERVER_LOG_TAG, "Error while getting connection status for client ", clientId, " with error ", err)
		}
		if state {
			zklogger.Debug(SERVER_LOG_TAG, "Connected status for client ", clientId)
		} else {
			zklogger.Debug(SERVER_LOG_TAG, "Not connected status for client ", clientId)
		}
		allClientConnectionStatus[clientId] = serviceModel.ClusterConnection{Connected: state}
	}
	baseURL := "http://" + ch.config.ZkCloud.Host + ":" + ch.config.ZkCloud.Port + ch.config.ZkCloud.ConnectionSyncPath
	ch.PushData(baseURL, allClientConnectionStatus)
}

// PushData pushes data to an external service
func (ch *ClusterConnectionHandler) PushData(url string, data map[string]serviceModel.ClusterConnection) error {
	request := serviceModel.ClusterConnectionRequest{
		Status: data,
	}
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error marshalling data: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-ok status code: %d", resp.StatusCode)
	}

	return nil
}
