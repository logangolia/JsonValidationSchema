package skiplist

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestUpsertInsert(t *testing.T) {
	sl := NewSkipList[int, string]()

	key := 1
	value := "test"

	updated, err := sl.Upsert(key, func(k int, v string, exists bool) (string, error) {
		return value, nil
	})

	if err != nil || !updated {
		t.Fatalf("Error during Upsert or not updated: %v", err)
	}
}

func TestUpsertUpdate(t *testing.T) {
	sl := NewSkipList[int, string]()

	key := 1
	initialValue := "initial"
	updatedValue := "updated"

	sl.Upsert(key, func(k int, v string, exists bool) (string, error) {
		return initialValue, nil
	})

	updated, err := sl.Upsert(key, func(k int, v string, exists bool) (string, error) {
		return updatedValue, nil
	})

	if err != nil || !updated {
		t.Fatalf("Error during Upsert or not updated: %v", err)
	}

	value, found := sl.Find(key)
	if !found || value != updatedValue {
		t.Errorf("Expected updated value %v, got %v", updatedValue, value)
	}
}

func TestUpsertWithError(t *testing.T) {
	sl := NewSkipList[int, string]()

	key := 1

	_, err := sl.Upsert(key, func(k int, v string, exists bool) (string, error) {
		return "", errors.New("mock error")
	})

	if err == nil {
		t.Fatal("Expected an error during Upsert, but got none")
	}
}

func TestFindExisting(t *testing.T) {
	sl := NewSkipList[int, string]()

	key := 1
	value := "test"

	sl.Upsert(key, func(k int, v string, exists bool) (string, error) {
		return value, nil
	})

	foundValue, found := sl.Find(key)
	if !found || foundValue != value {
		t.Errorf("Expected value %v, got %v", value, foundValue)
	}
}

func TestFindNonExisting(t *testing.T) {
	sl := NewSkipList[int, string]()

	key := 1

	_, found := sl.Find(key)
	if found {
		t.Error("Found a value for a non-existing key")
	}
}

func TestRemoveExisting(t *testing.T) {
	sl := NewSkipList[int, string]()

	key := 2
	value := "toRemove"

	sl.Upsert(key, func(k int, v string, exists bool) (string, error) {
		return value, nil
	})

	removedValue, removed := sl.Remove(key)
	if !removed || removedValue != value {
		t.Errorf("Expected removed value %v, got %v", value, removedValue)
	}
}

func TestRemoveNonExisting(t *testing.T) {
	sl := NewSkipList[int, string]()

	key := 2

	_, removed := sl.Remove(key)
	if removed {
		t.Error("Removed a non-existing key")
	}
}

// TestUpsertMultiple will check the ability of the skip list to insert multiple items.
func TestUpsertMultiple(t *testing.T) {
	sl := NewSkipList[int, string]()

	values := map[int]string{
		1: "one",
		2: "two",
		3: "three",
	}

	for key, value := range values {
		updated, err := sl.Upsert(key, func(k int, v string, exists bool) (string, error) {
			return value, nil
		})
		if err != nil || !updated {
			t.Fatalf("Error during Upsert for key %v: %v", key, err)
		}
	}

	for key, expectedValue := range values {
		value, found := sl.Find(key)
		if !found || value != expectedValue {
			t.Errorf("Expected value %v for key %v, got %v", expectedValue, key, value)
		}
	}
}

// TestUpsertConcurrent will perform concurrent upsert operations.
func TestUpsertConcurrent(t *testing.T) {
	sl := NewSkipList[int, string]()
	const numGoroutines = 100
	key := 1
	updatedCh := make(chan bool, numGoroutines)
	errCh := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			value := "test" + fmt.Sprint(idx)
			updated, err := sl.Upsert(key, func(k int, v string, exists bool) (string, error) {
				return value, nil
			})
			updatedCh <- updated
			errCh <- err
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		updated := <-updatedCh
		err := <-errCh
		if err != nil || !updated {
			t.Fatalf("Error during concurrent Upsert: %v", err)
		}
	}
}

// TestRemoveConcurrent will perform concurrent remove operations.
func TestRemoveConcurrent(t *testing.T) {
	sl := NewSkipList[int, string]()
	key := 1
	value := "test"
	sl.Upsert(key, func(k int, v string, exists bool) (string, error) {
		return value, nil
	})

	const numGoroutines = 10
	removedCh := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, removed := sl.Remove(key)
			removedCh <- removed
		}()
	}

	removedCount := 0
	for i := 0; i < numGoroutines; i++ {
		removed := <-removedCh
		if removed {
			removedCount++
		}
	}
	if removedCount != 1 {
		t.Fatalf("Expected only one removal to be successful, but got %v", removedCount)
	}
}

func TestQueryEmptyList(t *testing.T) {
	sl := NewSkipList[int, string]()
	ctx := context.TODO()

	// Query on an empty list
	startKey, endKey := 1, 10
	results, err := sl.Query(ctx, startKey, endKey)

	if err != nil {
		t.Fatalf("Error during Query: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected empty result, got %v", results)
	}
}

func TestQueryWithSingleElement(t *testing.T) {
	sl := NewSkipList[int, string]()
	ctx := context.TODO()

	key, value := 5, "five"
	sl.Upsert(key, func(k int, v string, exists bool) (string, error) {
		return value, nil
	})

	// Query that should return the single element
	startKey, endKey := 1, 10
	results, err := sl.Query(ctx, startKey, endKey)

	if err != nil {
		t.Fatalf("Error during Query: %v", err)
	}

	if len(results) != 1 || results[0].Key != key || results[0].Value != value {
		t.Errorf("Expected [%v], got %v", Pair[int, string]{Key: key, Value: value}, results)
	}
}

func TestQueryInRange(t *testing.T) {
	sl := NewSkipList[int, string]()
	ctx := context.TODO()

	values := map[int]string{
		1:  "one",
		5:  "five",
		10: "ten",
		15: "fifteen",
	}

	for key, value := range values {
		sl.Upsert(key, func(k int, v string, exists bool) (string, error) {
			return value, nil
		})
	}

	// Query that should return subset of elements
	startKey, endKey := 5, 10
	expectedResults := []Pair[int, string]{
		{Key: 5, Value: "five"},
		{Key: 10, Value: "ten"},
	}

	results, err := sl.Query(ctx, startKey, endKey)

	if err != nil {
		t.Fatalf("Error during Query: %v", err)
	}

	if len(results) != len(expectedResults) {
		t.Errorf("Expected results of length %v, got %v", len(expectedResults), len(results))
	}

	for i, pair := range expectedResults {
		if results[i] != pair {
			t.Errorf("Expected %v, got %v at index %d", pair, results[i], i)
		}
	}
}

func TestQueryOutOfRange(t *testing.T) {
	sl := NewSkipList[int, string]()
	ctx := context.TODO()

	values := map[int]string{
		1:  "one",
		5:  "five",
		10: "ten",
	}

	for key, value := range values {
		sl.Upsert(key, func(k int, v string, exists bool) (string, error) {
			return value, nil
		})
	}

	// Query out of range of inserted elements
	startKey, endKey := 15, 20
	results, err := sl.Query(ctx, startKey, endKey)

	if err != nil {
		t.Fatalf("Error during Query: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected empty result, got %v", results)
	}
}

func TestQueryWithTimeout(t *testing.T) {
	sl := NewSkipList[int, string]()
	ctx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()

	values := map[int]string{
		1:  "one",
		5:  "five",
		10: "ten",
	}

	for key, value := range values {
		sl.Upsert(key, func(k int, v string, exists bool) (string, error) {
			return value, nil
		})
	}

	// This simulates a situation where the query runs longer than expected.
	// This timeout duration is super short, so the query is expected to fail.
	_, err := sl.Query(ctx, 1, 10)

	if err != context.DeadlineExceeded {
		t.Fatalf("Expected DeadlineExceeded error, got: %v", err)
	}
}
