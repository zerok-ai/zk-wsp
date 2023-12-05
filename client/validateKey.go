package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	zkhttp "github.com/zerok-ai/zk-utils-go/http"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"io"
	"net/http"
	"strconv"
	"time"
)

type ValidateKey struct {
	ClusterKey string
	zkConfig   Config
	killed     bool
	ttl        time.Duration
}

type ValidateKeyResponse struct {
	Payload ValidateKeyObj      `json:"payload"`
	Error   *zkhttp.ZkHttpError `json:"error,omitempty"`
}

type ValidateKeyObj struct {
	ClusterKey string `json:"clusterKey"`
	Ttl        int    `json:"ttl"`
}

type ValidateKeyRequest struct {
	ClusterKey string `json:"clusterKey"`
}

func CreateValidateKey(config Config) *ValidateKey {
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
	requestPayload := ValidateKeyRequest{ClusterKey: h.ClusterKey}

	data, err := json.Marshal(requestPayload)

	if err != nil {
		logger.Error(WSP_LOGIN_LOG_TAG, "Error while creating payload for operator login request:", err)
		return err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(data))

	if err != nil {
		logger.Error(WSP_LOGIN_LOG_TAG, "Error creating operator login request:", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		logger.Error(WSP_LOGIN_LOG_TAG, "Error sending request for operator login api :", err)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		logger.Error(WSP_LOGIN_LOG_TAG, "Error reading response from operator login api :", err)
		return err
	}

	if !RespCodeIsOk(resp.StatusCode) {
		message := "response code is not ok for wsp login api - " + strconv.Itoa(resp.StatusCode)
		logger.Error(WSP_LOGIN_LOG_TAG, message)
		return fmt.Errorf(message)
	}

	var apiResponse ValidateKeyResponse
	err = json.Unmarshal(body, &apiResponse)

	if err != nil {
		logger.Error(WSP_LOGIN_LOG_TAG, "Error while unmarshalling rules operator login api response :", err)
		return err
	}

	if apiResponse.Error != nil {
		message := "found error in operator login api response " + apiResponse.Error.Message
		logger.Error(WSP_LOGIN_LOG_TAG, message)
		return fmt.Errorf(message)
	}

	h.ClusterKey = apiResponse.Payload.ClusterKey
	h.ttl = time.Duration(apiResponse.Payload.Ttl) * time.Minute
	return nil
}
