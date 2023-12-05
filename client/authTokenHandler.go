package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type TokenData struct {
	tokenString string `json:"token"`
	expiresAt   int64  `json:"expiresAt"`
}

func DecodeToken(base64Str string) (*TokenData, error) {
	data, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, fmt.Errorf("error decoding base64 string: %w", err)
	}

	var tokenData TokenData
	err = json.Unmarshal(data, &tokenData)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return &tokenData, nil
}
