package storage

import (
	"time"

	"github.com/patrickmn/go-cache"
)

const (
	defaultExpiration = 1 * time.Minute
	defaultCleanup    = 1 * time.Minute
)

// TypedCache provides type-safe caching as required and leverages in-memory
// storage to realize maximum performance.
type TypedCache[T any] struct {
	c *cache.Cache
}

func NewTypedCache[T any]() *TypedCache[T] {
	return &TypedCache[T]{
		c: cache.New(defaultExpiration, defaultCleanup),
	}
}

func (tc *TypedCache[T]) Get(key string) (T, bool) {
	val, found := tc.c.Get(key)
	if !found {
		var zero T

		return zero, false
	}

	typed, ok := val.(T)
	if !ok {
		tc.Delete(key)

		var zero T

		return zero, false
	}

	return typed, true
}

func (tc *TypedCache[T]) Set(key string, value T, exp time.Duration) {
	tc.c.Set(key, value, exp)
}

func (tc *TypedCache[T]) Delete(key string) {
	tc.c.Delete(key)
}
