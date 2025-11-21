package cfgutils

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
)

type CloudConfigItem interface {
	section() string
	addToCloudConfigFile() ([]any, error)
}

// structure for storing items that correspond to cloud config userdata file items from section 'runcmd'
type cloudConfigItemRunCmd struct {
	commands []string
}

func NewCloudConfigItemRunCmd(cmds []string) cloudConfigItemRunCmd {
	return cloudConfigItemRunCmd{cmds}
}

func (c cloudConfigItemRunCmd) addToCloudConfigFile() ([]any, error) {
	list := []any{}
	for _, cmd := range c.commands {
		list = append(list, cmd)
	}
	return list, nil
}

func (c cloudConfigItemRunCmd) section() string {
	return "runcmd"
}

// structure for storing items that corresponds to cloud config userdata file items from section 'write_files'
type cloudConfigItemWriteFiles struct {
	encoding    string
	content     string
	permissions string
	path        string
}

func NewCloudConfigItemWriteFiles(path, content string) cloudConfigItemWriteFiles {
	return cloudConfigItemWriteFiles{
		encoding:    "gzip+b64",
		content:     content,
		permissions: "0664",
		path:        path,
	}
}

func (c cloudConfigItemWriteFiles) addToCloudConfigFile() ([]any, error) {
	zippedContent, err := gzipEncode([]byte(c.content))
	b64Encoded := base64.StdEncoding.EncodeToString(zippedContent)
	if err != nil {
		return []any{}, err
	}
	return []any{
		map[string]string{
			"encoding":    c.encoding,
			"content":     b64Encoded,
			"permissions": c.permissions,
			"path":        c.path,
		}}, nil
}

func (c cloudConfigItemWriteFiles) section() string {
	return "write_files"
}

// gzipEncode Returns input data packed/compressed with gzip
func gzipEncode(data []byte) ([]byte, error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	gz.Flush()
	if _, err := gz.Write(data); err != nil {
		return []byte{}, err
	}
	if err := gz.Close(); err != nil {
		return []byte{}, err
	}

	return b.Bytes(), nil
}
