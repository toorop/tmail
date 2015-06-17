package core

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"
)

// DiskStore represents a physical disk store
type diskStore struct {
	basePath string
}

// NewDiskStore returns a store with local disk as backend
func NewDiskStore(basePath string) (*diskStore, error) {
	basePath = path.Clean(basePath)
	// check if path exists & is writable
	fi, err := os.Stat(basePath)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, errors.New(basePath + " is not a directory.")
	}
	f, err := os.OpenFile(path.Join(basePath, "testingIfIsWritable"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	os.Remove(path.Join(basePath, "testingIfIsWritable"))
	return &diskStore{basePath}, nil
}

// Get returns io.Reader corresponding to key
func (s *diskStore) Get(key string) (io.Reader, error) {
	if key == "" {
		return nil, errors.New("diskStore.Get: key is empty")
	}
	spath := s.getStoragePath(key)
	raw, err := ioutil.ReadFile(spath)
	if err != nil {
		return nil, errors.New("diskStore.Get: unable to open " + spath + " for reading." + err.Error())
	}
	return io.Reader(bytes.NewReader(raw)), nil
}

// Put save key value in store
func (s *diskStore) Put(key string, reader io.Reader) error {
	var err error
	if key == "" {
		return errors.New("diskStore.Put: key is empty")
	}
	spath := s.getStoragePath(key)

	// Is file exist (should never happen but...)
	_, err = os.Stat(spath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	// create path
	if err = os.MkdirAll(path.Dir(spath), 0766); err != nil {
		return err
	}
	f, err := os.OpenFile(spath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0700)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, reader)
	return err
}

// Del
func (s *diskStore) Del(key string) error {
	if key == "" {
		return errors.New("diskStore.Put: key is empty")
	}
	return os.Remove(s.getStoragePath(key))
}

// getStoragePath returns storage path associated with key key
func (s *diskStore) getStoragePath(key string) string {
	lenKey := len(key)
	if lenKey == 1 {
		return path.Join(s.basePath, key)
	}
	sPath := s.basePath
	for i := 0; i < lenKey-1; i++ {
		sPath = path.Join(sPath, key[i:i+1])
		if i == 3 {
			break
		}
	}
	return path.Join(sPath, key)
}
