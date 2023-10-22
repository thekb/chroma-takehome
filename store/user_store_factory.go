package store

import (
	"context"
	"sync"
	"time"
)

type userStoreFactory struct {
	m      sync.RWMutex
	stores map[int64]UserStore
}

var _ UserStoreFactory = (*userStoreFactory)(nil)

func (f *userStoreFactory) New(ctx context.Context, opts UserStoreOptions) (UserStore, error) {

	getIfExists := func() UserStore {
		f.m.RLock()
		defer f.m.RUnlock()
		if us, ok := f.stores[opts.ID]; ok {
			return us
		}
		return nil
	}

	if us := getIfExists(); us != nil {
		return us, nil
	}

	f.m.Lock()
	defer f.m.Unlock()
	id := time.Now().UnixNano()
	us, err := NewSQLite3Store(opts.DataSource, id)
	if err != nil {
		return nil, err
	}

	f.stores[us.ID()] = us
	return us, nil
}

func NewUserStoreFactory() *userStoreFactory {
	return &userStoreFactory{
		stores: make(map[int64]UserStore),
	}
}
