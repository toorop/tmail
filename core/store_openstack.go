package core

import (
	"errors"
	"io"

	"github.com/toorop/gopenstack/objectstorage/v1"
)

// DiskStore represents a physical disk store
type openstackStore struct {
	Region    string
	Container string
}

// newOpenstackStore check object storage and return a new openstackStore
func newOpenstackStore() (*openstackStore, error) {
	osPath := objectstorageV1.NewOsPathFromPath(Cfg.GetStoreSource())
	if !osPath.IsContainer() {
		return nil, errors.New("path " + Cfg.GetStoreDriver() + " is not a path to a valid openstack container")
	}
	// container exists ?
	container := &objectstorageV1.Container{
		Region: osPath.Region,
		Name:   osPath.Container,
	}
	err := container.Put(&objectstorageV1.ContainerRequestParameters{
		IfNoneMatch: true,
	})
	if err != nil {
		return nil, err
	}
	store := &openstackStore{
		Region:    osPath.Region,
		Container: osPath.Container,
	}
	return store, nil
}

// Put save key value in store
func (s *openstackStore) Put(key string, reader io.Reader) error {
	if key == "" {
		return errors.New("store.Put: key is empty")
	}
	object := objectstorageV1.Object{
		Name:      key,
		Region:    s.Region,
		Container: s.Container,
		RawData:   reader,
	}
	return object.Put(&objectstorageV1.ObjectRequestParameters{
		IfNoneMatch: true,
	})
}

// Get returns io.Reader corresponding to key
func (s *openstackStore) Get(key string) (io.Reader, error) {
	if key == "" {
		return nil, errors.New("store.Get: key is empty")
	}
	object := objectstorageV1.Object{
		Name:      key,
		Region:    s.Region,
		Container: s.Container,
	}
	err := object.Get(nil)
	if err != nil {
		return nil, err
	}
	return object.RawData, nil
}

// Del
func (s *openstackStore) Del(key string) error {
	return nil
	if key == "" {
		return errors.New("store.Del: key is empty")
	}
	object := objectstorageV1.Object{
		Name:      key,
		Region:    s.Region,
		Container: s.Container,
	}
	return object.Delete(false)
}
