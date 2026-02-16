package proxy

import (
	"testing"
)

// TestProverManagerGetAndCreate validates basic creation and retrieval semantics.
func TestProverManagerGetAndCreate(t *testing.T) {
	pm := NewProverManager(2, nil)

	if got := pm.Get("user1"); got != nil {
		t.Fatalf("expected nil for non-existent key, got: %+v", got)
	}

	sess1 := pm.GetOrCreate("user1")
	if sess1 == nil {
		t.Fatalf("expected non-nil session from GetOrCreate")
	}

	// Should be stable on subsequent Get
	if got := pm.Get("user1"); got != sess1 {
		t.Fatalf("expected same session pointer on Get, got different instance: %p vs %p", got, sess1)
	}
}

// TestProverManagerRolloverAndPromotion verifies rollover when sizeLimit is reached
// and that old entries are accessible and promoted back to active data map.
func TestProverManagerRolloverAndPromotion(t *testing.T) {
	pm := NewProverManager(2, nil)

	s1 := pm.GetOrCreate("u1")
	s2 := pm.GetOrCreate("u2")
	if s1 == nil || s2 == nil {
		t.Fatalf("expected sessions to be created for u1/u2")
	}

	// Precondition: data should contain 2 entries, no deprecated yet.
	pm.RLock()
	if len(pm.data) != 2 {
		pm.RUnlock()
		t.Fatalf("expected data len=2 before rollover, got %d", len(pm.data))
	}
	if len(pm.willDeprecatedData) != 0 {
		pm.RUnlock()
		t.Fatalf("expected willDeprecatedData len=0 before rollover, got %d", len(pm.willDeprecatedData))
	}
	pm.RUnlock()

	// Trigger rollover by creating a third key.
	s3 := pm.GetOrCreate("u3")
	if s3 == nil {
		t.Fatalf("expected session for u3 after rollover")
	}

	// After rollover: current data should only have u3, deprecated should hold u1 and u2.
	pm.RLock()
	if len(pm.data) != 1 {
		pm.RUnlock()
		t.Fatalf("expected data len=1 after rollover (only u3), got %d", len(pm.data))
	}
	if _, ok := pm.data["u3"]; !ok {
		pm.RUnlock()
		t.Fatalf("expected 'u3' to be in active data after rollover")
	}
	if len(pm.willDeprecatedData) != 2 {
		pm.RUnlock()
		t.Fatalf("expected willDeprecatedData len=2 after rollover, got %d", len(pm.willDeprecatedData))
	}
	pm.RUnlock()

	// Accessing an old key should return the same pointer and promote it to active data map.
	got1 := pm.Get("u1")
	if got1 != s1 {
		t.Fatalf("expected same pointer for u1 after promotion, got %p want %p", got1, s1)
	}

	// The promotion should add it to active data (without enforcing size limit on promotion).
	pm.RLock()
	if _, ok := pm.data["u1"]; !ok {
		pm.RUnlock()
		t.Fatalf("expected 'u1' to be present in active data after promotion")
	}
	if len(pm.data) != 2 {
		// Now should contain u3 and u1
		pm.RUnlock()
		t.Fatalf("expected data len=2 after promotion of u1, got %d", len(pm.data))
	}
	pm.RUnlock()

	// Access the other deprecated key and ensure behavior is consistent.
	got2 := pm.Get("u2")
	if got2 != s2 {
		t.Fatalf("expected same pointer for u2 after promotion, got %p want %p", got2, s2)
	}

	pm.RLock()
	if _, ok := pm.data["u2"]; !ok {
		pm.RUnlock()
		t.Fatalf("expected 'u2' to be present in active data after promotion")
	}
	// Note: promotion does not enforce sizeLimit, so data can grow beyond sizeLimit after promotions.
	if len(pm.data) != 3 {
		pm.RUnlock()
		t.Fatalf("expected data len=3 after promoting both u1 and u2, got %d", len(pm.data))
	}
	pm.RUnlock()
}
