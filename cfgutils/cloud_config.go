package cfgutils

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
)

const defaultWriteFilePermissions = os.FileMode(0644)

type options func(*config)
type config struct {
	encoding    string
	permissions fs.FileMode
}

func IsExecutable() options {
	return func(c *config) {
		c.permissions = os.FileMode(0744)
	}
}

type CloudConfigItem interface {
	getModuleName() string
	getNewCloudConfigContent() ([]interface{}, error)
}

// structure for storing items that correspond to cloud config userdata file items from module 'runcmd'

type cloudConfigItemRunCmd struct {
	commands []string
}

func NewCloudConfigItemRunCmd(cmds []string) cloudConfigItemRunCmd {
	return cloudConfigItemRunCmd{cmds}
}

func (c cloudConfigItemRunCmd) getNewCloudConfigContent() ([]interface{}, error) {
	ccItems := make([]interface{}, len(c.commands))
	for i, cmd := range c.commands {
		ccItems[i] = cmd
	}
	return ccItems, nil
}

func (c cloudConfigItemRunCmd) getModuleName() string {
	return "runcmd"
}

// structure for storing items that corresponds to cloud config userdata file items from module 'write_files'

type cloudConfigItemWriteFiles struct {
	encoding    string
	content     string
	permissions string
	path        string
}

func NewCloudConfigItemWriteFiles(path, content string, opts ...options) cloudConfigItemWriteFiles {

	cfg := &config{
		encoding:    "gzip+b64",
		permissions: defaultWriteFilePermissions,
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

func (c cloudConfigItemWriteFiles) getNewCloudConfigContent() ([]interface{}, error) {
	zippedContent, err := gzipEncode([]byte(c.content))
	if err != nil {
		return nil, err
	}
	b64Encoded := base64.StdEncoding.EncodeToString(zippedContent)
	return []interface{}{

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
