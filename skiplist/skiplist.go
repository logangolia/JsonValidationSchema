package skipList

import (
	"cmp"
	"context"
	"math/rand"
)

// Constants for the maximum number of levels in the skip list and the probability for random level generation.
const (
	maxLevel    = 4
	probability = 0.5
)

// UpdateCheck is a function type that, given a key and its current value, returns the new value to be set.
// If the key does not exist, exists will be false and currValue will be the zero value of V.
type UpdateCheck[K cmp.Ordered, V any] func(key K, currValue V, exists bool) (newValue V, err error)

// SkipList is an interface that defines the methods a skip list should implement.
type SkipList[K cmp.Ordered, V any] interface {
	Upsert(key K, check UpdateCheck[K, V]) (updated bool, err error)
	Remove(key K) (removedValue V, removed bool)
	Find(key K) (foundValue V, found bool)
	Query(ctx context.Context, start K, end K) (results []Node[K, V], err error)
}

// SkipListImpl is the concrete implementation of the SkipList interface.
type SkipListImpl[K cmp.Ordered, V any] struct {
	head  *Node[K, V] // Head node of the skip list.
	tail  *Node[K, V] // Tail node of the skip list.
	level int         // Current number of levels in the skip list.
}

// NewSkipList initializes and returns a new SkipListImpl.
func NewSkipList[K cmp.Ordered, V any](minKey K, maxKey K) *SkipListImpl[K, V] {
	var defaultV V
	return &SkipListImpl[K, V]{
		head: NewNode[K, V](minKey, defaultV, maxLevel),
		tail: NewNode[K, V](maxKey, defaultV, maxLevel),
	}
}

// Find retrieves a value from the skip list by key.
// If the key exists, it returns the associated value and true.
// Otherwise, it returns the zero value of V and false.
func (sl *SkipListImpl[K, V]) Find(key K) (V, bool) {
	levelFound, _, succs := sl.findHelper(key)

	if levelFound == -1 {
		var defaultV V
		return defaultV, false
	}

	found := succs[levelFound]
	return found.value, (found.fullyLinked && !found.marked)
}

// findHelper finds the level, predecessors, and successors to a given key
func (sl *SkipListImpl[K, V]) findHelper(key K) (int, []*Node[K, V], []*Node[K, V]) {
	foundLevel := -1
	pred := sl.head

	preds := make([]*Node[K, V], maxLevel+1)
	succs := make([]*Node[K, V], maxLevel+1)

	level := maxLevel
	for level >= 0 {
		curr := pred.next[level]
		for key > curr.key {
			pred = curr
			curr = pred.next[level]
		}
		if foundLevel == -1 && key == curr.key {
			foundLevel = level
		}
		preds[level] = pred
		succs[level] = curr
		level = level - 1
	}
	return foundLevel, preds, succs
}

// Upsert inserts or updates node in the skip list based on the provided check function.
func (sl *SkipListImpl[K, V]) Upsert(key K, check UpdateCheck[K, V]) (bool, error) {
	topLevel := sl.randomLevel()

	for true {
		// Check if key is in the list
		var checkValue V
		levelFound, preds, succs := sl.findHelper(key)
		if levelFound != -1 {
			found := succs[levelFound]
			checkValue = found.value
			if !found.marked {
				// Adding node wait for other operation
				for !found.fullyLinked {
				}
				return false, nil
			}
		}
		value, err := check(key, checkValue, levelFound != -1)
		if err != nil {
			return false, err
		}

		// Key not found so lock predecessor
		highestLocked := -1
		valid := true
		level := 0
		// Lock preds
		for valid && level <= topLevel {
			preds[level].mu.Lock()
			highestLocked = level
			// Check if pred/succ are valid
			unmarked := (!preds[level].marked && !succs[level].marked)
			connected := (preds[level].next[level] == succs[level])
			valid = unmarked && connected
			level = level + 1
		}
		if !valid {
			// Preds or succs changed, unlocked and try again
			level = highestLocked
			for level >= 0 {
				preds[level].mu.Unlock()
				level = level - 1
			}
		}
		// Insert new node
		node := NewNode(key, value, topLevel)

		// Set pointers
		level = 0
		for level <= topLevel {
			node.next[level] = succs[level]
			level = level + 1
		}

		// Add node to appropriate lists
		level = 0
		for level <= topLevel {
			preds[level].next[level] = node
			level = level + 1
		}
		node.fullyLinked = true
		level = highestLocked
		for level >= 0 {
			preds[level].mu.Unlock()
			level = level - 1
		}

		return true, nil
	}
	return true, nil
}

// Remove deletes a key from the skip list and returns the removed value.
func (sl *SkipListImpl[K, V]) Remove(key K) (V, bool) {
	var defaultV V
	var victim *Node[K, V]
	isMarked := false
	topLevel := -1
	for true {
		levelFound, preds, succs := sl.findHelper(key)
		if levelFound != -1 {
			victim = succs[levelFound]
		}
		if !isMarked {
			if levelFound == -1 || !victim.fullyLinked ||
				victim.marked || victim.topLevel != levelFound {
				return defaultV, false
			}
			topLevel = victim.topLevel
			victim.mu.Lock()
			if victim.marked {
				// Another remove call is operating on the node
				victim.mu.Unlock()
				return defaultV, false
			}
			victim.marked = true
			isMarked = true
		}

		// Victim is locked and marked
		highestLocked := -1
		level := 0
		valid := true
		for valid && (level <= topLevel) {
			pred := preds[level]
			pred.mu.Lock()
			highestLocked = level
			successor := (pred.next[level] == victim)
			valid = (!pred.marked && successor)
			level = level + 1
		}

		if !valid {
			level = highestLocked
			for level >= 0 {
				preds[level].mu.Unlock()
				level = level - 1
			}
			// Preds changed, try again
			continue
		}

		// All preds locked and valid
		level = topLevel
		for level >= 0 {
			preds[level].next[level] = victim.next[level]
			level = level - 1
		}

		victim.mu.Unlock()
		level = highestLocked
		for level >= 0 {
			preds[level].mu.Unlock()
			level = level - 1
		}
		return victim.value, true
	}
	return victim.value, true
}

// randomLevel generates a random level for a new node.
func (sl *SkipListImpl[K, V]) randomLevel() int {
	lvl := 1
	for rand.Float64() < probability && lvl < maxLevel {
		lvl++
	}
	return lvl
}

// // Query returns all elements in the skip list (in order) with keys between start and end inclusive.
// func (sl *SkipListImpl[K, V]) Query(ctx context.Context, start K, end K) ([]Node[K, V], error) {
// 	sl.mu.RLock()
// 	defer sl.mu.RUnlock()

// 	var results []Node[K, V]

// 	current := sl.header.forward[0]
// 	// Traverse the list to find the starting node for the query.
// 	for current != nil && current.key < start {
// 		current = current.forward[0]
// 	}

// 	// Traverse the list and collect all nodes between start and end keys.
// 	for current != nil && current.key <= end {
// 		select {
// 		case <-ctx.Done(): // Check for context cancellation.
// 			return nil, ctx.Err()
// 		default:
// 			results = append(results, Node[K, V]{Key: current.key, Value: current.value})
// 			current = current.forward[0]
// 		}
// 	}
// 	return results, nil
// }
