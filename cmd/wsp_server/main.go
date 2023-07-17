package main

import (
	zklogger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-wsp/utils"
	"os"
	"os/signal"
	"syscall"

	"github.com/zerok-ai/zk-wsp/server"
)

var LOG_TAG = "WspServerMain"

func main() {
	config := server.NewConfig()
	if err := utils.ProcessArgs(config); err != nil {
		zklogger.Debug(LOG_TAG, "Unable to process wsp server config. Stopping wsp server.")
		return
	}

	zklogger.Init(config.LogsConfig)

	wspServer := server.NewServer(config)
	wspServer.Start()

	// Wait signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	// When receives the signal, shutdown
	wspServer.Shutdown()
}
