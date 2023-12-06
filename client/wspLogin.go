package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	zkhttp "github.com/zerok-ai/zk-utils-go/http"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	clientModel "github.com/zerok-ai/zk-utils-go/zkClient"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var WSP_LOGIN_LOG_TAG = "WspLogin"

var refreshTokenMutex sync.Mutex

type WspLogin struct {
	TokenData        *clientModel.ClusterKeyData
	zkConfig         *Config
	killed           bool
	lastTokenRefresh time.Time
}

type WspLoginResponse struct {
	Payload WspTokenObj         `json:"payload"`
	Error   *zkhttp.ZkHttpError `json:"error,omitempty"`
}

type WspTokenObj struct {
	ClusterKey string `json:"clusterKey"`
	ClusterId  string `json:"clusterId"`
	Killed     bool   `json:"killed"`
}

type WspLoginRequest struct {
	ClusterKey string `json:"clusterKey"`
}

func CreateWspLogin(config *Config) *WspLogin {
	wspLogin := WspLogin{}

	//Assigning initial values.
	wspLogin.zkConfig = config
	wspLogin.killed = false

	return &wspLogin
}

func (h *WspLogin) isKilled() bool {
	return h.killed
}

func (h *WspLogin) RefreshWspToken() error {
	logger.Info(WSP_LOGIN_LOG_TAG, "Request Wsp token.")
	refreshTokenMutex.Lock()
	defer refreshTokenMutex.Unlock()

	if h.killed {
		logger.Info(WSP_LOGIN_LOG_TAG, "Skipping refresh access token api since cluster is killed.")
		return fmt.Errorf("cluster is killed")
	}

	maxRetries := h.zkConfig.WspLogin.MaxRetries
	retryCount := 0

	for retryCount <= maxRetries {
		err2 := h.updateClusterKeyFromZkCloud()
		if err2 != nil {
			retryCount++
		} else {
			break
		}
	}

	return nil
}

func (h *WspLogin) updateClusterKeyFromZkCloud() error {
	port := h.zkConfig.WspLogin.Port
	protocol := "http"
	if port == "443" {
		protocol = "https"
	}
	endpoint := protocol + "://" + h.zkConfig.WspLogin.Host + ":" + h.zkConfig.WspLogin.Port + h.zkConfig.WspLogin.Path

	clusterKey, err := GetSecretValue(h.zkConfig.WspLogin.ClusterKeyNamespace, h.zkConfig.WspLogin.ClusterSecretName, h.zkConfig.WspLogin.ClusterKeyData)

	if err != nil {
		logger.Error(WSP_LOGIN_LOG_TAG, "Error while getting cluster key from secrets :", err)
		return err
	}

	requestPayload := WspLoginRequest{ClusterKey: clusterKey}

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

	var apiResponse WspLoginResponse
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

	if apiResponse.Payload.Killed {
		logger.Info(WSP_LOGIN_LOG_TAG, "Api response came as killed.")
		h.killed = true
	}

	//Saving data to secret
	newData := map[string]string{}
	newData[h.zkConfig.WspLogin.ClusterIdKey] = apiResponse.Payload.ClusterId
	if h.killed {
		newData[h.zkConfig.WspLogin.KilledKey] = "true"
	} else {
		newData[h.zkConfig.WspLogin.KilledKey] = "false"
	}
	newData[h.zkConfig.WspLogin.ClusterKeyData] = apiResponse.Payload.ClusterKey

	err = UpdateSecretValue(h.zkConfig.WspLogin.ClusterKeyNamespace, h.zkConfig.WspLogin.ClusterSecretName, newData)

	tokenData, err := clientModel.DecodeToken(apiResponse.Payload.ClusterKey)
	if err != nil {
		logger.Error(WSP_LOGIN_LOG_TAG, "Error while decoding token :", err)
		return err
	}
	h.TokenData = tokenData
	return nil
}
