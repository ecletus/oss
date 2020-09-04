package filesystem

import (
	"fmt"

	"github.com/mitchellh/go-homedir"
	"github.com/moisespsena-go/assetfs"
	"github.com/pkg/errors"

	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ecletus/helpers"

	"github.com/ecletus/oss/factories"

	errwrap "github.com/moisespsena-go/error-wrap"

	path_helpers "github.com/moisespsena-go/path-helpers"

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

func (this *FileSystem) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pth := this.GetFullPath(r.URL.Path)
	http.ServeFile(w, r, pth)
}

// GetFullPath get full path from absolute/relative path
func (this *FileSystem) GetFullPath(path string) string {
	fullpath := path
	if !strings.HasPrefix(path, this.Base) {
		fullpath, _ = filepath.Abs(filepath.Join(this.Base, path))
	}
	return fullpath
}

func (this *FileSystem) Stat(path string) (info os.FileInfo, notFound bool, err error) {
	info, err = os.Stat(this.GetFullPath(path))
	if err != nil && os.IsNotExist(err) {
		return nil, true, nil
	}
	return
}

// Get receive file with given path
func (this *FileSystem) Get(path string) (*os.File, error) {
	return os.Open(this.GetFullPath(path))
}

// Put store a reader into given path
func (this *FileSystem) Put(path string, reader io.Reader) (*oss.Object, error) {
	var (
		fullpath      = this.GetFullPath(path)
		base          = filepath.Dir(fullpath)
		baseMode, err = path_helpers.ResolveMode(base)
		fileMode      os.FileMode
	)

	if err != nil {
		return nil, errwrap.Wrap(err, "Resolve mode of %q", base)
	}

	if err = os.MkdirAll(base, baseMode); err != nil {
		return nil, errwrap.Wrap(err, "Create base directory %q", base)
	}

	if fileMode, err = path_helpers.ResolveFileMode(fullpath); err != nil {
		return nil, errwrap.Wrap(err, "Resolve mode of %q", fullpath)
	}

	dst, err := os.OpenFile(fullpath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return nil, errwrap.Wrap(err, "Create file %q", fullpath)
	}
	defer dst.Close()

	if seeker, ok := reader.(io.ReadSeeker); ok {
		seeker.Seek(0, 0)
	}
	if _, err = io.Copy(dst, reader); err != nil {
		return nil, err
	}

	return &oss.Object{Path: path, Name: filepath.Base(path), StorageInterface: this}, err
}

// Delete delete file
func (this FileSystem) Delete(path string) error {
	return os.Remove(this.GetFullPath(path))
}

// List list all objects under current path
func (this *FileSystem) List(path string) ([]*oss.Object, error) {
	var (
		objects  []*oss.Object
		fullpath = this.GetFullPath(path)
	)

	filepath.Walk(fullpath, func(path string, info os.FileInfo, err error) error {
		if path == fullpath {
			return nil
		}

		if err == nil && !info.IsDir() {
			modTime := info.ModTime()
			objects = append(objects, &oss.Object{
				Path:             strings.TrimPrefix(path, this.Base),
				Name:             info.Name(),
				LastModified:     &modTime,
				StorageInterface: this,
			})
		}
		return nil
	})

	return objects, nil
}

// GetEndpoint get Endpoint, FileSystem's Endpoint is /
func (this *FileSystem) GetEndpoint() *oss.Endpoint {
	return &this.Endpoint
}

func (this *FileSystem) GetURL(p ...string) (url string) {
	url = this.Endpoint.URL()
	if len(p) > 0 {
		url += "/" + strings.TrimPrefix(strings.Join(p, "/"), "/")
	}
	return
}

func (this *FileSystem) GetDynamicURL(scheme, host string, p ...string) (url string) {
	return this.GetURL(p...)
}

func (this *FileSystem) AssetFS() (_ assetfs.Interface, err error) {
	fs := assetfs.NewAssetFileSystem()
	if err = fs.RegisterPath(filepath.Join(this.Base, "assets")); err != nil {
		err = errors.Wrapf(err, "register path %q", this.Base)
	}
	return fs, nil
}
