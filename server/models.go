package server

import zkhttp "github.com/zerok-ai/zk-utils-go/http"

type WspLoginRequest struct {
	ClusterKey string `json:"clusterKey"`
}

type WspLoginResponse struct {
	Payload WspLoginResponsePayload `json:"payload"`
	Error   *zkhttp.ZkHttpError     `json:"error,omitempty"`
}

type WspLoginResponsePayload struct {
	ClusterId string `json:"clusterId"`
}
