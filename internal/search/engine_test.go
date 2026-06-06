package search_test

import (
	"testing"
	"time"

	"github.com/advanced-dada-system/ads/internal/meta"
	"github.com/advanced-dada-system/ads/internal/search"
	"github.com/advanced-dada-system/ads/internal/sessiondb"
)

func TestFederatedSearchAllSessions(t *testing.T) {
	// Setup a clean test environment
	t.Setenv("HOME", t.TempDir())

	// Initialize Meta DB
	metaDB, err := meta.Open()
	if err != nil {
		t.Fatalf("Failed to open meta db: %v", err)
	}
	defer metaDB.Close()

	// Create Session A and B
	uuidA, err := metaDB.CreateLocalSession("SessionA", "default")
	if err != nil {
		t.Fatalf("Failed to create session A: %v", err)
	}

	uuidB, err := metaDB.CreateLocalSession("SessionB", "default")
	if err != nil {
		t.Fatalf("Failed to create session B: %v", err)
	}

	// Wait briefly to simulate real usage
	time.Sleep(10 * time.Millisecond)

	// Open session DBs and write distinct searchable text
	dbA, err := sessiondb.Open(uuidA)
	if err != nil {
		t.Fatalf("Failed to open session A db: %v", err)
	}
	defer dbA.Close()

	err = dbA.WriteChunk([]byte("This is a unique keyword unique_to_session_a"), 1)
	if err != nil {
		t.Fatalf("Failed to write to session A: %v", err)
	}

	dbB, err := sessiondb.Open(uuidB)
	if err != nil {
		t.Fatalf("Failed to open session B db: %v", err)
	}
	defer dbB.Close()

	err = dbB.WriteChunk([]byte("This is a unique keyword unique_to_session_b"), 1)
	if err != nil {
		t.Fatalf("Failed to write to session B: %v", err)
	}

	// Wait to ensure writes are flushed and visible
	time.Sleep(100 * time.Millisecond)

	// Now run federated search for "keyword" which exists in both
	results, err := search.Query("keyword")
	if err != nil {
		t.Fatalf("Search query failed: %v", err)
	}

	foundA := false
	foundB := false
	for _, res := range results {
		if res.SessionUUID == uuidA {
			foundA = true
		}
		if res.SessionUUID == uuidB {
			foundB = true
		}
	}

	if !foundA || !foundB {
		t.Errorf("Federated search failed. Found Session A: %v, Found Session B: %v", foundA, foundB)
	}
}
