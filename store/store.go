package store

// put: err := storage.disk.put(key, io.reader).Error
// get: io.reader, err := storage.disk.get(id)

import (
	"errors"
	"io"
)

// storer is a interface for stores
type Storer interface {
	Get(key string) (io.Writer, error)
	Put(key string, reader io.Reader) error
	Del(key string) error
}

// New return a new srore
func New(driver, source string) (Storer, error) {
	switch driver {
	case "disk":
		return NewDiskStore(source)
	default:
		return nil, errors.New("no such driver " + driver + " for store")
	}
}
