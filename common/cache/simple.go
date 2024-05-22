// The MIT License (MIT)
//
// Copyright (c) 2017-2020 Uber Technologies Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package cache

import (
	"container/list"
	"sync"
	"time"
)

var (
	// DummyCreateTime is the create time used by all entries in the cache.
	DummyCreateTime = time.Time{}
)

type (
	simple[K Key, V Value] struct {
		sync.RWMutex
		accessMap   map[K]*list.Element
		iterateList *list.List
		rmFunc      RemovedFunc
	}

	simpleItr[K Key, V Value] struct {
		simple   *simple[K, V]
		nextItem *list.Element
	}

	simpleEntry[K Key, V Value] struct {
		key   K
		value V
	}
)

// Close closes the iterator
func (it *simpleItr[K, V]) Close() {
	it.simple.RUnlock()
}

// HasNext return true if there is more items to be returned
func (it *simpleItr[K, V]) HasNext() bool {
	return it.nextItem != nil
}

// Next returns the next item
func (it *simpleItr[K, V]) Next() Entry {
	if it.nextItem == nil {
		panic("Simple cache iterator Next called when there is no next item")
	}

	entry := it.nextItem.Value.(*simpleEntry[K, V])
	it.nextItem = it.nextItem.Next()
	// make a copy of the entry so there will be no concurrent access to this entry
	entry = &simpleEntry[K, V]{
		key:   entry.key,
		value: entry.value,
	}
	return entry
}

func (e *simpleEntry[K, V]) Key() interface{} {
	return e.key
}

func (e *simpleEntry[K, V]) Value() interface{} {
	return e.value
}

// CreateTime is not implemented for simple cache entries
func (e *simpleEntry[K, V]) CreateTime() time.Time {
	return DummyCreateTime
}

// NewSimple creates a new simple cache with given options.
// Simple cache will never evict entries and it will never reorder the elements.
// Simple cache also does not have the concept of pinning that LRU cache has.
// Internally simple cache uses a RWMutex instead of the exclusive Mutex that LRU cache uses.
// The RWMutex makes simple cache readable by many threads without introducing lock contention.
func NewSimple[K Key, V Value](opts *SimpleOptions) Cache[K, V] {
	if opts == nil {
		opts = &SimpleOptions{}
	}
	return &simple[K, V]{
		iterateList: list.New(),
		accessMap:   make(map[K]*list.Element),
		rmFunc:      opts.RemovedFunc,
	}
}

// Get retrieves the value stored under the given key
func (c *simple[K, V]) Get(key K) interface{} {
	c.RLock()
	defer c.RUnlock()

	element := c.accessMap[key]
	if element == nil {
		return nil
	}
	return element.Value.(*simpleEntry[K, V]).Value()
}

// Put puts a new value associated with a given key, returning the existing value (if present).
func (c *simple[K, V]) Put(key K, value V) interface{} {
	c.Lock()
	defer c.Unlock()
	existing := c.putInternal(key, value, true)
	return existing
}

// PutIfNotExist puts a value associated with a given key if it does not exist
func (c *simple[K, V]) PutIfNotExist(key K, value V) (interface{}, error) {
	c.Lock()
	defer c.Unlock()
	existing := c.putInternal(key, value, false)
	if existing == nil {
		// This is a new value
		return value, nil
	}
	return existing, nil
}

// Delete deletes a key, value pair associated with a key
func (c *simple[K, V]) Delete(key K) {
	c.Lock()
	defer c.Unlock()

	element := c.accessMap[key]
	if element == nil {
		return
	}
	entry := c.iterateList.Remove(element).(*simpleEntry[K, V])
	if c.rmFunc != nil {
		go c.rmFunc(entry.value)
	}
	delete(c.accessMap, entry.key)
}

// Release does nothing for simple cache
func (c *simple[K, V]) Release(_ K) {}

// Size returns the number of entries currently in the cache
func (c *simple[K, V]) Size() int {
	c.RLock()
	defer c.RUnlock()

	return len(c.accessMap)
}

func (c *simple[K, V]) Iterator() Iterator {
	c.RLock()
	iterator := &simpleItr[K, V]{
		simple:   c,
		nextItem: c.iterateList.Front(),
	}
	return iterator
}

func (c *simple[K, V]) putInternal(key K, value V, allowUpdate bool) interface{} {
	elt := c.accessMap[key]
	if elt != nil {
		entry := elt.Value.(*simpleEntry[K, V])
		existing := entry.value
		if allowUpdate {
			entry.value = value
		}
		return existing
	}
	entry := &simpleEntry[K, V]{
		key:   key,
		value: value,
	}
	c.accessMap[key] = c.iterateList.PushFront(entry)
	return nil
}
