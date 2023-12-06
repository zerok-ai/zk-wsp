package common

import zkhttp "github.com/zerok-ai/zk-utils-go/http"

type ValidateKeyResponse struct {
	Payload ValidateKeyObj      `json:"payload"`
	Error   *zkhttp.ZkHttpError `json:"error,omitempty"`
}

type ValidateKeyObj struct {
	IsValid   bool   `json:"is_valid"`
	Ttl       int    `json:"ttl"`
	ClusterId string `json:"clusterId"`
}

type ValidateKeyRequest struct {
	ClusterKey string `json:"clusterKey"`
}
