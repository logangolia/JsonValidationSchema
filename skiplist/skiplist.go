package skiplist

import (
	"context"
	"math/rand"
	"sync"
	"cmp"
)

// Constants for the maximum number of levels in the skip list and the probability for random level generation.
const (
	maxLevel    = 4
	probability = 0.5
)

// UpdateCheck is a function type that, given a key and its current value, returns the new value to be set.
// If the key does not exist, exists will be false and currValue will be the zero value of V.
type UpdateCheck[K cmp.Ordered, V any] func(key K, currValue V, exists bool) (newValue V, err error)

// Pair represents a key-value pair in the skip list.
type Pair[K cmp.Ordered, V any] struct {
	Key   K
	Value V
}

// SkipList is an interface that defines the methods a skip list should implement.
type SkipList[K cmp.Ordered, V any] interface {
	Upsert(key K, check UpdateCheck[K, V]) (updated bool, err error)
	Remove(key K) (removedValue V, removed bool)
	Find(key K) (foundValue V, found bool)
	Query(ctx context.Context, start K, end K) (results []Pair[K, V], err error)
}

// node represents an individual node in the skip list.
type node[K cmp.Ordered, V any] struct {
	key     K
	value   V
	forward []*node[K, V]  // Pointers to the next nodes at each level.
}

// SkipListImpl is the concrete implementation of the SkipList interface.
type SkipListImpl[K cmp.Ordered, V any] struct {
	header *node[K, V]     // Starting point of the skip list.
	level  int             // Current number of levels in the skip list.
	mu     sync.RWMutex 
}

// NewSkipList initializes and returns a new SkipListImpl.
func NewSkipList[K cmp.Ordered, V any]() *SkipListImpl[K, V] {
	return &SkipListImpl[K, V]{
		header: &node[K, V]{
			forward: make([]*node[K, V], maxLevel),
		},
	}
}

// randomLevel generates a random level for a new node.
func (sl *SkipListImpl[K cmp.Ordered, V any]) randomLevel() int {
	lvl := 1
	for rand.Float64() < probability && lvl < maxLevel {
		lvl++
	}
	return lvl
}

// Upsert inserts or updates a key-value pair in the skip list based on the provided check function.
func (sl *SkipListImpl[K cmp.Ordered, V any]) Upsert(key K, check UpdateCheck[K, V]) (bool, error) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	update := make([]*node[K, V], maxLevel)
	current := sl.header

	// Traverse the list to find the insert/update position.
	for i := sl.level - 1; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].key < key {
			current = current.forward[i]
		}
		update[i] = current
	}

	// If the key exists, update its value.
	current = current.forward[0]
	if current != nil && current.key == key {
		newValue, err := check(key, current.value, true)
		if err != nil {
			return false, err
		}
		current.value = newValue
		return true, nil
	}

	// If the key doesn't exist, insert it.
	level := sl.randomLevel()
	if level > sl.level {
		for i := sl.level; i < level; i++ {
			update[i] = sl.header
		}
		sl.level = level
	}

	newValue, err := check(key, V{}, false)
	if err != nil {
		return false, err
	}

	newNode := &node[K, V]{
		key:     key,
		value:   newValue,
		forward: make([]*node[K, V], level),
	}

	for i := 0; i < level; i++ {
		newNode.forward[i] = update[i].forward[i]
		update[i].forward[i] = newNode
	}

	return true, nil
}

// Remove deletes a key from the skip list and returns the removed value.
func (sl *SkipListImpl[K cmp.Ordered, V any]) Remove(key K) (V, bool) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	update := make([]*node[K, V], maxLevel)
	current := sl.header

	// Traverse the list to find the node to be removed.
	for i := sl.level - 1; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].key < key {
			current = current.forward[i]
		}
		update[i] = current
	}

	// If the key exists, remove it.
	current = current.forward[0]
	if current != nil && current.key == key {
		for i := 0; i < sl.level && update[i].forward[i] == current; i++ {
			update[i].forward[i] = current.forward[i]
		}

		for sl.level > 1 && sl.header.forward[sl.level-1] == nil {
			sl.level--
		}
		return current.value, true
	}
	return V{}, false
}

// Find retrieves a value from the skip list by key.
// If the key exists, it returns the associated value and true.
// Otherwise, it returns the zero value of V and false.
func (sl *SkipListImpl[K cmp.Ordered, V any]) Find(key K) (V, bool) {
	sl.mu.RLock()           
	defer sl.mu.RUnlock()   

	current := sl.header
	// Traverse the list to find the node with the given key.
	for i := sl.level - 1; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].key < key {
			current = current.forward[i]
		}
	}

	current = current.forward[0]
	if current != nil && current.key == key {
		return current.value, true
	}
	return V{}, false
}

// Query returns all elements in the skip list (in order) with keys between start and end inclusive.
func (sl *SkipListImpl[K cmp.Ordered, V any]) Query(ctx context.Context, start K, end K) ([]Pair[K, V], error) {
	sl.mu.RLock()     
	defer sl.mu.RUnlock()    

	var results []Pair[K, V]

	current := sl.header.forward[0]
	// Traverse the list to find the starting node for the query.
	for current != nil && current.key < start {
		current = current.forward[0]
	}

	// Traverse the list and collect all nodes between start and end keys.
	for current != nil && current.key <= end {
		select {
		case <-ctx.Done():   // Check for context cancellation.
			return nil, ctx.Err()
		default:
			results = append(results, Pair[K, V]{Key: current.key, Value: current.value})
			current = current.forward[0]
		}
	}

	return results, nil
}
