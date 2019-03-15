package oss

import (
	"io"
	"net/http"
	"os"
	"time"
)

type Endpoint struct {
	Scheme string
	Host   string
	Path   string
}

func (ep *Endpoint) URL() string {
	return ep.DinamicURL("", "")
}

func (ep *Endpoint) DinamicURL(scheme, host string) string {
	var url string
	if scheme == "" {
		url = ep.Scheme
	} else {
		url = scheme
	}

	if url != "" {
		url += "://"
	}

	if host == "" {
		url += ep.Host
	} else {
		url += host
	}
	if ep.Path != "" {
		url += ep.Path
	}
	return url
}

// StorageInterface define common API to operate storage
type StorageInterface interface {
	http.Handler
	Stat(path string) (info os.FileInfo, notFound bool, err error)
	Get(path string) (*os.File, error)
	Put(path string, reader io.Reader) (*Object, error)
	Delete(path string) error
	List(path string) ([]*Object, error)
	GetEndpoint() *Endpoint
	GetURL(p ...string) string
	GetDynamicURL(scheme, host string, p ...string) (url string)
}

type NamedStorageInterface interface {
	StorageInterface
	Name() string
}

type NamedStorage struct {
	StorageInterface
	StorageName string
}

func (ns *NamedStorage) Name() string {
	return ns.StorageName
}

// Object content object
type Object struct {
	Path             string
	Name             string
	LastModified     *time.Time
	StorageInterface StorageInterface
}

// Get retrieve object's content
func (object Object) Get() (*os.File, error) {
	return object.StorageInterface.Get(object.Path)
}
