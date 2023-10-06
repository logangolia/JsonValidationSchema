package skiplist

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
	Find(key K) (foundValue V, found bool)
	Upsert(key K, check UpdateCheck[K, V]) (updated bool, err error)
	Remove(key K) (removedValue V, removed bool)
	Query(ctx context.Context, start K, end K) (results []Node[K, V], err error)
}

// SkipListImpl is the concrete implementation of the SkipList interface.
type SkipListImpl[K cmp.Ordered, V any] struct {
	head *Node[K, V] // Head node of the skip list.
	tail *Node[K, V] // Tail node of the skip list.
}

// NewSkipList initializes and returns a new SkipListImpl.
func NewSkipList[K cmp.Ordered, V any]() SkipList[K, V] {
	var defaultK K
	var defaultV V
	headNode := NewNode[K, V](defaultK, defaultV)
	headNode.fullyLinked = true
	headNode.isHead = true
	tailNode := NewNode[K, V](defaultK, defaultV)
	tailNode.fullyLinked = true
	tailNode.isTail = true
	// Make the head's next pointers point to the tail node for all levels
	for i := 0; i <= maxLevel; i++ {
		headNode.next[i] = tailNode
	}

	return &SkipListImpl[K, V]{
		head: headNode,
		tail: tailNode,
	}
}

// Find retrieves a value from the skip list by key.
// If the key exists, it returns the associated value and true.
// Otherwise, it returns the zero value of V and false.
func (sl *SkipListImpl[K, V]) Find(key K) (V, bool) {
	levelFound, _, succs := sl.findHelper(key)

	// If the key was not found, return an empty V and false
	if levelFound == -1 {
		var defaultV V
		return defaultV, false
	}

	// If the key was found it is stored in succs at the level
	found := succs[levelFound]
	// Return the value and true iff the node is fullyLinked and not marked
	return found.value, (found.fullyLinked && !found.marked)
}

// findHelper finds the level, predecessors, and successors to a given key
func (sl *SkipListImpl[K, V]) findHelper(key K) (int, []*Node[K, V], []*Node[K, V]) {
	foundLevel := -1
	pred := sl.head

	preds := make([]*Node[K, V], maxLevel+1)
	succs := make([]*Node[K, V], maxLevel+1)

	// Starting from the maxLevel, traverse down and across the skiplist to the node
	level := maxLevel
	for level >= 0 {
		curr := pred.next[level]
		// Continue until we reach the tail node or a key greater or equal to the desired key.
		// We treat head nodes as if they have a key less than any key
		// and tail nodes as if they have a key greater than any key.
		for !curr.isTail && (curr.isHead || cmp.Compare(key, curr.key) > 0) {
			pred = curr
			curr = pred.next[level]
		}
		// If this is the first time the key has been found, set the foundLevel to the current level
		if foundLevel == -1 && cmp.Compare(key, curr.key) == 0 {
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
	// Choose a random level as the topLevel to insert (for balancing)
	topLevel := sl.randomLevel()

	for true {
		// Check if key is in the list
		var checkValue V
		levelFound, preds, succs := sl.findHelper(key)
		// fmt.Println("preds:", preds)
		// fmt.Println("succs:", succs)
		// fmt.Println("foundLevel:", levelFound)
		// If the key is in the list
		if levelFound != -1 {
			// fmt.Println("you shouldnt be here")
			found := succs[levelFound]
			checkValue = found.value
			if found.marked {
				// Adding node, wait for other operation, and fail
				for !found.fullyLinked {
				}
				return false, nil
			} else {
				// The node was found and is not inside another operation, so update it
				value, err := check(key, checkValue, true)
				if err != nil {
					return false, err
				}
				found.mu.Lock()
				found.value = value
				found.mu.Unlock()
				return true, nil
			}
		}
		// fmt.Println("you should be here")
		// Key was not found, so we have to Insert it
		value, err := check(key, checkValue, levelFound != -1)
		if err != nil {
			return false, err
		}
		// fmt.Println("value: ", value)
		// Lock the predecessors
		highestLocked := -1
		valid := true
		level := 0
		// Ascend the levels to the topLevel, checking that the location is suitable for insertion
		// fmt.Println("topLevel: ", topLevel)
		lastLockedNode := (*Node[K, V])(nil) // Initialize to nil. This will hold reference to the last node we locked.
		for valid && level <= topLevel {
			pred := preds[level]
			if pred != lastLockedNode {
				pred.mu.Lock()
				lastLockedNode = pred // Update the reference to the last locked node
			}
			highestLocked = level
			// Ensure the predecessor and successors are not marked for removal
			unmarked := (!preds[level].marked && !succs[level].marked)
			// Ensure there exists no node between the predecessor and successor to the inserted node
			connected := (preds[level].next[level] == succs[level])
			valid = unmarked && connected
			level = level + 1
		}
		// fmt.Println("exited preds loop")
		// If the location became invalid for any reason, unlock and restart
		if !valid {
			for level := 0; level <= highestLocked; level++ {
				preds[level].mu.Unlock()
			}
			continue // Return to start of the loop
		}
		// fmt.Println("made through valid check")
		// Create node for insertion
		node := NewNode(key, value)
		node.mu.Lock()
		node.topLevel = topLevel

		// Set pointers of the inserted node
		level = 0
		for level <= topLevel {
			preds[level].next[level] = node
			node.next[level] = succs[level]
			level = level + 1
		}
		// fmt.Println("set inserted pointers")
		// Unlock preds and inserted node
		node.fullyLinked = true
		level = highestLocked
		lastUnlockedNode := (*Node[K, V])(nil) // Initialize to nil. This will hold reference to the last node we unlocked.
		for level >= 0 {
			if preds[level] != lastUnlockedNode {
				preds[level].mu.Unlock()
				lastUnlockedNode = preds[level] // Update the reference to the last unlocked node
			}
			level = level - 1
		}
		node.mu.Unlock()
		// fmt.Println("return?")
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
		// Find the key for removal
		levelFound, preds, succs := sl.findHelper(key)
		if levelFound != -1 {
			victim = succs[levelFound]
		}

		// First iteration
		if !isMarked {
			// Check if the node was not found, it was fullyLinked, it was marked
			// or the topLevel doesn't match the level it was found on
			if levelFound == -1 || !victim.fullyLinked ||
				victim.marked || victim.topLevel != levelFound {
				return defaultV, false
			}

			topLevel = victim.topLevel
			victim.mu.Lock()
			if victim.marked {
				// Another remove call beat us
				victim.mu.Unlock()
				return defaultV, false
			}
			// This remove call controls the node
			victim.marked = true
			isMarked = true
		}
		// Lock the predecessors
		highestLocked := -1
		level := 0
		valid := true
		// Ascend the levels, locking the predecessor and
		// Ensuring the predecessor is not marked for removal, and the successor is the victim
		lastLockedNode := (*Node[K, V])(nil) // Initialize to nil. This will hold reference to the last node we locked.
		for valid && level <= topLevel {
			pred := preds[level]
			if pred != lastLockedNode {
				pred.mu.Lock()
				lastLockedNode = pred // Update the reference to the last locked node
			}
			highestLocked = level
			validSuccessor := (pred.next[level] == victim)
			valid = (!pred.marked && validSuccessor)
			level = level + 1
		}

		// If the removal was not valid for any reason, unlock locked predecessors and try again
		if !valid {
			level = highestLocked
			for level >= 0 {
				preds[level].mu.Unlock()
				level = level - 1
			}
			// Victim remains locked as this removal has ownership
			continue
		}

		// All preds locked and valid, unlink the nodes
		level = topLevel
		for level >= 0 {
			preds[level].next[level] = victim.next[level]
			level = level - 1
		}

		// Unlock the victim and the predecessors
		victim.mu.Unlock()
		lastUnlockedNode := (*Node[K, V])(nil) // Initialize to nil. This will hold reference to the last node we unlocked.
		for level >= 0 {
			if preds[level] != lastUnlockedNode {
				preds[level].mu.Unlock()
				lastUnlockedNode = preds[level] // Update the reference to the last unlocked node
			}
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

// Query returns all elements in the skip list (in order) with keys between start and end inclusive.
func (sl *SkipListImpl[K, V]) Query(ctx context.Context, start K, end K) ([]Node[K, V], error) {
	var results []Node[K, V]
	return results, nil
}
