package skipList

import (
	"cmp"
	"sync"
)

// Node represents an individual node in the skip list.
type Node[K cmp.Ordered, V any] struct {
	mu          sync.Mutex
	key         K
	value       V
	topLevel    int           // Highest level list that contains this node
	next        []*Node[K, V] // Slice of next pointers at each level
	marked      bool          // Is the node marked for removal
	fullyLinked bool          // Has this node been fully added to the lists
}

func NewNode[K cmp.Ordered, V any](key K, value V, level int) *Node[K, V] {
	return &Node[K, V]{
		key:         key,
		value:       value,
		next:        make([]*Node[K, V], level),
		topLevel:    level - 1, // Assuming level is 1-based; adjust if 0-based
		marked:      false,
		fullyLinked: false,
	}
}
