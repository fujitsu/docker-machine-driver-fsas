package cfgutils

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"

	slog "github.com/fujitsu/docker-machine-driver-fsas/logger"
	"gopkg.in/yaml.v3"
)

const (
	defaultFilePermissions = os.FileMode(0644)
	cloudConfigHeader      = "#cloud-config"
)

/*
cloudInitFile is a cloud-init payload.
SSHPasswordAuth uses *bool with omitempty to preserve 3 states in YAML:
nil (omit), true, and false.
*/
type cloudInitFile struct {
	Hostname        string                      `yaml:"hostname,omitempty"`
	SSHPasswordAuth *bool                       `yaml:"ssh_pwauth,omitempty"`
	Users           []cloudConfigItemUsers      `yaml:"users,omitempty"`
	RunCmds         []string                    `yaml:"runcmd,omitempty"`
	WriteFiles      []CloudConfigItemWriteFiles `yaml:"write_files,omitempty"`
}

// cloudInitFileOption applies a single optional setting to cloudInitFile.
type cloudInitFileOption func(*cloudInitFile) error

// WithHostname sets the hostname field.
func WithHostname(hostname string) cloudInitFileOption {
	return func(c *cloudInitFile) error {
		c.Hostname = hostname
		return nil
	}
}

// WithSSHPasswordAuth sets the ssh_pwauth field.
func WithSSHPasswordAuth(auth *bool) cloudInitFileOption {
	return func(c *cloudInitFile) error {
		c.SSHPasswordAuth = auth
		return nil
	}
}

// WithUsers sets the users field. Returns an error when the provided slice is empty.
func WithUsers(users []cloudConfigItemUsers) cloudInitFileOption {
	return func(c *cloudInitFile) error {
		if len(users) > 0 {
			c.Users = users
		} else {
			return errors.New("section 'users' cannot be empty")
		}
		return nil
	}
}

// WithRunCmds sets the runcmd field. Returns an error when the provided slice is empty.
func WithRunCmds(cmds []string) cloudInitFileOption {
	return func(c *cloudInitFile) error {
		if len(cmds) > 0 {
			c.RunCmds = cmds
		} else {
			return errors.New("section 'runcmd' cannot be empty")
		}
		return nil
	}
}

// WithWriteFiles sets the write_files field. Returns an error when the provided slice is empty.
func WithWriteFiles(files []CloudConfigItemWriteFiles) cloudInitFileOption {
	return func(c *cloudInitFile) error {
		if len(files) > 0 {
			c.WriteFiles = files
		} else {
			return errors.New("section 'write_files' cannot be empty")
		}
		return nil
	}
}

// NewCloudInitFile builds cloudInitFile from functional options.
func NewCloudInitFile(opts ...cloudInitFileOption) (cloudInitFile, error) {
	cif := cloudInitFile{}
	for _, opt := range opts {
		if err := opt(&cif); err != nil {
			return cloudInitFile{}, err
		}

	}
	return cif, nil
}

// writeFilesConfig stores defaults and overrides for write_files entries.
type writeFilesConfig struct {
	encoding    string
	permissions fs.FileMode
}

// options configures writeFilesConfig.
type options func(*writeFilesConfig)

// SetCustomPermissions overrides the default mode used by write_files entries.
func SetCustomPermissions(permissions fs.FileMode) options {
	return func(c *writeFilesConfig) {
		c.permissions = permissions
	}
}

// cloudConfigItemUsers defines a single user entry for cloud-init users section.
type cloudConfigItemUsers struct {
	Name              string   `yaml:"name"`
	SSHAuthorizedKeys []string `yaml:"ssh_authorized_keys"`
}

// NewCloudConfigItemUsers creates a users section entry.
func NewCloudConfigItemUsers(name string, keys []string) cloudConfigItemUsers {
	return cloudConfigItemUsers{
		Name:              name,
		SSHAuthorizedKeys: keys,
	}
}

// cloudConfigItemRunCmd stores run commands for cloud-init.
type cloudConfigItemRunCmd struct {
	cmds []string
}

// NewCloudConfigItemRunCmd creates a runcmd module entry.
func NewCloudConfigItemRunCmd(cmds []string) cloudConfigItemRunCmd {
	return cloudConfigItemRunCmd{cmds: cmds}
}

// CloudConfigItemWriteFiles defines one write_files entry for cloud-init.
type CloudConfigItemWriteFiles struct {
	Encoding    string
	Content     string
	Permissions string
	Path        string
}

// NewCloudConfigItemWriteFiles creates a write_files module entry.
func NewCloudConfigItemWriteFiles(path, content string, opts ...options) CloudConfigItemWriteFiles {

	cfg := &writeFilesConfig{
		encoding:    "gzip+b64",
		permissions: defaultFilePermissions,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return CloudConfigItemWriteFiles{
		Encoding:    cfg.encoding,
		Content:     content,
		Permissions: fmt.Sprintf("%04o", cfg.permissions),
		Path:        path,
	}
}

// extendUserdata extends cloud-config user-data content in place.
func extendUserdata(userDataFile string, cif cloudInitFile) error {

	userdata, err := os.ReadFile(userDataFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			slog.Error("User data file does not exist", "path", userDataFile, "err", err)
		} else {
			slog.Error("User data cannot be read", "path", userDataFile, "err", err)
		}
		return err
	}

	var cloudInitFile cloudInitFile
	if err := yaml.Unmarshal(userdata, &cloudInitFile); err != nil {
		slog.Error("Unmarshal error. Failed to parse user data as YAML", "path", userDataFile, "err", err)
		slog.Debug("Unmarshalling error details", "yaml-content", string(userdata),
			"new-content", fmt.Sprintf("%+v", cif))
		return err
	}

	if err := updateCloudConfigFileStruct(&cloudInitFile, &cif); err != nil {
		return err
	}

	data, err := yaml.Marshal(cloudInitFile)
	if err != nil {
		return fmt.Errorf("failed to marshal cloud init file: %w", err)
	}

	trimmed := bytes.TrimSpace(data)
	if !bytes.HasPrefix(trimmed, []byte(cloudConfigHeader)) {
		trimmed = append([]byte(cloudConfigHeader+"\n"), trimmed...)
	}

	if err := os.WriteFile(userDataFile, trimmed, os.FileMode(0644)); err != nil {
		slog.Error("Failed to write userdata file", "path", userDataFile, "err", err)
		return err
	}
	return nil
}

// updateCloudConfigFileStruct merges values from new into current.
func updateCloudConfigFileStruct(current, new *cloudInitFile) error {
	if new.Hostname != "" {
		current.Hostname = new.Hostname
	}
	if new.SSHPasswordAuth != nil {
		current.SSHPasswordAuth = new.SSHPasswordAuth
	}
	if len(new.RunCmds) > 0 {
		current.RunCmds = append(current.RunCmds, new.RunCmds...)
	}
	if len(new.Users) > 0 {
		current.Users = append(current.Users, new.Users...)
	}
	if len(new.WriteFiles) > 0 {
		writeFilesPackedAndEncoded, err := getWriteFilesPackedAndEncoded(new.WriteFiles)
		if err != nil {
			return fmt.Errorf("failed to pack and encode write_files content: %w", err)
		}
		current.WriteFiles = append(current.WriteFiles, writeFilesPackedAndEncoded...)
	}
	return nil
}

// getWriteFilesPackedAndEncoded gzip-compresses and base64-encodes write_files content.
func getWriteFilesPackedAndEncoded(writeFiles []CloudConfigItemWriteFiles) ([]CloudConfigItemWriteFiles, error) {
	var zippedContent []byte
	var err error
	updatedSlice := make([]CloudConfigItemWriteFiles, 0)
	for _, i := range writeFiles {
		zippedContent, err = gzipEncode([]byte(i.Content))
		if err != nil {
			slog.Error("Failed to gzip content for write_files", "path", i.Path, "err", err)
			return nil, err
		}
		b64Encoded := base64.StdEncoding.EncodeToString(zippedContent)
		updatedSlice = append(updatedSlice, CloudConfigItemWriteFiles{
			Path:        i.Path,
			Content:     b64Encoded,
			Encoding:    i.Encoding,
			Permissions: i.Permissions,
		})
	}

	return updatedSlice, nil
}

// gzipEncode compresses input data with gzip.
func gzipEncode(data []byte) ([]byte, error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	gz.Flush()

	if _, err := gz.Write(data); err != nil {
		return nil, err
	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
