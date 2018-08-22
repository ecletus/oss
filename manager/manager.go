package oss

import (
	"github.com/aghape/oss"
	"github.com/aghape/oss/filesystem"
	"github.com/aghape/core"
)

var Storages = &StoragesManager{nil,
	nil,
	make(map[string]oss.StorageInterface),
	&StorageNamesResolver{}}

type StoragesManager struct {
	Default       oss.StorageInterface
	DefaultFS     *filesystem.FileSystem
	storages      map[string]oss.StorageInterface
	NameResolvers *StorageNamesResolver
}

func (s *StoragesManager) Register(name string, storage oss.StorageInterface) {
	s.storages[name] = storage
}

func (s *StoragesManager) Get(name string) oss.StorageInterface {
	if name == "default" {
		return s.Default
	}
	if name == "default_fs" {
		return s.DefaultFS
	}
	return s.storages[name]
}

func (s *StoragesManager) GetOrDefault(name string) oss.StorageInterface {
	storage := s.Get(name)
	if storage == nil {
		storage = s.Default
	}
	return storage
}

func (s *StoragesManager) ResolveName(context *core.Context, name string) oss.StorageInterface {
	name = s.NameResolvers.Discovery(context, name)
	return s.Get(name)
}

func (s *StoragesManager) ResolveNameOrDefault(context *core.Context, name string) oss.StorageInterface {
	name = s.NameResolvers.Discovery(context, name)
	return s.GetOrDefault(name)
}

type StorageNameResolverHandler func(name *StorageNameDiscover)

type StorageNameResolver struct {
	name    string
	handler StorageNameResolverHandler
	prev    *StorageNameResolver
	next    *StorageNameResolver
}

type StorageNamesResolver struct {
	FirstResolver *StorageNameResolver
	LastResolver  *StorageNameResolver
}

type StorageNameDiscover struct {
	Name            string
	Context         *core.Context
	CurrentResolver *StorageNameResolver
	Resolver        *StorageNameResolver
	Parent          *StorageNameDiscover
}

func (nd *StorageNameDiscover) SetName(name string) {
	parent := *nd
	nd.Parent = &parent
	nd.Name = name
	nd.Resolver = nd.CurrentResolver
}

func (nd *StorageNameDiscover) Next() {
	if nd.CurrentResolver.next != nil {
		nd.CurrentResolver.next.handler(nd)
	}
}

func (r *StorageNamesResolver) Discovery(context *core.Context, name string) string {
	snd := &StorageNameDiscover{name, context, r.FirstResolver, nil, nil}
	snd.Next()
	return snd.Name
}

func (r *StorageNamesResolver) RegisterResolver(name string, handler StorageNameResolverHandler) (resolver *StorageNameResolver) {
	resolver = &StorageNameResolver{name: name, handler: handler, prev: r.LastResolver}
	if r.LastResolver == nil {
		r.FirstResolver = &StorageNameResolver{name: "", handler: nil, prev: nil, next:resolver}
		r.LastResolver = resolver
	} else {
		r.LastResolver.next = resolver
	}
	r.LastResolver = resolver
	return
}
