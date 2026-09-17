package seedutils

import (
	"bytes"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/fujitsu/docker-machine-driver-fsas/httputils"
	slog "github.com/fujitsu/docker-machine-driver-fsas/logger"
)

const (
	seedEndpointPrefix = "/upload"
	seedEndpointHealth = "/health"
)

var (
	isInit                = false
	ErrEmptySeedServerUrl = errors.New("seed server URL cannot be empty")
)

// SeedManager interface defines the methods for publishing cloud-init artifacts to the seed server.
type SeedManager interface {
	IsInit() bool
	IsActive() error
	PublishFile(dmiSystemUUID, ip string, filename ConfigFiles, content []byte) error
	CleanupFolderWithConfigFiles(dmiSystemUUID, ip string) error
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

// IsActive checks if the seed manager is active
func (s *StandardSeedManager) IsActive() error {
	if !isInit {
		return errors.New("seed manager is not initialized")
	}

	queryParams := map[string]string{}
	headers := map[string]string{}
	statusCode, err := s.cdiClient.Get(seedEndpointHealth, queryParams, nil, headers)
	if err != nil {
		return fmt.Errorf("error while getting health check: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("Error from server: Status code: %d", statusCode)
	}

	slog.Info("seed server is active", "status_code", statusCode)

	return nil
}

type ConfigFiles string

const (
	UserDataFileName      ConfigFiles = "user-data"
	MetaDataFileName      ConfigFiles = "meta-data"
	NetworkConfigFileName ConfigFiles = "network-config"
)

func (c ConfigFiles) String() string {
	return string(c)
}

func (s *StandardSeedManager) PublishFile(dmiSystemUUID, ip string, filename ConfigFiles, content []byte) error {
	if dmiSystemUUID == "" {
		return errors.New("DMI.system-uuid cannot be empty")
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Text field with DMI.system-uuid
	err := writer.WriteField("dmi-system-uuid", dmiSystemUUID)
	if err != nil {
		return fmt.Errorf("error while writing DMI.system-uuid: %w", err)
	}

	// Text field with IP address
	err = writer.WriteField("ip", ip)
	if err != nil {
		return fmt.Errorf("error while writing IP: %w", err)
	}

	// File field
	part, err := writer.CreateFormFile("file", filename.String())
	if err != nil {
		return fmt.Errorf("error while creating form file: %w", err)
	}

	_, err = part.Write(content)
	if err != nil {
		return fmt.Errorf("error while writing file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("error while closing multipart writer: %w", err)
	}

	headers := map[string]string{
		"Content-Type": writer.FormDataContentType(),
	}

	statusCode, err := s.cdiClient.Post(body.Bytes(), seedEndpointPrefix, nil, nil, headers)
	if err != nil {
		return fmt.Errorf("error while sending POST request to endpoint: %s; error: %w", seedEndpointPrefix, err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("error while sending POST request to endpoint: %s; status code: %d", seedEndpointPrefix, statusCode)
	}

	slog.Info("upload succeeded", "file", filename, "status_code", statusCode)
	return nil
}

func (s *StandardSeedManager) CleanupFolderWithConfigFiles(dmiSystemUUID, ip string) error {
	endpoint := fmt.Sprintf("/%s", dmiSystemUUID)
	headers := map[string]string{}

	statusCode, err := s.cdiClient.Delete(endpoint, nil, nil, headers)
	if err != nil {
		return fmt.Errorf("error while sending DELETE request to endpoint: %s; error: %w", endpoint, err)
	}
	if statusCode != http.StatusNoContent {
		return fmt.Errorf("error while sending DELETE request to endpoint: %s; Status code: %d", endpoint, statusCode)
	}

	slog.Info("Folder with config files successfully cleaned up", "folder", dmiSystemUUID, "status_code", statusCode)
	return nil
}
