package smap

import "sync"

// Map is like a sync.Map, but utilizes generics for type safety
type Map[K comparable, V any] struct {
	m    map[K]V
	lock sync.RWMutex
}

func Make[K comparable, V any](size int) *Map[K, V] {
	return &Map[K, V]{
		m: make(map[K]V, size),
	}
}

// Len returns the number of items currently stored in the Map
func (sm *Map[K, V]) Len() int {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	l := len(sm.m)
	return l
}

// Set sets the value for a key.
func (sm *Map[K, V]) Set(k K, v V) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	sm.m[k] = v
}

// Get returns the value stored in the map for a key, or nil if no value is present. The ok result indicates whether value was found in the map.
func (sm *Map[K, V]) Get(k K) (v V, ok bool) {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	v, ok = sm.m[k]
	return v, ok
}

// Delete deletes the value for a key.
func (sm *Map[K, V]) Delete(k K) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	delete(sm.m, k)
}

// GetAndDelete deletes the value for a key, returning the previous value if any. The ok result reports whether the key was present.
func (sm *Map[K, V]) GetAndDelete(k K) (v V, ok bool) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	v, ok = sm.m[k]
	delete(sm.m, k)
	return v, ok
}

// Update will set a new value for k if k already exists.
//
//   - If k exists it's value will be updated to v, and the return value will be true.
//   - If k does not exist, no changes will be made to the map, and the return value will be false.
func (sm *Map[K, V]) Update(k K, v V) (ok bool) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	_, ok = sm.m[k]
	if !ok {
		return false
	}
	sm.m[k] = v
	return true
}

// Range executes the provided function for each value in the map.
// If f returns false, it will return without calling for any additional values.
func (sm *Map[K, V]) Range(f func(k K, v V) (cont bool)) {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	for k, v := range sm.m {
		cont := f(k, v)
		if !cont {
			break
		}
	}
}

// RangeAndUpdate is like Range, but f returns a new value for k, which will be used to update the map.
// k is always updated, so f must return the provided old value if no change is desired.
func (sm *Map[K, V]) RangeAndUpdate(f func(k K, oldV V) (newV V, cont bool)) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	for k, oldV := range sm.m {
		newV, cont := f(k, oldV)
		sm.m[k] = newV
		if !cont {
			break
		}
	}
}

// RangeAndDelete is like Range, but f returns an additional bool indicating if k should be removed from the map.
func (sm *Map[K, V]) RangeAndDelete(f func(k K, v V) (del, cont bool)) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	for k, v := range sm.m {
		del, cont := f(k, v)
		if del {
			delete(sm.m, k)
		}
		if !cont {
			break
		}
	}
}

// Keys returns a slice containing all keys in the map
func (sm *Map[K, V]) Keys() []K {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	keys := make([]K, 0, len(sm.m))
	for k := range sm.m {
		keys = append(keys, k)
	}
	return keys
}

// Values returns a slice containing all values in the map
func (sm *Map[K, V]) Values() []V {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	values := make([]V, 0, len(sm.m))
	for _, v := range sm.m {
		values = append(values, v)
	}
	return values
}

// Do is the ultimate escape hatch, operate on the underlying storage in any way required
func (sm *Map[K, V]) Do(f func(m map[K]V) error) error {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	err := f(sm.m)
	return err
}
