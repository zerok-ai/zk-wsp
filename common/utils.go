package common

import (
	"fmt"
	"github.com/zerok-ai/zk-wsp"
	"math/rand"
	"net/http"
	"net/url"
	"time"
)

func GetValueWithTimeout(ch <-chan *WriteConnection, timeout time.Duration) (*WriteConnection, error) {
	select {
	case value := <-ch:
		return value, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout occurred")
	}
}

func GetDestinationUrl(w http.ResponseWriter, r *http.Request) (*url.URL, error) {
	dstURL := r.Header.Get("X-PROXY-DESTINATION")
	if dstURL == "" {
		wsp.ProxyErrorf(w, "Missing X-PROXY-DESTINATION header")
		return nil, fmt.Errorf("missing X-PROXY-DESTINATION header")
	}
	URL, err := url.Parse(dstURL)
	if err != nil {
		wsp.ProxyErrorf(w, "Unable to parse X-PROXY-DESTINATION header")
		return nil, fmt.Errorf("unable to parse X-PROXY-DESTINATION header")
	}
	return URL, nil
}

func GenerateRandomNumber(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min+1) + min
}
