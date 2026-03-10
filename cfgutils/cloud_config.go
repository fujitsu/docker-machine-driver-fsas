package cfgutils

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
)

const defaultFilePermissions = os.FileMode(0644)

type writeFilesConfig struct {
	encoding    string
	permissions fs.FileMode
}
type options func(*writeFilesConfig)

func SetCustomPermissions(permissions fs.FileMode) options {
	return func(c *writeFilesConfig) {
		c.permissions = permissions
	}
}

type CloudConfigItem interface {
	getModuleName() string
	getNewCloudConfigContent() ([]any, error)
}

type cloudConfigItemBase struct {
	lines []string
}

func (b cloudConfigItemBase) getNewCloudConfigContent() ([]any, error) {
	ccItems := make([]any, len(b.lines))
	for i, line := range b.lines {
		ccItems[i] = line
	}
	return ccItems, nil
}

type cloudConfigItemUsers struct {
	users []cloudConfigUser `yaml:"users"`
}

type cloudConfigUser struct {
	Name              string                     `yaml:"name"`
	SSHAuthorizedKeys cloudConfigItemSshAuthKeys `yaml:"ssh_authorized_keys"`
}

func NewCloudConfigItemUsers(name string, keys []string) cloudConfigItemUsers {
	user := cloudConfigUser{
		Name:              name,
		SSHAuthorizedKeys: NewCloudConfigItemSshAuthKeys(keys),
	}

	return cloudConfigItemUsers{
		users: []cloudConfigUser{user},
	}
}

func (c cloudConfigItemUsers) getNewCloudConfigContent() ([]any, error) {
	ccItems := make([]any, len(c.users))
	for i, u := range c.users {
		ccItems[i] = u
	}
	return ccItems, nil
}

func (c cloudConfigItemUsers) getModuleName() string {
	return "users"
}

/*
	module 'ssh_authorized_keys'

Structure and methods for handling items from module 'ssh_authorized_keys'
*/
type cloudConfigItemSshAuthKeys struct {
	cloudConfigItemBase
}

func NewCloudConfigItemSshAuthKeys(cmds []string) cloudConfigItemSshAuthKeys {
	return cloudConfigItemSshAuthKeys{cloudConfigItemBase{lines: cmds}}
}

/*
The MarshalYAML method is needed because Go's YAML library (e.g., gopkg.in/yaml.v3)
requires custom marshaling for structs that don't have standard YAML tags or need to serialize
in a non-default way.

Without MarshalYAML, cloudConfigItemSshAuthKeys would serialize its embedded
cloudConfigItemBase fields (e.g., commands []string) as a nested map like
{"commands": ["key1", "key2"]}, which is invalid for cloud-init's ssh_authorized_keys
(expects a direct list of strings).

MarshalYAML overrides this by returning c.commands directly, ensuring the struct serializes
as ["key1", "key2"] under the ssh_authorized_keys field in cloudConfigUser.

This fixes the serialization error and produces correct YAML output. If the struct
had appropriate YAML tags on fields, it might not be needed, but here it's essential
for flattening the output.
*/
func (c cloudConfigItemSshAuthKeys) MarshalYAML() (any, error) {
	return c.lines, nil
}

func (c cloudConfigItemSshAuthKeys) getModuleName() string {
	return "ssh_authorized_keys"
}

/*
	module 'runcmd'

Structure and methods for handling items from module 'runcmd'
*/
type cloudConfigItemRunCmd struct {
	cloudConfigItemBase
}

func NewCloudConfigItemRunCmd(cmds []string) cloudConfigItemRunCmd {
	return cloudConfigItemRunCmd{cloudConfigItemBase{lines: cmds}}
}

func (c cloudConfigItemRunCmd) getModuleName() string {
	return "runcmd"
}

/*
	module 'write_files'

Structure and methods for handling items from module 'write_files'
*/

type cloudConfigItemWriteFiles struct {
	encoding    string
	content     string
	permissions string
	path        string
}

func NewCloudConfigItemWriteFiles(path, content string, opts ...options) cloudConfigItemWriteFiles {

	cfg := &writeFilesConfig{
		encoding:    "gzip+b64",
		permissions: defaultFilePermissions,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cloudConfigItemWriteFiles{
		encoding:    cfg.encoding,
		content:     content,
		permissions: fmt.Sprintf("%04o", cfg.permissions),
		path:        path,
	}
}

func (c cloudConfigItemWriteFiles) getNewCloudConfigContent() ([]any, error) {
	zippedContent, err := gzipEncode([]byte(c.content))
	if err != nil {
		return nil, err
	}
	b64Encoded := base64.StdEncoding.EncodeToString(zippedContent)
	return []any{
		map[string]string{
			"encoding":    c.encoding,
			"content":     b64Encoded,
			"permissions": c.permissions,
			"path":        c.path,
		}}, nil
}

func (c cloudConfigItemWriteFiles) getModuleName() string {
	return "write_files"
}

// gzipEncode Returns input data packed/compressed with gzip
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
