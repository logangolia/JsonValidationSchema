package skiplist

import (
	"testing"
)

func TestUpsertAndFind(t *testing.T) {
	sl := NewSkipList[int, string](-1000, 1000)

	key := 1
	value := "test"

	_, err := sl.Upsert(key, func(k int, v string, exists bool) (string, error) {
		return value, nil
	})

	if err != nil {
		t.Fatalf("Error during Upsert: %v", err)
	}

	foundValue, found := sl.Find(key)
	if !found || foundValue != value {
		t.Errorf("Expected value %v, got %v", value, foundValue)
	}
}

func TestRemove(t *testing.T) {
	sl := NewSkipList[int, string](-1000, 1000)

	key := 2
	value := "toRemove"

	_, err := sl.Upsert(key, func(k int, v string, exists bool) (string, error) {
		return value, nil
	})

	if err != nil {
		t.Fatalf("Error during Upsert: %v", err)
	}

	removedValue, removed := sl.Remove(key)
	if !removed || removedValue != value {
		t.Errorf("Expected removed value %v, got %v", value, removedValue)
	}

	_, found := sl.Find(key)
	if found {
		t.Error("Key was found after removal")
	}
}
