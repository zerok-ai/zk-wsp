package main

import (
	"fmt"
	"github.com/zerok-ai/zk-wsp/utils"
	"os"
	"os/signal"
	"syscall"

	"github.com/zerok-ai/zk-wsp/server"
)

func main() {
	config := server.NewConfig()
	if err := utils.ProcessArgs(config); err != nil {
		fmt.Println("Unable to process wsp server config. Stopping wsp server.")
		return
	}

	wspServer := server.NewServer(config)
	wspServer.Start()

	// Wait signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	// When receives the signal, shutdown
	wspServer.Shutdown()
}
