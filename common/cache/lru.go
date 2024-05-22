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
	"container/list"
	"sync"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"

	"go.temporal.io/server/common/clock"
	"go.temporal.io/server/common/metrics"
)

var (
	// ErrCacheFull is returned if Put fails due to cache being filled with pinned elements
	ErrCacheFull = &serviceerror.ResourceExhausted{
		Cause:   enumspb.RESOURCE_EXHAUSTED_CAUSE_SYSTEM_OVERLOADED,
		Scope:   enumspb.RESOURCE_EXHAUSTED_SCOPE_SYSTEM,
		Message: "cache capacity is fully occupied with pinned elements",
	}
	// ErrCacheItemTooLarge is returned if Put fails due to item size being larger than max cache capacity
	ErrCacheItemTooLarge = serviceerror.NewInternal("cache item size is larger than max cache capacity")
)

const emptyEntrySize = 0

// lru is a concurrent fixed size cache that evicts elements in lru order
type (
	lru[K Key, V Value] struct {
		mut            sync.Mutex
		byAccess       *list.List
		byKey          map[K]*list.Element
		maxSize        int
		currSize       int
		pinnedSize     int
		ttl            time.Duration
		pin            bool
		timeSource     clock.TimeSource
		metricsHandler metrics.Handler
	}

	iteratorImpl[K Key, V Value] struct {
		lru        *lru[K, V]
		createTime time.Time
		nextItem   *list.Element
	}

	entryImpl[K Key, V Value] struct {
		key        K
		createTime time.Time
		value      *V
		refCount   int
		size       int
	}
)

// Close closes the iterator
func (it *iteratorImpl[K, V]) Close() {
	it.lru.mut.Unlock()
}

// HasNext return true if there is more items to be returned
func (it *iteratorImpl[K, V]) HasNext() bool {
	return it.nextItem != nil
}

// Next return the next item
func (it *iteratorImpl[K, V]) Next() Entry {
	if it.nextItem == nil {
		panic("LRU cache iterator Next called when there is no next item")
	}

	entry := it.nextItem.Value.(*entryImpl[K, V])
	it.nextItem = it.nextItem.Next()
	// make a copy of the entry so there will be no concurrent access to this entry
	entry = &entryImpl[K, V]{
		key:        entry.key,
		value:      entry.value,
		size:       entry.size,
		createTime: entry.createTime,
	}
	it.prepareNext()
	return entry
}

func (it *iteratorImpl[K, V]) prepareNext() {
	for it.nextItem != nil {
		entry := it.nextItem.Value.(*entryImpl[K, V])
		if it.lru.isEntryExpired(entry, it.createTime) {
			nextItem := it.nextItem.Next()
			it.lru.deleteInternal(it.nextItem)
			it.nextItem = nextItem
		} else {
			return
		}
	}
}

// Iterator returns an iterator to the map. This map
// does not use re-entrant locks, so access or modification
// to the map during iteration can cause a dead lock.
func (c *lru[K, V]) Iterator() Iterator {
	c.mut.Lock()
	iterator := &iteratorImpl[K, V]{
		lru:        c,
		createTime: c.timeSource.Now().UTC(),
		nextItem:   c.byAccess.Front(),
	}
	iterator.prepareNext()
	return iterator
}

func (entry *entryImpl[K, V]) Key() interface{} {
	return entry.key
}

func (entry *entryImpl[K, V]) Value() interface{} {
	return entry.value
}

func (entry *entryImpl[K, V]) Size() int {
	return entry.size
}

func (entry *entryImpl[K, V]) CreateTime() time.Time {
	return entry.createTime
}

// New creates a new cache with the given options
func New[K Key, V Value](maxSize int, opts *Options, handler metrics.Handler) Cache[K, V] {
	if opts == nil {
		opts = &Options{}
	}
	timeSource := opts.TimeSource
	if timeSource == nil {
		timeSource = clock.NewRealTimeSource()
	}

	metrics.CacheSize.With(handler).Record(float64(maxSize))
	metrics.CacheTtl.With(handler).Record(opts.TTL)
	return &lru[K, V]{
		byAccess:       list.New(),
		byKey:          make(map[K]*list.Element),
		ttl:            opts.TTL,
		maxSize:        maxSize,
		currSize:       0,
		pin:            opts.Pin,
		timeSource:     timeSource,
		metricsHandler: handler,
	}
}

// NewLRU creates a new LRU cache of the given size, setting initial capacity
// to the max size
func NewLRU[K Key, V Value](maxSize int, handler metrics.Handler) Cache[K, V] {
	return New[K, V](maxSize, nil, handler)
}

// Get retrieves the value stored under the given key
func (c *lru[K, V]) Get(key K) *V {
	if c.maxSize == 0 { //
		return nil
	}
	c.mut.Lock()
	defer c.mut.Unlock()

	element := c.byKey[key]
	if element == nil {
		return nil
	}

	entry := element.Value.(*entryImpl[K, V])

	metrics.CacheEntryAgeOnGet.With(c.metricsHandler).Record(c.timeSource.Now().UTC().Sub(entry.createTime))

	if c.isEntryExpired(entry, c.timeSource.Now().UTC()) {
		// Entry has expired
		c.deleteInternal(element)
		return nil
	}

	c.updateEntryRefCount(entry)
	c.byAccess.MoveToFront(element)
	return entry.value
}

// Put puts a new value associated with a given key, returning the existing value (if present)
func (c *lru[K, V]) Put(key K, value V) V {
	if c.pin {
		panic("Cannot use Put API in Pin mode. Use Delete and PutIfNotExist if necessary")
	}
	val, _ := c.putInternal(key, value, true)
	return val
}

// PutIfNotExist puts a value associated with a given key if it does not exist
func (c *lru[K, V]) PutIfNotExist(key K, value V) (*V, error) {
	existing, err := c.putInternal(key, value, false)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		// This is a new value
		return &value, err
	}

	return existing, err
}

// Delete deletes a key, value pair associated with a key
func (c *lru[K, V]) Delete(key K) {
	if c.maxSize == 0 {
		return
	}
	c.mut.Lock()
	defer c.mut.Unlock()

	element := c.byKey[key]
	if element != nil {
		c.deleteInternal(element)
	}
}

// Release decrements the ref count of a pinned element.
func (c *lru[K, V]) Release(key K) {
	if c.maxSize == 0 || !c.pin {
		return
	}
	c.mut.Lock()
	defer c.mut.Unlock()

	elt, ok := c.byKey[key]
	if !ok {
		return
	}
	entry := elt.Value.(*entryImpl[K, V])
	entry.refCount--
	if entry.refCount == 0 {
		c.pinnedSize -= entry.Size()
		metrics.CachePinnedUsage.With(c.metricsHandler).Record(float64(c.pinnedSize))
	}
	// Entry size might have changed. Recalculate size and evict entries if necessary.
	newEntrySize := getSize(entry.value)
	c.currSize = c.calculateNewCacheSize(newEntrySize, entry.Size())
	entry.size = newEntrySize
	if c.currSize > c.maxSize {
		c.tryEvictUntilCacheSizeUnderLimit()
	}
	metrics.CacheUsage.With(c.metricsHandler).Record(float64(c.currSize))
}

// Size returns the current size of the lru, useful if cache is not full. This size is calculated by summing
// the size of all entries in the cache. And the entry size is calculated by the size of the value.
// The size of the value is calculated implementing the Sizeable interface. If the value does not implement
// the Sizeable interface, the size is 1.
func (c *lru[K, V]) Size() int {
	c.mut.Lock()
	defer c.mut.Unlock()

	return c.currSize
}

// Put puts a new value associated with a given key, returning the existing value (if present)
// allowUpdate flag is used to control overwrite behavior if the value exists.
func (c *lru[K, V]) putInternal(key K, value V, allowUpdate bool) (*V, error) {
	if c.maxSize == 0 {
		return nil, nil
	}
	newEntrySize := getSize(value)
	if newEntrySize > c.maxSize {
		return nil, ErrCacheItemTooLarge
	}

	c.mut.Lock()
	defer c.mut.Unlock()

	elt := c.byKey[key]
	// If the entry exists, check if it has expired or update the value
	if elt != nil {
		existingEntry := elt.Value.(*entryImpl[K, V])
		if !c.isEntryExpired(existingEntry, time.Now().UTC()) {
			existingVal := existingEntry.value
			if allowUpdate {
				newCacheSize := c.calculateNewCacheSize(newEntrySize, existingEntry.Size())
				if newCacheSize > c.maxSize {
					c.tryEvictUntilEnoughSpaceWithSkipEntry(newEntrySize, existingEntry)
					// calculate again after eviction
					newCacheSize = c.calculateNewCacheSize(newEntrySize, existingEntry.Size())
					if newCacheSize > c.maxSize {
						// This should never happen since allowUpdate is always **true** for non-pinned cache,
						// and if all entries are not pinned(ref==0), then the cache should never be full as long as
						// new entry's size is less than max size.
						// However, to prevent any unexpected behavior, it checks the cache size again.
						return nil, ErrCacheFull
					}
				}
				existingEntry.value = &value
				existingEntry.size = newEntrySize
				c.currSize = newCacheSize
				metrics.CacheUsage.With(c.metricsHandler).Record(float64(c.currSize))
				c.updateEntryTTL(existingEntry)
			}

			c.updateEntryRefCount(existingEntry)
			c.byAccess.MoveToFront(elt)
			return existingVal, nil
		}

		// Entry has expired
		c.deleteInternal(elt)
	}

	c.tryEvictUntilEnoughSpaceWithSkipEntry(newEntrySize, nil)

	// check if the new entry can fit in the cache
	newCacheSize := c.calculateNewCacheSize(newEntrySize, emptyEntrySize)
	if newCacheSize > c.maxSize {
		return nil, ErrCacheFull
	}

	entry := &entryImpl[K, V]{
		key:   key,
		value: value,
		size:  newEntrySize,
	}

	c.updateEntryTTL(entry)
	c.updateEntryRefCount(entry)
	element := c.byAccess.PushFront(entry)
	c.byKey[key] = element
	c.currSize = newCacheSize
	metrics.CacheUsage.With(c.metricsHandler).Record(float64(c.currSize))
	return nil, nil
}

func (c *lru[K, V]) calculateNewCacheSize(newEntrySize int, existingEntrySize int) int {
	return c.currSize - existingEntrySize + newEntrySize
}

func (c *lru[K, V]) deleteInternal(element *list.Element) {
	entry := c.byAccess.Remove(element).(*entryImpl[K, V])
	c.currSize -= entry.Size()
	metrics.CacheUsage.With(c.metricsHandler).Record(float64(c.currSize))
	metrics.CacheEntryAgeOnEviction.With(c.metricsHandler).Record(c.timeSource.Now().UTC().Sub(entry.createTime))
	delete(c.byKey, entry.key)
}

// tryEvictUntilCacheSizeUnderLimit tries to evict entries until c.currSize is less than c.maxSize.
func (c *lru[K, V]) tryEvictUntilCacheSizeUnderLimit() {
	c.tryEvictUntilEnoughSpaceWithSkipEntry(0, nil)
}

// tryEvictUntilEnoughSpaceWithSkipEntry try to evict entries until there is enough space for the new entry without
// evicting the existing entry. the existing entry is skipped because it is being updated.
func (c *lru[K, V]) tryEvictUntilEnoughSpaceWithSkipEntry(newEntrySize int, existingEntry *entryImpl[K, V]) {
	element := c.byAccess.Back()
	existingEntrySize := 0
	if existingEntry != nil {
		existingEntrySize = existingEntry.Size()
	}

	for c.calculateNewCacheSize(newEntrySize, existingEntrySize) > c.maxSize && element != nil {
		entry := element.Value.(*entryImpl[K, V])
		if existingEntry != nil && entry.key == existingEntry.key {
			element = element.Prev()
			continue
		}
		element = c.tryEvictAndGetPreviousElement(entry, element)
	}
}

func (c *lru[K, V]) tryEvictAndGetPreviousElement(entry *entryImpl[K, V], element *list.Element) *list.Element {
	if entry.refCount == 0 {
		elementPrev := element.Prev()
		// currSize will be updated within deleteInternal
		c.deleteInternal(element)
		return elementPrev
	}
	// entry.refCount > 0
	// skip, entry still being referenced
	return element.Prev()
}

func (c *lru[K, V]) isEntryExpired(entry *entryImpl[K, V], currentTime time.Time) bool {
	return entry.refCount == 0 && !entry.createTime.IsZero() && currentTime.After(entry.createTime.Add(c.ttl))
}

func (c *lru[K, V]) updateEntryTTL(entry *entryImpl[K, V]) {
	if c.ttl != 0 {
		entry.createTime = c.timeSource.Now().UTC()
	}
}

func (c *lru[K, V]) updateEntryRefCount(entry *entryImpl[K, V]) {
	if c.pin {
		entry.refCount++
		if entry.refCount == 1 {
			c.pinnedSize += entry.Size()
			metrics.CachePinnedUsage.With(c.metricsHandler).Record(float64(c.pinnedSize))
		}
	}
}
