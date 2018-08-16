package oss

import (
	"errors"
)

type NameGetter func(name string) string

type Names struct {
	data    map[string]string
	aliases    map[string]string
	getters []NameGetter
}

func (names *Names) Getter(getter NameGetter) *Names {
	names.getters = append(names.getters, getter)
	return names
}

func (names *Names) Alias(real, alias string, aliases ...string) error {
	if _, ok := names.aliases[alias]; ok {
		return errors.New("Alias \"" + alias + "\" has be duplicated.")
	}
	names.aliases[alias] = real
	for _, alias := range aliases {
		if _, ok := names.aliases[alias]; ok {
			return errors.New("Alias \"" + alias + "\" has be duplicated.")
		}
		names.aliases[alias] = real
	}
	return nil
}

// Get used to get storage name with key
func (names *Names) Get(key string) (name string) {
	if name, ok := names.data[key]; ok {
		return name
	} else if name, ok := names.aliases[key]; ok {
		return name
	} else {
		for _, getter := range names.getters {
			name = getter(key)
			if name != "" {
				return name
			}
		}
		return key
	}
}

func (names *Names) GetOrDefault(key string, defauls ...string) (name string) {
	name = names.Get(key)
	if name == key {
		for _, defaul := range defauls {
			name = names.Get(defaul)
			if name != defaul {
				return
			}
		}
	}
	return
}

// Get used to set option with name
func (names *Names) Set(key, value string) *Names {
	names.data[key] = value
	return names
}

func NewNames() *Names {
	return &Names{make(map[string]string), make(map[string]string), []NameGetter{}}
}