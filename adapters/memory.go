package adapters

import (
	"fmt"
	"sync"
	"time"
)

type Value struct {
	V         string
	ExpiresAt time.Time
}

var store sync.Map

type MemoryAdapter struct{}

func (m MemoryAdapter) Get(key string) (bool, string, error) {
	if i, ok := store.Load(key); ok {
		if v, ok := i.(*Value); ok {
			exp := v.ExpiresAt
			until := time.Until(exp)
			if until.Seconds() <= 0 {
				// is expired
				fmt.Println("expired")
				return false, "", nil
			}
			fmt.Println("not expired")
			return true, v.V, nil
		}
	}
	fmt.Println("not existing")
	return false, "", nil
}

func (m MemoryAdapter) Set(key string, content string, TTL int) error {
	expiresAt := time.Now().Add(time.Second * time.Duration(TTL))
	V := Value{
		V:         content,
		ExpiresAt: expiresAt,
	}
	store.Store(key, &V)
	return nil
}

func (m MemoryAdapter) Clear(key string) error {
	return nil
}
