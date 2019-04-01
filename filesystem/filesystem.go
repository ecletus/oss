package filesystem

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ecletus/helpers"

	"github.com/ecletus/oss/factories"

	"github.com/moisespsena-go/error-wrap"

	"github.com/moisespsena-go/path-helpers"

	"github.com/ecletus/oss"
)

func init() {
	factories.Registry("fs", factories.StorageFactoryFunc(func(ctx *factories.Context, config map[string]interface{}) (storage oss.StorageInterface, err error) {
		var cfg Config
		if err = helpers.ParseMap(config, &cfg); err != nil {
			return nil, err
		}
		if ctx.Var != nil {
			ctx.Var.FormatPathPtr(&cfg.RootDir)
			if cfg.Endpoint != nil {
				ctx.Var.FormatPtr(&cfg.Endpoint.Path, &cfg.Endpoint.Host)
			}
		}
		return New(&cfg), nil
	}))
}

type Config struct {
	RootDir  string
	Endpoint *oss.Endpoint
}

// FileSystem file system storage
type FileSystem struct {
	Base     string
	Endpoint oss.Endpoint
}

// New initialize FileSystem storage
func New(cfg *Config) *FileSystem {
	if cfg.RootDir != "" && cfg.RootDir[0] == '~' {
		if hpth, err := homedir.Expand(cfg.RootDir); err == nil {
			cfg.RootDir = hpth
		}
	}

	absbase, err := filepath.Abs(cfg.RootDir)
	if err != nil {
		fmt.Println("FileSystem storage's directory haven't been initialized")
	}
	if cfg.Endpoint == nil {
		cfg.Endpoint = &oss.Endpoint{Path: "!"}
	}
	return &FileSystem{Base: absbase, Endpoint: *cfg.Endpoint}
}

func (f FileSystem) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pth := f.GetFullPath(r.URL.Path)
	http.ServeFile(w, r, pth)
}

// GetFullPath get full path from absolute/relative path
func (fileSystem FileSystem) GetFullPath(path string) string {
	fullpath := path
	if !strings.HasPrefix(path, fileSystem.Base) {
		fullpath, _ = filepath.Abs(filepath.Join(fileSystem.Base, path))
	}
	return fullpath
}

func (fileSystem FileSystem) Stat(path string) (info os.FileInfo, notFound bool, err error) {
	info, err = os.Stat(fileSystem.GetFullPath(path))
	if err != nil && os.IsNotExist(err) {
		return nil, true, nil
	}
	return
}

// Get receive file with given path
func (fileSystem FileSystem) Get(path string) (*os.File, error) {
	return os.Open(fileSystem.GetFullPath(path))
}

// Put store a reader into given path
func (fileSystem FileSystem) Put(path string, reader io.Reader) (*oss.Object, error) {
	var (
		fullpath      = fileSystem.GetFullPath(path)
		base          = filepath.Dir(fullpath)
		baseMode, err = path_helpers.ResolvePerms(base)
	)

	if err != nil {
		return nil, errwrap.Wrap(err, "Resolve mode of %q", base)
	}

	if err = os.MkdirAll(base, os.FileMode(baseMode)); err != nil {
		return nil, errwrap.Wrap(err, "Create base directory %q", base)
	}

	dst, err := os.Create(fullpath)

	if err != nil {
		return nil, errwrap.Wrap(err, "Create file %q", fullpath)
	}

	if seeker, ok := reader.(io.ReadSeeker); ok {
		seeker.Seek(0, 0)
	}
	_, err = io.Copy(dst, reader)

	return &oss.Object{Path: path, Name: filepath.Base(path), StorageInterface: fileSystem}, err
}

// Delete delete file
func (fileSystem FileSystem) Delete(path string) error {
	return os.Remove(fileSystem.GetFullPath(path))
}

// List list all objects under current path
func (fileSystem FileSystem) List(path string) ([]*oss.Object, error) {
	var (
		objects  []*oss.Object
		fullpath = fileSystem.GetFullPath(path)
	)

	filepath.Walk(fullpath, func(path string, info os.FileInfo, err error) error {
		if path == fullpath {
			return nil
		}

		if err == nil && !info.IsDir() {
			modTime := info.ModTime()
			objects = append(objects, &oss.Object{
				Path:             strings.TrimPrefix(path, fileSystem.Base),
				Name:             info.Name(),
				LastModified:     &modTime,
				StorageInterface: fileSystem,
			})
		}
		return nil
	})

	return objects, nil
}

// GetEndpoint get Endpoint, FileSystem's Endpoint is /
func (fileSystem FileSystem) GetEndpoint() *oss.Endpoint {
	return &fileSystem.Endpoint
}

func (fileSystem FileSystem) GetURL(p ...string) (url string) {
	url = fileSystem.Endpoint.URL()
	if len(p) > 0 {
		url += "/" + strings.TrimPrefix(strings.Join(p, "/"), "/")
	}
	return
}

func (fileSystem FileSystem) GetDynamicURL(scheme, host string, p ...string) (url string) {
	return fileSystem.GetURL(p...)
}
