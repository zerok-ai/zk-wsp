package client

type ClusterKillResponse struct {
	Payload ClusterKillResponseObj `json:"payload"`
}

type ClusterKillResponseObj struct {
	Killed bool `json:"killed"`
}
