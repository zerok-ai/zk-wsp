package common

import zkhttp "github.com/zerok-ai/zk-utils-go/http"

type ValidateAccessTokenResponse struct {
	Payload ValidateAccessTokenObj `json:"payload"`
	Error   *zkhttp.ZkHttpError    `json:"error,omitempty"`
}

type ValidateAccessTokenObj struct {
	IsValid   bool   `json:"isValid"`
	Ttl       int    `json:"ttl"`
	ClusterId string `json:"clusterId"`
}

type ValidateAccessTokenRequest struct {
	AccessToken string `json:"token"`
	ClusterId   string `json:"clusterId"`
}
