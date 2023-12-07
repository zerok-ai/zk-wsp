package client

import (
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-wsp/common"
	"time"
)

type ValidateKey struct {
	ClusterKey string
	ClusterId  string
	zkConfig   *Config
	killed     bool
	ttl        time.Duration
}

func (h *ValidateKey) GetClusterId() string {
	return h.ClusterId
}

func CreateValidateKey(config *Config) *ValidateKey {
	validateKey := ValidateKey{}

	//Assigning initial values.
	validateKey.zkConfig = config
	validateKey.killed = false

	return &validateKey
}

func (h *ValidateKey) isKilled() bool {
	return h.killed
}

func (h *ValidateKey) UpdateClusterKey() error {
	clusterKey, err := GetSecretValue(h.zkConfig.WspLogin.ClusterKeyNamespace, h.zkConfig.WspLogin.ClusterSecretName, h.zkConfig.WspLogin.ClusterKeyData)

	if err != nil {
		logger.Error(WSP_LOGIN_LOG_TAG, "Error while getting cluster key from secrets :", err)
		return err
	}
	h.ClusterKey = clusterKey
	return nil
}

func (h *ValidateKey) ValidateKeyWithZkCloud() error {
	port := h.zkConfig.WspLogin.Port
	protocol := "http"
	if port == "443" {
		protocol = "https"
	}
	endpoint := protocol + "://" + h.zkConfig.WspLogin.Host + ":" + h.zkConfig.WspLogin.Port + h.zkConfig.WspLogin.ValidateKeyPath
	if h.ClusterKey == "" {
		err := h.UpdateClusterKey()
		if err != nil {
			return err
		}
	}
	apiResponse, err2 := common.ValidateKeyWithZkCloud(h.ClusterKey, endpoint)
	if err2 != nil {
		return err2
	}

	h.ttl = time.Duration(apiResponse.Payload.Ttl) * time.Second
	h.ClusterId = apiResponse.Payload.ClusterId
	return nil
}
