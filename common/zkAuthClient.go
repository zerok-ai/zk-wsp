package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"io"
	"net/http"
	"strconv"
)

var ZK_AUTH_LOG_TAG = "ZkAuthClient"

func ValidateKeyWithZkCloud(clusterKey, clusterId, endpoint string) (ValidateAccessTokenResponse, error) {
	requestPayload := ValidateAccessTokenRequest{AccessToken: clusterKey, ClusterId: clusterId}

	logger.Debug(ZK_AUTH_LOG_TAG, "endpoint is ", endpoint)

	payloadBuf := new(bytes.Buffer)
	json.NewEncoder(payloadBuf).Encode(requestPayload)

	req, err := http.NewRequest("POST", endpoint, payloadBuf)

	if err != nil {
		logger.Error(ZK_AUTH_LOG_TAG, "Error creating validate key request:", err)
		return ValidateAccessTokenResponse{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		logger.Error(ZK_AUTH_LOG_TAG, "Error sending request for validate key api :", err)
		return ValidateAccessTokenResponse{}, err
	}
	defer resp.Body.Close()

	logger.Debug(ZK_AUTH_LOG_TAG, "response is ", resp)
	logger.Debug(ZK_AUTH_LOG_TAG, "response code is ", resp.StatusCode)

	if !RespCodeIsOk(resp.StatusCode) {
		message := "response code is not ok for validate key api - " + strconv.Itoa(resp.StatusCode)
		logger.Error(ZK_AUTH_LOG_TAG, message)
		return ValidateAccessTokenResponse{}, fmt.Errorf(message)
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		logger.Error(ZK_AUTH_LOG_TAG, "Error reading response from validate key api :", err)
		return ValidateAccessTokenResponse{}, err
	}

	var apiResponse ValidateAccessTokenResponse
	err = json.Unmarshal(body, &apiResponse)

	if err != nil {
		logger.Error(ZK_AUTH_LOG_TAG, "Error while unmarshalling rules validate key api response :", err)
		return ValidateAccessTokenResponse{}, err
	}

	if apiResponse.Error != nil {
		message := "found error in validate key api response " + apiResponse.Error.Message
		logger.Error(ZK_AUTH_LOG_TAG, message)
		return ValidateAccessTokenResponse{}, fmt.Errorf(message)
	}
	return apiResponse, nil
}
