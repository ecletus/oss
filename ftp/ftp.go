package ftp

import (
	"github.com/secsy/goftp"
	"time"
	"io"
	"path/filepath"
	"github.com/aghape/oss"
	"io/ioutil"
	"os"
	"strings"
)

type Config struct {
	Hosts              []string
	RootDir            string
	User               string
	Password           string
	Endpoint           string
	ConnectionsPerHost int
	// value in seconds
	Timeout int64
}

type Client struct {
	Config Config
	Client *goftp.Client
}

func New(config Config) (*Client, error) {
	client, err := goftp.DialConfig(goftp.Config{
		User:               config.User,
		Password:           config.Password,
		Timeout:            time.Duration(config.Timeout) * time.Second,
		ConnectionsPerHost: config.ConnectionsPerHost,
	}, config.Hosts...)

	if err != nil {
		return nil, err
	}

	if config.RootDir != "" {
		config.RootDir = strings.TrimPrefix(config.RootDir, "/")
	}

	if config.Endpoint != "" {
		config.Endpoint = strings.TrimSuffix(config.Endpoint, "/")
	}

	return &Client{config, client}, nil
}

// Get receive file with given path
func (client Client) Path(path string) string {
	if path[0:2] == "//" {
		ep := client.Config.Endpoint
		for _, prefix := range []string{"http:", "https:"} {
			ep = strings.TrimPrefix(ep, prefix)
		}
		path = strings.TrimPrefix(path, ep)
	}
	path = strings.Trim(path, "/")
	if client.Config.RootDir == "" {
		return path
	}
	return filepath.Join(client.Config.RootDir, path)
}

// Get receive file with given path
func (client Client) Get(path string) (file *os.File, err error) {
	path = client.Path(path)

	if file, err = ioutil.TempFile("/tmp", "s3"); err == nil {
		err = client.Client.Retrieve(path, file)
		if err == nil {
			file.Seek(0, 0)
			return file, nil
		} else {
			file.Close()
		}
	}

	return nil, err
}

// Put store a reader into given path
func (client Client) MkdirAll(path string) error {
	parts := strings.Split(path, "/")
	var dir, p, tmp string
	i := 0

	for i, p = range parts {
		tmp = dir
		if tmp != "" {
			tmp += "/" + p
		} else {
			tmp = p
		}
		_, err := client.Client.Stat(tmp)

		if err != nil {
			if ftpErr, ok := err.(goftp.Error); ok && ftpErr.Code() == 550 {
				break
			} else {
				return err
			}
		} else {
			dir = tmp
			i++
		}
	}

	for _, p = range parts[i:len(parts)] {
		if dir != "" {
			dir += "/" + p
		} else {
			dir = p
		}
		_, err := client.Client.Mkdir(dir)

		if err != nil {
			return err
		}
	}

	return nil
}

// Put store a reader into given path
func (client Client) Put(path string, reader io.Reader) (*oss.Object, error) {
	if seeker, ok := reader.(io.ReadSeeker); ok {
		seeker.Seek(0, 0)
	}

	rpath := client.Path(path)
	err := client.MkdirAll(filepath.Dir(rpath))

	if err != nil {
		return nil, err
	}

	err = client.Client.Store(rpath, reader)

	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &oss.Object{
		Path:             path,
		Name:             filepath.Base(path),
		LastModified:     &now,
		StorageInterface: client,
	}, err
}

// Delete delete file
func (client Client) Stat(path string) (info os.FileInfo, notFound bool, err error) {
	stat, err := client.Client.Stat(client.Path(path))
	if err != nil {
		if ftpError, ok := err.(goftp.Error); ok && ftpError.Code() == 550 {
			return nil, true, nil
		}
	}
	return stat, false, err
}

// Delete delete file
func (client Client) Delete(path string) error {
	return client.Client.Delete(client.Path(path))
}

// List list all objects under current path
func (client Client) List(path string) ([]*oss.Object, error) {
	var objects []*oss.Object
	items, err := client.Client.ReadDir(client.Path(path))

	if err == nil {
		for _, content := range items {
			t := content.ModTime()
			objects = append(objects, &oss.Object{
				Path:             filepath.Join(path, content.Name()),
				Name:             content.Name(),
				LastModified:     &t,
				StorageInterface: client,
			})
		}
	}

	return objects, err
}

// GetEndpoint get endpoint, FileSystem's endpoint is /
func (client Client) GetEndpoint() string {
	return client.Config.Endpoint
}

func (client Client) GetURL(p ...string) string {
	if len(p) > 0 {
		return client.Config.Endpoint + "/" + strings.TrimPrefix(strings.Join(p, "/"), "/")
	}
	return client.Config.Endpoint
}
