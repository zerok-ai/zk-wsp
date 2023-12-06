package client

import (
	"context"
	"encoding/base64"
	"fmt"
	logger "github.com/zerok-ai/zk-utils-go/logs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

var LOG_TAG = "k8sutils"

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

func UpdateSecretValue(namespace, secretName string, newData map[string]string) error {
	// Get the existing secret
	clientset, err := GetK8sClient()
	if err != nil {
		logger.Error(LOG_TAG, " Error while getting k8s client.")
		return err
	}
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret: %v", err)
	}

	for key, newValue := range newData {
		// Update the secret data
		secret.Data[key] = []byte(base64.StdEncoding.EncodeToString([]byte(newValue)))
	}

	// Update the secret in Kubernetes
	_, err = clientset.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update secret: %v", err)
	}

	return nil
}

func GetSecretValue(namespace, secretName, dataKey string) (string, error) {
	logger.Debug(LOG_TAG, namespace, secretName, dataKey)
	clientSet, err := GetK8sClient()
	if err != nil {
		logger.Error(LOG_TAG, " Error while getting k8s client.")
		return "", err
	}

	secret, err := clientSet.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		logger.Error(LOG_TAG, "Failed to get secret: ", err)
		return "", err
	}

	value, ok := secret.Data[dataKey]

	if ok {
		logger.Debug(LOG_TAG, dataKey, value)
		return string(value), nil
	}

	return "", fmt.Errorf("secret Value not found for %v and key %v", secretName, dataKey)
}
