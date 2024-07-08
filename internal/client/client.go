package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"time"

	"github.com/observiq/bindplane-op-action/internal/client/config"
	"github.com/observiq/bindplane-op-action/internal/client/model"
	"github.com/observiq/bindplane-op-action/internal/client/version"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

const (
	KeyHeader = "X-Bindplane-Api-Key"

	DefaultTimeout = time.Second * 60
)

// Option is a functional option for configuring the BindPlane client
type Option func(*BindPlane) error

// WithTimeout sets the timeout for the BindPlane client. If a zero value is provided,
// this function will no-op.
func WithTimeout(timeout time.Duration) Option {
	return func(c *BindPlane) error {
		if timeout == 0 {
			return nil
		}

		c.client.SetTimeout(timeout)
		return nil
	}
}

type BindPlane struct {
	logger *zap.Logger
	config *config.Config
	client *resty.Client
}

// NewBindPlane takes a config and logger and optional options. Returns a configured BindPlane client.
// Options are not required, the client will be returned with default values if no options are specified.
func NewBindPlane(config *config.Config, logger *zap.Logger, opts ...Option) (*BindPlane, error) {
	restryClient := resty.New()
	restryClient.SetDisableWarn(true)
	restryClient.SetTimeout(DefaultTimeout)

	restryClient.SetBasicAuth(config.Auth.Username, config.Auth.Password)

	if config.Auth.APIKey != "" {
		restryClient.SetHeader(KeyHeader, config.Auth.APIKey)
	}

	restryClient.SetBaseURL(fmt.Sprintf("%s/v1", config.Network.RemoteURL))

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}
	if len(config.Network.CertificateAuthority) > 0 {
		tlsConfig.RootCAs = x509.NewCertPool()
		for _, ca := range config.Network.CertificateAuthority {
			if ok := tlsConfig.RootCAs.AppendCertsFromPEM([]byte(ca)); !ok {
				return nil, fmt.Errorf("failed to append certificate authority")
			}
		}
	}

	restryClient.SetTLSClientConfig(tlsConfig)

	b := &BindPlane{
		logger: logger,
		config: config,
		client: restryClient,
	}

	for _, opt := range opts {
		if err := opt(b); err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}

	return b, nil
}

// Version queries the BindPlane API for the version information
func (b *BindPlane) Version(_ context.Context) (version.Version, error) {
	v := version.Version{}
	_, err := b.client.R().SetResult(&v).Get("/version")
	return v, err
}

// Apply applies a list of resources to the BindPlane API
func (c *BindPlane) Apply(_ context.Context, resources []*model.AnyResource) ([]*model.AnyResourceStatus, error) {
	payload := model.ApplyPayload{
		Resources: resources,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("client apply: %w", err)
	}

	ar := &model.ApplyResponseClientSide{}
	resp, err := c.client.R().SetHeader("Content-Type", "application/json").SetBody(data).SetResult(ar).Post("/apply")
	if err != nil {
		return nil, fmt.Errorf("failed to apply file: %w", err)
	}

	status := resp.StatusCode()
	if status > 399 {
		return nil, fmt.Errorf("BindPlane API returned status %d: %s", status, resp.String())
	}

	return ar.Updates, nil
}

// Configuration queries the BindPlane API and returns a configuration by name
func (c *BindPlane) Configuration(_ context.Context, name string) (*model.Configuration, error) {
	pr, err := c.configuration(name)
	return pr.Configuration, err
}

// RawConfiguration queries the BindPlane API and returns a raw configuration by name
func (c *BindPlane) RawConfiguration(_ context.Context, name string) (string, error) {
	pr, err := c.configuration(name)
	return pr.Raw, err
}

func (c *BindPlane) configuration(name string) (*model.ConfigurationResponse, error) {
	pr := &model.ConfigurationResponse{}
	resp, err := c.client.R().SetResult(pr).Get(fmt.Sprintf("/configurations/%s", name))
	if err != nil {
		return nil, err
	}

	status := resp.StatusCode()
	if status > 399 {
		return nil, fmt.Errorf("BindPlane API returned status %d: %s", status, resp.String())
	}

	return pr, nil
}

// StartRollout starts a rollout by name
// NOTE: Does not use context or rollout options unlike the original client implementation
// NOTE: Returns only an error, not a configuration
func (c *BindPlane) StartRollout(name string) error {
	endpoint := fmt.Sprintf("/rollouts/%s/start", name)

	body := model.StartRolloutPayload{
		Options: &model.RolloutOptions{},
	}

	resp, err := c.client.R().
		SetBody(body).
		Post(endpoint)
	if err != nil {
		return err
	}

	status := resp.StatusCode()
	if status > 399 {
		return fmt.Errorf("BindPlane API returned status %d: %s", status, resp.String())
	}

	return nil
}

// RolloutStatus queries the BindPlane API for the status of a rollout by configuration name
func (c *BindPlane) RolloutStatus(name string) (*model.Configuration, error) {
	var response model.ConfigurationResponse
	endpoint := fmt.Sprintf("/rollouts/%s/status", name)

	resp, err := c.client.R().
		SetResult(&response).
		Get(endpoint)
	if err != nil {
		return nil, err
	}

	status := resp.StatusCode()
	if status > 399 {
		return nil, fmt.Errorf("BindPlane API returned status %d: %s", status, resp.String())
	}

	return response.Configuration, nil
}
