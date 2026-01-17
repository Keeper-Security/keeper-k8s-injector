package cache

import (
	"sync"
	"testing"
	"time"
)

func TestSecretCache_SetGet(t *testing.T) {
	cache := NewSecretCache(1 * time.Hour)

	// Set a secret
	data := []byte("test-secret-value")
	cache.Set("test-secret", data)

	// Get should return the secret
	cached, ok := cache.Get("test-secret")
	if !ok {
		t.Error("expected secret to be in cache")
	}
	if string(cached.Data) != string(data) {
		t.Errorf("cached data = %q, want %q", cached.Data, data)
	}

	// Non-existent secret
	_, ok = cache.Get("nonexistent")
	if ok {
		t.Error("expected false for nonexistent secret")
	}
}

func TestSecretCache_Expiry(t *testing.T) {
	cache := NewSecretCache(100 * time.Millisecond)  // Short TTL for testing

	// Set a secret
	cache.Set("test-secret", []byte("value"))

	// Should be available immediately
	_, ok := cache.Get("test-secret")
	if !ok {
		t.Error("secret should be available immediately after set")
	}

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, ok = cache.Get("test-secret")
	if ok {
		t.Error("secret should be expired after TTL")
	}
}

func TestSecretCache_Age(t *testing.T) {
	cache := NewSecretCache(1 * time.Hour)

	// Non-existent secret
	age := cache.Age("nonexistent")
	if age != 0 {
		t.Errorf("age of nonexistent secret should be 0, got %v", age)
	}

	// Set and check age
	cache.Set("test-secret", []byte("value"))
	time.Sleep(50 * time.Millisecond)

	age = cache.Age("test-secret")
	if age < 40*time.Millisecond || age > 100*time.Millisecond {
		t.Errorf("age should be ~50ms, got %v", age)
	}
}

func TestSecretCache_Size(t *testing.T) {
	cache := NewSecretCache(1 * time.Hour)

	if cache.Size() != 0 {
		t.Errorf("new cache should have size 0, got %d", cache.Size())
	}

	cache.Set("secret1", []byte("value1"))
	cache.Set("secret2", []byte("value2"))

	if cache.Size() != 2 {
		t.Errorf("cache size should be 2, got %d", cache.Size())
	}
}

func TestSecretCache_Clear(t *testing.T) {
	cache := NewSecretCache(1 * time.Hour)

	cache.Set("secret1", []byte("value1"))
	cache.Set("secret2", []byte("value2"))

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("cache should be empty after Clear(), got size %d", cache.Size())
	}

	_, ok := cache.Get("secret1")
	if ok {
		t.Error("secret1 should not exist after Clear()")
	}
}

func TestSecretCache_ConcurrentAccess(t *testing.T) {
	cache := NewSecretCache(1 * time.Hour)
	var wg sync.WaitGroup

	// Concurrent writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cache.Set("secret", []byte("value"))
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cache.Get("secret")
				cache.Age("secret")
			}
		}(i)
	}

	wg.Wait()

	// Should not panic and should have the secret
	_, ok := cache.Get("secret")
	if !ok {
		t.Error("secret should exist after concurrent access")
	}
}

func TestSecretCache_DefaultMaxAge(t *testing.T) {
	cache := NewSecretCache(0)  // 0 should default to 24h

	// Set secret
	cache.Set("test", []byte("value"))

	// Should be available for a long time
	time.Sleep(10 * time.Millisecond)
	_, ok := cache.Get("test")
	if !ok {
		t.Error("secret should be available with default 24h TTL")
	}
}

func TestSecretCache_UpdateExistingSecret(t *testing.T) {
	cache := NewSecretCache(1 * time.Hour)

	// Set initial value
	cache.Set("secret", []byte("value1"))
	time.Sleep(10 * time.Millisecond)

	// Update value
	cache.Set("secret", []byte("value2"))

	// Get should return new value
	cached, ok := cache.Get("secret")
	if !ok {
		t.Error("secret should exist")
	}
	if string(cached.Data) != "value2" {
		t.Errorf("cached data = %q, want %q", cached.Data, "value2")
	}

	// Age should be recent (from second Set)
	age := cache.Age("secret")
	if age > 50*time.Millisecond {
		t.Errorf("age should be recent after update, got %v", age)
	}
}
