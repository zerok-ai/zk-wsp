package client

import (
	zklogger "github.com/zerok-ai/zk-utils-go/logs"
	zktick "github.com/zerok-ai/zk-utils-go/ticker"
	"time"
)

var minimumInterval = 5 * time.Minute

var ClusterTokenHandlerLogTag = "ClusterTokenHandler"

type ClusterTokenHandler struct {
	config        *Config
	ticker        *zktick.TickerTask
	timerInterval time.Duration
	expiryTime    time.Time
	wspLogin      *WspLogin
	validateKey   *ValidateKey
}

func NewClusterTokenHandler(config *Config, wspLogin *WspLogin, validateKey *ValidateKey) *ClusterTokenHandler {
	return &ClusterTokenHandler{
		config:        config,
		timerInterval: minimumInterval,
		//Setting the expiry time as now to frequently force a refresh
		expiryTime:  time.Now(),
		wspLogin:    wspLogin,
		validateKey: validateKey,
	}
}

func (h *ClusterTokenHandler) StartPeriodicSync() {
	h.ticker = zktick.GetNewTickerTask("ClusterTokenHandler", h.timerInterval, h.CheckExpiryAndUpdateClusterToken)
	h.ticker.Start()
}

func (h *ClusterTokenHandler) CheckExpiryAndUpdateClusterToken() {
	err := h.updateExpiry()
	if err != nil {
		zklogger.Error(ClusterTokenHandlerLogTag, "Error while updating expiry :", err)
	}
	currentTime := time.Now()
	if h.expiryTime.Sub(currentTime) < time.Hour {
		err := h.wspLogin.RefreshWspToken()
		if err != nil {
			zklogger.Error(ClusterTokenHandlerLogTag, "Error while refreshing wsp token :", err)
		} else {
			err := h.updateExpiry()
			if err != nil {
				zklogger.Error(ClusterTokenHandlerLogTag, "Error while updating expiry :", err)
			}
		}
	}
	h.ResetTickerTime()
}

func (h *ClusterTokenHandler) updateExpiry() error {
	err := h.validateKey.ValidateKeyWithZkCloud()
	if err != nil {
		zklogger.Error(ClusterTokenHandlerLogTag, "Error while validating wsp token :", err)
		return err
	} else {
		h.expiryTime = time.Now().Add(h.validateKey.ttl)
	}
	return nil
}

// ResetTickerTime resets the ticker's time interval
func (h *ClusterTokenHandler) ResetTickerTime() {
	newInterval := minimumInterval

	// If we are not forcing the reset, then we need to check the remaining time
	remainingTime := h.expiryTime.Sub(time.Now())
	if remainingTime < 30*time.Minute {
		newInterval = minimumInterval
	} else if remainingTime < time.Hour {
		newInterval = 5 * time.Minute
	} else if remainingTime < 6*time.Hour {
		newInterval = 15 * time.Minute
	} else {
		newInterval = 30 * time.Minute
	}

	// Check if the interval is different before resetting
	if h.timerInterval != newInterval {
		h.timerInterval = newInterval
		if h.ticker != nil {
			h.ticker.Stop()
		}
		h.ticker = zktick.GetNewTickerTask("ClusterTokenHandler", h.timerInterval, h.CheckExpiryAndUpdateClusterToken)
		h.ticker.Start()
	}
}
