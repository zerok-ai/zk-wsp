package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// ConnectionStatus represents the status of a connection
type ConnectionStatus struct {
	ClientID string `json:"clientID"`
	IsActive bool   `json:"isActive"`
}

// PushData pushes data to an external service
func PushData(url string, data []ConnectionStatus) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshalling data: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-ok status code: %d", resp.StatusCode)
	}

	return nil
}
