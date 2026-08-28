package seedutils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fujitsu/docker-machine-driver-fsas/httputils"
	slog "github.com/fujitsu/docker-machine-driver-fsas/logger"
)

const (
	seedEndpointPrefix = "/seed"
	userDataName       = "user-data"
	metadataName       = "meta-data"
	networkConfigName  = "network-config"
)

var (
	isInit                = false
	ErrEmptySeedServerUrl = errors.New("seed server URL cannot be empty")
)

// SeedManager interface defines the methods for publishing cloud-init artifacts to the seed server.
type SeedManager interface {
	IsInit() bool
	PublishUserData(content string) error
	PublishMetadata(content string) error
	PublishNetworkConfig(content string) error
}

// StandardSeedManager struct holds configuration for seed server interaction.
type StandardSeedManager struct {
	cdiClient httputils.CdiHTTPClient
}

// This makes StandardSeedManager implement the SeedManager interface
var _ SeedManager = (*StandardSeedManager)(nil)

// NewStandardSeedManager returns a new instance of Standard Seed Manager.
func NewStandardSeedManager(seedServerUrl string) (*StandardSeedManager, error) {
	slog.Debug("Creating StandardSeedManager", "seedServerUrl", seedServerUrl)
	if seedServerUrl == "" {
		return nil, ErrEmptySeedServerUrl
	}

	isInit = true
	return &StandardSeedManager{
		cdiClient: httputils.NewStandardCdiHTTPClient(strings.TrimSuffix(seedServerUrl, "/")),
	}, nil
}

// IsInit Returns true if constructor succeeded else false
func (s *StandardSeedManager) IsInit() bool {
	return isInit
}

// publish uploads a single cloud-init artifact to the seed server via HTTP PUT.
func (s *StandardSeedManager) publish(name, content string) error {
	endpoint := fmt.Sprintf("%s/%s", seedEndpointPrefix, name)
	headers := map[string]string{"Content-Type": "text/plain"}

	statusCode, err := s.cdiClient.Put([]byte(content), endpoint, nil, nil, headers)
	if err != nil {
		slog.Error("Failed to publish cloud-init artifact to seed server", "name", name, "err", err)
		return err
	}

	slog.Info("Successfully published cloud-init artifact to seed server", "name", name, "status_code", statusCode)
	return nil
}

func (s *StandardSeedManager) PublishUserData(content string) error {
	return s.publish(userDataName, content)
}

func (s *StandardSeedManager) PublishMetadata(content string) error {
	return s.publish(metadataName, content)
}

func (s *StandardSeedManager) PublishNetworkConfig(content string) error {
	return s.publish(networkConfigName, content)
}
