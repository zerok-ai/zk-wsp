package main

import (
	"context"
	"fmt"
	"github.com/zerok-ai/zk-wsp/common"
	"github.com/zerok-ai/zk-wsp/utils"
	"os"
	"os/signal"
	"syscall"

	"github.com/zerok-ai/zk-wsp/client"
)

func main() {

	config := client.NewConfig()
	if err := utils.ProcessArgs(config); err != nil {
		fmt.Println("Unable to process wsp client config. Stopping wsp client.")
		return
	}

	//If secretKey is not provided in config, get it from cluster secrets.
	for _, target := range config.Targets {
		if target.SecretKey == "" {
			fmt.Println("SecretKey is empty. Getting from secret in cluster.")
			var err1 error
			target.SecretKey, err1 = common.GetSecretValue(target.ClusterKeyNamespace, target.ClusterSecretName, target.ClusterKeyData)
			if err1 != nil {
				fmt.Println("Error while getting cluster key for target ", err1, " with url ", target.URL)
				return
			}
		}
	}

	proxy := client.NewClient(config)
	proxy.Start(context.Background())

	// Wait signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	// When receives the signal, shutdown
	proxy.Shutdown()
}
