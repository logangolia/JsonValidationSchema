package skiplist

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
	isHead      bool          // Marker if this is the head of the skiplist
	isTail      bool          // Marker if this is the tail of the skiplist
}

func NewNode[K cmp.Ordered, V any](key K, value V) *Node[K, V] {
	return &Node[K, V]{
		key:         key,
		value:       value,
		next:        make([]*Node[K, V], maxLevel+1),
		topLevel:    maxLevel,
		marked:      false,
		fullyLinked: false,
		isHead:      false,
		isTail:      false,
	}
}
