package common

import (
	"context"
	"fmt"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	"github.com/zerok-ai/zk-wsp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"net/url"
	"os"
	"time"
)

func getValueWithTimeoutInternal(ch <-chan *WriteConnection, timeout time.Duration) (*WriteConnection, error) {
	select {
	case value := <-ch:
		return value, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout occurred")
	}
}

func GetValueWithTimeout(ch <-chan *WriteConnection, timeout time.Duration) (*WriteConnection, error) {
	startTime := time.Now()
	for {
		conn, err := getValueWithTimeoutInternal(ch, timeout)
		if err != nil {
			return nil, err
		}
		//Checking the status here again because the connection might have been closed by the time we get it from the channel.
		if conn != nil && conn.Status == IDLE {
			return conn, nil
		}
		timeout = timeout - time.Since(startTime)
	}
}

func GetClientId(w http.ResponseWriter, r *http.Request) (string, error) {
	clientIdKey := "X-CLIENT-ID"
	clientId := r.Header.Get(clientIdKey)
	if clientId == "" {
		logger.Debug(LOG_TAG, "Missing clientId header, searching in query param")
		//Checking in query params.
		queryParams := r.URL.Query()
		clientId = queryParams.Get(clientIdKey)
		if clientId == "" {
			wsp.ProxyErrorf(w, "Missing clientId value")
			return "", fmt.Errorf("missing clientId value")
		}
	}
	return clientId, nil
}

func GetDestinationUrl(w http.ResponseWriter, r *http.Request) (*url.URL, error) {
	proxyDestinationKey := "X-PROXY-DESTINATION"
	dstURL := r.Header.Get(proxyDestinationKey)
	if dstURL == "" {
		logger.Debug(LOG_TAG, "Missing X-PROXY-DESTINATION header, searching in query param")
		//Checking in query params.
		queryParams := r.URL.Query()
		dstURL = queryParams.Get(proxyDestinationKey)

		if dstURL == "" {
			wsp.ProxyErrorf(w, "Missing X-PROXY-DESTINATION value")
			return nil, fmt.Errorf("missing X-PROXY-DESTINATION value")
		}
	}
	URL, err := url.Parse(dstURL)
	if err != nil {
		wsp.ProxyErrorf(w, "Unable to parse X-PROXY-DESTINATION header")
		return nil, fmt.Errorf("unable to parse X-PROXY-DESTINATION header")
	}
	return URL, nil
}

// TODO: Can we move this method and other methods in zk-operator k8sutils to utils-go?
func GetSecretValue(namespace, secretName, dataKey string) (string, error) {
	//logger.Debug(LOG_TAG, namespace, secretName, dataKey)
	clientSet, err := GetK8sClient()
	if err != nil {
		//logger.Error(LOG_TAG, " Error while getting k8s client.")
		return "", err
	}

	secret, err := clientSet.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		//logger.Error(LOG_TAG, "Failed to get secret: ", err)
		return "", err
	}

	value, ok := secret.Data[dataKey]

	if ok {
		//logger.Debug(LOG_TAG, dataKey, value)
		return string(value), nil
	}

	return "", fmt.Errorf("secret Value not found for %v and key %v", secretName, dataKey)
}

func GetK8sClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		// If incluster config failes, reading from kubeconfig.
		// However, this is not connecting to gcp clusters. Only working for kind now(probably minikube also).
		kubeconfig := os.Getenv("KUBECONFIG")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kubernetes config: %v", err)
		}
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func RespCodeIsOk(status int) bool {
	if status > 199 && status < 300 {
		return true
	}
	return false

}
