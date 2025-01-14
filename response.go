package wsp

import (
	"fmt"
	zklogger "github.com/zerok-ai/zk-utils-go/logs"
	"net/http"
)

var RESP_LOG_TAG = "Response"

// HTTPResponse is a serializable version of http.Response ( with only useful fields )
type HTTPResponse struct {
	StatusCode    int
	Header        http.Header
	ContentLength int64
}

// SerializeHTTPResponse create a new HTTPResponse from a http.Response
func SerializeHTTPResponse(resp *http.Response) *HTTPResponse {
	r := new(HTTPResponse)
	r.StatusCode = resp.StatusCode
	r.Header = resp.Header
	r.ContentLength = resp.ContentLength
	return r
}

// NewHTTPResponse creates a new HTTPResponse
func NewHTTPResponse() (r *HTTPResponse) {
	r = new(HTTPResponse)
	r.Header = make(http.Header)
	return
}

// ProxyError log error and return a HTTP 500 error with the message
func ProxyError(w http.ResponseWriter, err error) {
	zklogger.Debug(RESP_LOG_TAG, err)
	http.Error(w, err.Error(), 500)
}

// ProxyErrorf log error and return a HTTP 526 error with the message
func ProxyErrorf(w http.ResponseWriter, format string, args ...interface{}) {
	ProxyError(w, fmt.Errorf(format, args...))
}

// InvalidClusterError log error and return a HTTP 526 error with the message
func InvalidClusterError(w http.ResponseWriter, err error) {
	zklogger.Debug(RESP_LOG_TAG, err)
	http.Error(w, err.Error(), 526)
}

// InvalidClusterErrorf log error and return a HTTP 526 error with the message
func InvalidClusterErrorf(w http.ResponseWriter, format string, args ...interface{}) {
	InvalidClusterError(w, fmt.Errorf(format, args...))
}
