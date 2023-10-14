package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	zklogger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-wsp/common"
	"io"
	"net/http"
	"strconv"
)

var ZK_LOG_TAG = "ZkAuthClient"

func GetClientId(secretKey string, cfg *Config) (string, bool, error) {
	wspLoginResponse, err := WspLogin(secretKey, cfg)
	if err != nil || wspLoginResponse == nil {
		zklogger.Debug(ZK_LOG_TAG, "Error while getting clientId.")
		return "", false, err
	}
	return wspLoginResponse.ClusterId, wspLoginResponse.Killed, nil
}

// WspLogin This method will validate the cluster key and returns the cluster id.
func WspLogin(clusterKey string, cfg *Config) (*WspLoginResponsePayload, error) {
	zklogger.Debug(ZK_LOG_TAG, "Validating secret key.")

	baseURL := "http://" + cfg.ZkCloud.Host + ":" + cfg.ZkCloud.Port + cfg.ZkCloud.LoginPath

	zklogger.Debug(ZK_LOG_TAG, "Url for wsp login ", baseURL)

	requestPayload := WspLoginRequest{ClusterKey: clusterKey}

	data, err := json.Marshal(requestPayload)

	if err != nil {
		zklogger.Error(ZK_LOG_TAG, "Error while creating payload for wsp login request:", err)
		return nil, err
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewReader(data))
	if err != nil {
		zklogger.Error(ZK_LOG_TAG, "Error creating wsp login request:", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		zklogger.Error(ZK_LOG_TAG, "Error sending request for wsp login api :", err)
		return nil, err
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode

	if !common.RespCodeIsOk(statusCode) {
		message := "response code is not ok for wsp login api - " + strconv.Itoa(resp.StatusCode)
		zklogger.Error(ZK_LOG_TAG, message)
		return nil, fmt.Errorf(message)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		zklogger.Error(ZK_LOG_TAG, "Error reading response from wsp login api :", err)
		return nil, err
	}

	zklogger.Debug(ZK_LOG_TAG, "WspLogin response body ", body)

	var apiResponse WspLoginResponse

	err = json.Unmarshal(body, &apiResponse)

	if err != nil {
		zklogger.Error(ZK_LOG_TAG, "Error while unmarshalling wsp login api response :", err)
		return nil, err
	}

	if apiResponse.Error != nil {
		message := "found error in wsp login api response " + apiResponse.Error.Message
		zklogger.Error(ZK_LOG_TAG, message)
		return nil, fmt.Errorf(message)
	}

	return &apiResponse.Payload, nil
}
