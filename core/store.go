package core

import (
	"errors"
	"io"
)

// Storer is a interface for stores
type Storer interface {
	//TODO should return perm or temp failure
	Get(key string) (io.Reader, error)
	Put(key string, reader io.Reader) error
	Del(key string) error
}

// NewStore return a new srore
func NewStore(driver, source string) (Storer, error) {
	switch driver {
	case "disk":
		return NewDiskStore(source)
	default:
		return nil, errors.New("no such driver " + driver + " for store")
	}
}
