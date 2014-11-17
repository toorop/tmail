package store

// put: err := storage.disk.put(key, io.reader).Error
// get: io.reader, err := storage.disk.get(id)

import (
	"io"
)

type Storer interface {
	Get(key string) (io.Writer, error)
	Put(key string, reader io.Reader) error
	Del(key string) error
}
