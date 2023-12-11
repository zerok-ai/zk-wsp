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
	"sync"
)

var WSP_LOGIN_LOG_TAG = "WspLogin"

type WspLogin struct {
	authToken         string
	clusterId         string
	zkConfig          *Config
	killed            bool
	refreshInProgress bool
	refreshTokenMutex sync.Mutex
}

type WspLoginResponse struct {
	Payload WspTokenObj         `json:"payload"`
	Error   *zkhttp.ZkHttpError `json:"error,omitempty"`
}

type WspTokenObj struct {
	AuthToken string `json:"accessToken"`
	ClusterId string `json:"clusterId"`
	Killed    bool   `json:"killed"`
}

type WspLoginRequest struct {
	ClusterKey string `json:"clusterKey"`
}

func CreateWspLogin(config *Config) *WspLogin {
	wspLogin := WspLogin{}

	//Assigning initial values.
	wspLogin.zkConfig = config
	wspLogin.killed = false
	wspLogin.refreshInProgress = false
	wspLogin.clusterId = ""
	wspLogin.authToken = ""
	err := wspLogin.RefreshWspToken()
	if err != nil {
		logger.Error(WSP_LOGIN_LOG_TAG, "Error while refreshing wsp token :", err)
		return nil
	}
	return &wspLogin
}

func (h *WspLogin) isKilled() bool {
	return h.killed
}

func (h *WspLogin) RefreshWspToken() error {
	logger.Info(WSP_LOGIN_LOG_TAG, "Request Wsp token.")
	h.refreshTokenMutex.Lock()
	if h.refreshInProgress {
		logger.Info(WSP_LOGIN_LOG_TAG, "Refresh token api is already in progress.")
		h.refreshTokenMutex.Unlock()
		return nil
	}
	h.refreshInProgress = true
	h.refreshTokenMutex.Unlock()

	if h.killed {
		logger.Info(WSP_LOGIN_LOG_TAG, "Skipping refresh access token api since cluster is killed.")
		return fmt.Errorf("cluster is killed")
	}

	err := h.updateAuthTokenFromZkCloud()
	h.refreshTokenMutex.Lock()
	h.refreshInProgress = false
	h.refreshTokenMutex.Unlock()
	return err
}

func (h *WspLogin) updateAuthTokenFromZkCloud() error {
	port := h.zkConfig.WspLogin.Port
	protocol := "http"
	if port == "443" {
		protocol = "https"
	}
	endpoint := protocol + "://" + h.zkConfig.WspLogin.Host + ":" + h.zkConfig.WspLogin.Port + h.zkConfig.WspLogin.Path
	logger.Debug(WSP_LOGIN_LOG_TAG, "Endpoint for wsp login api is ", endpoint)

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
		message := "found error in wsp login api response " + apiResponse.Error.Message
		logger.Error(WSP_LOGIN_LOG_TAG, message)
		return fmt.Errorf(message)
	}

	if apiResponse.Payload.Killed {
		logger.Info(WSP_LOGIN_LOG_TAG, "Api response came as killed.")
		h.killed = true
	}

	h.authToken = apiResponse.Payload.AuthToken
	h.clusterId = apiResponse.Payload.ClusterId

	return nil
}
