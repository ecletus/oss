package ftp

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ecletus/helpers"
	"github.com/moisespsena-go/assetfs"

	"github.com/ecletus/oss"
	"github.com/ecletus/oss/factories"
	"github.com/secsy/goftp"
)

func init() {
	factories.Registry("fs", factories.StorageFactoryFunc(func(ctx *factories.Context, config map[string]interface{}) (storage oss.StorageInterface, err error) {
		var cfg Config
		if err = helpers.ParseMap(config, &cfg); err != nil {
			return nil, err
		}

		if ctx.Var != nil {
			ctx.Var.FormatPathPtr(&cfg.RootDir).
				FormatPtr(&cfg.Endpoint.Path, &cfg.Endpoint.Host, &cfg.User, &cfg.Password)

			for i := range cfg.Hosts {
				ctx.Var.FormatPtr(&cfg.Hosts[i])
			}
		}

		return New(cfg)
	}))
}

type Config struct {
	Hosts              []string
	RootDir            string
	User               string
	Password           string
	Endpoint           oss.Endpoint
	ConnectionsPerHost int
	// value in seconds
	Timeout int64
}

type Client struct {
	Config Config
	Client *goftp.Client
	fs     http.FileSystem
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

	if config.Endpoint.Path != "" {
		config.Endpoint.Path = strings.TrimSuffix(config.Endpoint.Path, "/")
	}
	c := &Client{config, client, nil}
	return c, nil
}

func (client Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	file, err := client.Get(r.URL.Path)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	s, err := file.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, r.URL.Path, s.ModTime(), file)
}

// Get receive file with given path
func (client Client) Path(path string) string {
	if path[0:2] == "//" {
		ep := client.Config.Endpoint.Path
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
func (client Client) GetEndpoint() *oss.Endpoint {
	return &client.Config.Endpoint
}

func (client Client) GetURL(p ...string) (url string) {
	url = client.Config.Endpoint.URL()
	if len(p) > 0 {
		url += "/" + strings.TrimPrefix(strings.Join(p, "/"), "/")
	}
	return
}

func (client Client) GetDynamicURL(scheme, host string, p ...string) (url string) {
	url = client.Config.Endpoint.DinamicURL(scheme, host)
	if len(p) > 0 {
		url += "/" + strings.TrimPrefix(strings.Join(p, "/"), "/")
	}
	return
}

func (this Client) AssetFS() (assetfs.Interface, error) {
	return nil, oss.ErrAssetFsUnavailable
}
