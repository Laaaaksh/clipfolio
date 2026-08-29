package api

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// fakeObjectStore is an in-memory ObjectStore for tests, so API and
// transcode-queue tests don't need a real S3-compatible bucket.
type fakeObjectStore struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func newFakeObjectStore() *fakeObjectStore {
	return &fakeObjectStore{objects: make(map[string][]byte)}
}

func (f *fakeObjectStore) Put(_ context.Context, key string, data []byte, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[key] = append([]byte(nil), data...)
	return nil
}

func (f *fakeObjectStore) Get(_ context.Context, key string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.objects[key]
	if !ok {
		return nil, fmt.Errorf("not found: %s", key)
	}
	return data, nil
}

func (f *fakeObjectStore) DeletePrefix(_ context.Context, prefix string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for key := range f.objects {
		if strings.HasPrefix(key, prefix) {
			delete(f.objects, key)
		}
	}
	return nil
}

func (f *fakeObjectStore) PublicURL(key string) string {
	return "https://fake-cdn.test/" + key
}
