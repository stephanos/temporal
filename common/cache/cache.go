// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cache

import (
	"time"

	"go.temporal.io/server/common/clock"
)

type (
	Key   comparable
	Value any

	// A Cache is a generalized interface to a cache.  See cache.LRU for a specific
	// implementation (bounded cache with LRU eviction)
	Cache[K Key, V Value] interface {
		// Get retrieves an element based on a key, returning nil if the element
		// does not exist
		Get(key K) *V

		// Put adds an element to the cache, returning the previous element
		Put(key K, value V) V

		// PutIfNotExist puts a value associated with a given key if it does not exist
		PutIfNotExist(key K, value V) (*V, error)

		// Delete deletes an element in the cache
		Delete(key K)

		// Release decrements the ref count of a pinned element. If the ref count
		// drops to 0, the element can be evicted from the cache.
		Release(key K)

		// Iterator returns the iterator of the cache
		Iterator() Iterator

		// Size returns current size of the Cache, the size definition is implementation of SizeGetter interface
		// for the entry size, if the entry does not implement SizeGetter interface, the size is 1
		Size() int
	}

	// Options control the behavior of the cache
	Options struct {
		// TTL controls the time-to-live for a given cache entry.  Cache entries that
		// are older than the TTL will not be returned.
		TTL time.Duration

		// Pin prevents in-use objects from getting evicted.
		Pin bool

		// TimeSource is an optional clock to use for time-skipping and testing. If this is nil, a real clock will be used.
		TimeSource clock.TimeSource
	}

	// SimpleOptions provides options that can be used to configure SimpleCache
	SimpleOptions struct {
		// RemovedFunc is an optional function called when an element
		// is scheduled for deletion
		RemovedFunc RemovedFunc
	}

	// RemovedFunc is a type for notifying applications when an item is
	// scheduled for removal from the Cache. If f is a function with the
	// appropriate signature and i is the interface{} scheduled for
	// deletion, Cache calls go f(i)
	RemovedFunc func(interface{})

	// Iterator represents the interface for cache iterators
	Iterator interface {
		// Close closes the iterator
		// and releases any allocated resources
		Close()
		// HasNext return true if there is more items to be returned
		HasNext() bool
		// Next return the next item
		Next() Entry
	}

	// Entry represents a key-value entry within the map
	Entry interface {
		// Key represents the key
		Key() interface{}
		// Value represents the value
		Value() interface{}
		// CreateTime represents the time when the entry is created
		CreateTime() time.Time
	}
)
