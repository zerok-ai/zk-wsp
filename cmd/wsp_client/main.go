package main

import (
	"context"
	"fmt"
	"github.com/zerok-ai/zk-wsp/utils"
	"os"
	"os/signal"
	"syscall"

	"github.com/zerok-ai/zk-wsp/client"
)

func main() {

	var config client.Config
	if err := utils.ProcessArgs(&config); err != nil {
		fmt.Println("Unable to process wsp client config. Stopping wsp client.")
		return
	}

	proxy := client.NewClient(&config)
	proxy.Start(context.Background())

	// Wait signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	// When receives the signal, shutdown
	proxy.Shutdown()
}
