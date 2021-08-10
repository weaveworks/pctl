package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

const v1 = "/v1"

// NewFromOptions creates a new Client from the supplied options
func NewFromOptions(options ServiceOptions) (*Client, error) {
	config, err := clientcmd.BuildConfigFromFlags("", options.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create config from kubeconfig path %q: %w", options.KubeconfigPath, err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &Client{
		clientset:      clientset,
		serviceOptions: options,
	}, nil
}

// ServiceOptions holds options to connect to the catalog service
type ServiceOptions struct {
	KubeconfigPath string
	Namespace      string
	ServiceName    string
	ServicePort    string
}

// StatusError represents an HTTP status error
type StatusError struct {
	wrapped *errors.StatusError
}

// Code returns the HTTP status code
func (e *StatusError) Code() int32 {
	return e.wrapped.ErrStatus.Code
}

// Error implements error
func (e *StatusError) Error() string {
	return e.wrapped.Error()
}

// Client is a catalog client
type Client struct {
	clientset      *kubernetes.Clientset
	serviceOptions ServiceOptions
}

// DoRequest sends a request to the catalog service
func (c *Client) DoRequest(path string, query map[string]string) ([]byte, int, error) {
	o := c.serviceOptions
	u, err := url.Parse(path)
	if err != nil {
		return nil, 0, err
	}
	u.Path = filepath.Join(v1, u.Path)
	responseWrapper := c.clientset.CoreV1().Services(o.Namespace).ProxyGet("http", o.ServiceName, o.ServicePort, u.String(), query)
	data, err := responseWrapper.DoRaw(context.TODO())
	if err != nil {
		if se, ok := err.(*errors.StatusError); ok {
			return nil, int(se.Status().Code), nil
		}
		return nil, 0, err
	}
	return data, http.StatusOK, nil
}
