package main

import (
	"context"
	zklogger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-wsp/common"
	"github.com/zerok-ai/zk-wsp/utils"
	"os"
	"os/signal"
	"syscall"

	"github.com/zerok-ai/zk-wsp/client"
)

var LOG_TAG = "WspClientMain"

func main() {

	config := client.NewConfig()
	if err := utils.ProcessArgs(config); err != nil {
		zklogger.Error(LOG_TAG, "Unable to process wsp client config. Stopping wsp client.")
		return
	}

	zklogger.Init(config.LogsConfig)

	//If secretKey is not provided in config, get it from cluster secrets.
	if config.Target.SecretKey == "" {
		zklogger.Debug(LOG_TAG, "SecretKey is empty. Getting from secret in cluster.")
		var err1 error
		config.Target.SecretKey, err1 = common.GetSecretValue(config.WspLogin.ClusterKeyNamespace, config.WspLogin.ClusterSecretName, config.WspLogin.ClusterKeyData)
		if err1 != nil {
			zklogger.Error(LOG_TAG, "Error while getting cluster key for target ", err1, " with url ", config.Target.URL)
			return
		}
	}

	proxy := client.NewClient(config)
	if proxy == nil {
		zklogger.Error(LOG_TAG, "Unable to create wsp client. Stopping wsp client.")
		return
	}
	proxy.Start(context.Background())

	// Wait signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	// When receives the signal, shutdown
	proxy.Shutdown()
}
