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

func ValidateKeyWithZkCloud(clusterKey, endpoint string) (ValidateKeyResponse, error) {
	requestPayload := ValidateKeyRequest{ClusterKey: clusterKey}

	data, err := json.Marshal(requestPayload)

	if err != nil {
		logger.Error(ZK_AUTH_LOG_TAG, "Error while creating payload for operator login request:", err)
		return ValidateKeyResponse{}, err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(data))

	if err != nil {
		logger.Error(ZK_AUTH_LOG_TAG, "Error creating operator login request:", err)
		return ValidateKeyResponse{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		logger.Error(ZK_AUTH_LOG_TAG, "Error sending request for operator login api :", err)
		return ValidateKeyResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		logger.Error(ZK_AUTH_LOG_TAG, "Error reading response from operator login api :", err)
		return ValidateKeyResponse{}, err
	}

	if !RespCodeIsOk(resp.StatusCode) {
		message := "response code is not ok for wsp login api - " + strconv.Itoa(resp.StatusCode)
		logger.Error(ZK_AUTH_LOG_TAG, message)
		return ValidateKeyResponse{}, fmt.Errorf(message)
	}

	var apiResponse ValidateKeyResponse
	err = json.Unmarshal(body, &apiResponse)

	if err != nil {
		logger.Error(ZK_AUTH_LOG_TAG, "Error while unmarshalling rules operator login api response :", err)
		return ValidateKeyResponse{}, err
	}

	if apiResponse.Error != nil {
		message := "found error in operator login api response " + apiResponse.Error.Message
		logger.Error(ZK_AUTH_LOG_TAG, message)
		return ValidateKeyResponse{}, fmt.Errorf(message)
	}
	return apiResponse, nil
}
