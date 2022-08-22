package twoQueues

import (
	"container/list"
	"hash/maphash"
	"sync"
)

type Cache[K ~string, V any] struct {
	seed      maphash.Seed
	shardMask uint64
	shard     []*cacheShard[K, V]
}

type cacheShard[K ~string, V any] struct {
	history *lruCache[K, V]
	cache   *lruCache[K, V]

	maxShardSize   int
	maxHistorySize int

	mu sync.Mutex
}

type lruCache[K ~string, V any] struct {
	mp map[K]*list.Element
	l  *list.List
}

type entry[K ~string, V any] struct {
	key   K
	value V
}

func New[K ~string, V any](maxCacheSize int, shards uint64) *Cache[K, V] {

	if shards != 1 && shards&1 != 0 {
		panic("shards must be a power of 2")
	}

	ca := &Cache[K, V]{
		seed:      maphash.MakeSeed(),
		shardMask: shards - 1,
		shard:     make([]*cacheShard[K, V], shards),
	}

	for i := 0; i < int(shards); i++ {
		ca.shard[i] = &cacheShard[K, V]{
			history: &lruCache[K, V]{
				mp: make(map[K]*list.Element),
				l:  list.New(),
			},
			cache: &lruCache[K, V]{
				mp: make(map[K]*list.Element),
				l:  list.New(),
			},
			maxShardSize:   maxCacheSize / int(shards),
			maxHistorySize: maxCacheSize / int(shards) * 10,
		}
	}

	return ca
}

func (ca *cacheShard[K, V]) get(key K) (V, bool) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	if e, ok := ca.cache.mp[key]; ok {
		ca.cache.l.MoveToFront(e)
		return e.Value.(*entry[K, V]).value, true
	} else {

		if e, ok := ca.history.mp[key]; ok {
			ca.cache.mp[key] = ca.cache.l.PushFront(e.Value)
			delete(ca.history.mp, key)
			ca.history.l.Remove(e)
			if ca.cache.l.Len() > ca.maxShardSize {
				elm := ca.cache.l.Back()
				delete(ca.cache.mp, elm.Value.(*entry[K, V]).key)
				ca.cache.l.Remove(elm)
			}
			return e.Value.(*entry[K, V]).value, true
		}
	}
	return *new(V), false
}

func (ca *cacheShard[K, V]) set(key K, value V) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	// key 存在于 history 里
	if e, ok := ca.history.mp[key]; ok {
		e.Value.(*entry[K, V]).value = value
	} else {
		// key 存在于 cache 里
		if e, ok := ca.cache.mp[key]; ok {
			e.Value.(*entry[K, V]).value = value
			ca.history.mp[key] = ca.history.l.PushFront(e.Value)
			delete(ca.cache.mp, key)
			ca.cache.l.Remove(e)
		} else {
			ca.history.mp[key] = ca.history.l.PushFront(&entry[K, V]{key, value})
		}
		if ca.history.l.Len() > ca.maxHistorySize {
			elm := ca.history.l.Back()
			delete(ca.history.mp, elm.Value.(*entry[K, V]).key)
			ca.history.l.Remove(elm)
		}
	}
}

func (ca *Cache[K, V]) Get(k K) (V, bool) {
	if ca.shardMask == 0 {
		return ca.shard[0].get(k)
	}
	return ca.shard[maphash.String(ca.seed, string(k))&ca.shardMask].get(k)
}

func (ca *Cache[K, V]) Set(key K, value V) {
	if ca.shardMask == 0 {
		ca.shard[0].set(key, value)
	} else {
		ca.shard[maphash.String(ca.seed, string(key))&ca.shardMask].set(key, value)
	}
}
