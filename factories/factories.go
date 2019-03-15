package factories

import (
	"github.com/aghape/oss"
	"github.com/moisespsena-go/stringvar"
	"github.com/moisespsena/go-options"
)

type Context struct {
	Var *stringvar.StringVar
	Options options.Options
}

func NewContext(data ...map[string]interface{}) *Context {
	return &Context{Options:options.NewOptions(data...)}
}

type StorageFactory interface {
	Factory(ctx *Context, config map[string]interface{}) (storage oss.StorageInterface, err error)
}

type StorageFactoryFunc func(ctx *Context, config map[string]interface{}) (storage oss.StorageInterface, err error)

func (f StorageFactoryFunc) Factory(ctx *Context, config map[string]interface{}) (storage oss.StorageInterface, err error) {
	return f(ctx, config)
}

type StorageFactoriesRegister struct {
	factories map[string]StorageFactory
}

func NewStorageFactoriesRegister() *StorageFactoriesRegister {
	return &StorageFactoriesRegister{map[string]StorageFactory{}}
}

func (r *StorageFactoriesRegister) Registry(name string, factory StorageFactory) {
	r.factories[name] = factory
}

func (r *StorageFactoriesRegister) Get(name string) (factory StorageFactory, ok bool) {
	factory, ok = r.factories[name]
	return
}

var FactoriesRegister = NewStorageFactoriesRegister()

func Registry(name string, factory StorageFactory) {
	FactoriesRegister.Registry(name, factory)
}

func Get(name string) (factory StorageFactory, ok bool) {
	return FactoriesRegister.Get(name)
}
