package glitterboot

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/libs/sync"
)

type store interface {
	Get(key string) (value string, err error)
	Set(key, value string) (err error)
}

var _ store = new(fileStore)

type fileStore struct {
	path string

	m  map[string]string
	mu sync.RWMutex
}

func newFileStore(path string, createIfNotExist bool) (store, error) {
	s := &fileStore{path: path}
	err := s.loadORCreate(createIfNotExist)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *fileStore) loadORCreate(createIfNotExist bool) error {
	data, err := ioutil.ReadFile(s.path)
	if os.IsNotExist(err) && createIfNotExist {
		s.m = map[string]string{}
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.m)
}

func (s *fileStore) Get(key string) (string, error) {
	s.mu.RLock()
	s.mu.RUnlock()
	return s.m[key], nil
}

func (s *fileStore) Set(key, value string) error {
	s.mu.Lock()
	s.mu.Unlock()
	s.m[key] = value
	b, err := json.Marshal(s.m)
	if err != nil {
		return errors.Errorf("Set: failed to marshal: %v", err)
	}
	err = ioutil.WriteFile(s.path, b, 0644)
	if err != nil {
		return errors.Errorf("Set: failed to update file: %v", err)
	}
	return nil
}
