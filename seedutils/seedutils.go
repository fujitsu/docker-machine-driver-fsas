package seedutils

import (
	"bytes"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/fujitsu/docker-machine-driver-fsas/httputils"
	slog "github.com/fujitsu/docker-machine-driver-fsas/logger"
)

const (
	seedEndpointPrefix = "/upload"
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
	PublishFile(machineUUID string, filename ConfigFiles, content []byte) error
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

type ConfigFiles string

const (
	UserDataFileName      ConfigFiles = "user-data"
	MetaDataFileName      ConfigFiles = "meta-data"
	NetworkConfigFileName ConfigFiles = "network-config"
)

func (c ConfigFiles) String() string {
	return string(c)
}

func (s *StandardSeedManager) PublishFile(machineUUID string, filename ConfigFiles, content []byte) error {
	if machineUUID == "" {
		return errors.New("machine UUID cannot be empty")
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Text field with machine UUID
	err := writer.WriteField("mach_uuid", machineUUID)
	if err != nil {
		return fmt.Errorf("error while writing machine UUID: %w", err)
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
		return fmt.Errorf("error while posting file: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("Error from server: Status code: %d", statusCode)
	}

	slog.Info("upload succeeded", "file", filename, "status_code", statusCode)
	return nil
}

func SendFormAndFile(machineUUID, filename string) {
	fmt.Println("send file to remote server")
	// TODO: validate machineUUID

	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Text field with machine UUID
	err = writer.WriteField("mach_uuid", machineUUID)
	if err != nil {
		panic(err)
	}

	// File field
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		panic(err)
	}

	_, err = file.WriteTo(part)
	if err != nil {
		panic(err)
	}

	// IMPORTANT: finish the multipart body
	err = writer.Close()
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"http://localhost:8501/upload",
		&body,
	)
	if err != nil {
		panic(err)
	}

	// Contains the boundary, so don't manually set this to just
	// "multipart/form-data".
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error while sending request:", "error:", err)
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error from server:", "Status code:", resp.StatusCode, "Status:", resp.Status)
	}
	defer resp.Body.Close()

	fmt.Println("response status=", resp.Status)
}
