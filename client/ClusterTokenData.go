package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type ClusterTokenData struct {
	TokenString string `json:"token"`
}

func DecodeToken(base64Str string) (*ClusterTokenData, error) {
	data, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, fmt.Errorf("error decoding base64 string: %w", err)
	}

	var tokenData ClusterTokenData
	err = json.Unmarshal(data, &tokenData)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return &tokenData, nil
}
