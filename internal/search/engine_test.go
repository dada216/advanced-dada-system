package search_test

import (
	"database/sql"
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
	results, err := search.Query("keyword", false, false, "")
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

func TestSearchFilters(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	metaDB, err := meta.Open()
	if err != nil {
		t.Fatalf("Failed to open meta db: %v", err)
	}
	defer metaDB.Close()

	uuidProd, _ := metaDB.CreateLocalSession("SessionProd", "default")
	uuidDev, _ := metaDB.CreateLocalSession("SessionDev", "default")

	// Setup Prod Session
	dbProd, _ := sessiondb.Open(uuidProd)
	defer dbProd.Close()
	_ = dbProd.InjectMetadata(sql.NullString{String: "customer-a", Valid: true}, sql.NullString{String: "prod-server", Valid: true}, sql.NullString{String: "project-x", Valid: true})
	_ = dbProd.WriteChunk([]byte("systemctl restart nginx"), 0) // input
	_ = dbProd.WriteChunk([]byte("nginx started successfully"), 1) // output
	_ = dbProd.WriteChunk([]byte("secret_password_typed"), 0) // input

	// Setup Dev Session
	dbDev, _ := sessiondb.Open(uuidDev)
	defer dbDev.Close()
	_ = dbDev.InjectMetadata(sql.NullString{String: "customer-b", Valid: true}, sql.NullString{String: "dev-server", Valid: true}, sql.NullString{String: "project-y", Valid: true})
	_ = dbDev.WriteChunk([]byte("systemctl restart nginx"), 0) // input
	_ = dbDev.WriteChunk([]byte("nginx failed to start"), 1) // output
	_ = dbDev.WriteChunk([]byte("secret_password_typed"), 0) // input

	time.Sleep(100 * time.Millisecond)

	// Test 1: No filters (finds both inputs and outputs across all sessions)
	results, err := search.Query("nginx", false, false, "")
	if err != nil { t.Fatalf("Query failed: %v", err) }
	if len(results) != 4 { t.Errorf("Expected 4 results for 'nginx' without filters, got %d", len(results)) }

	// Test 2: Input filter only
	results, err = search.Query("nginx", true, false, "")
	if err != nil { t.Fatalf("Query failed: %v", err) }
	if len(results) != 2 { t.Errorf("Expected 2 results for 'nginx' with inputOnly=true, got %d", len(results)) }

	// Test 3: Output filter only
	results, err = search.Query("nginx", false, true, "")
	if err != nil { t.Fatalf("Query failed: %v", err) }
	if len(results) != 2 { t.Errorf("Expected 2 results for 'nginx' with outputOnly=true, got %d", len(results)) }

	// Test 4: Tag filter (prod-server)
	results, err = search.Query("nginx", false, false, "prod-server")
	if err != nil { t.Fatalf("Query failed: %v", err) }
	if len(results) != 2 { t.Errorf("Expected 2 results for 'nginx' with tag='prod-server', got %d", len(results)) }
	for _, r := range results {
		if r.SessionUUID != uuidProd { t.Errorf("Expected only prod session, got %s", r.SessionUUID) }
	}

	// Test 5: Tag filter (customer-b) + Input filter
	results, err = search.Query("secret_password_typed", true, false, "customer-b")
	if err != nil { t.Fatalf("Query failed: %v", err) }
	if len(results) != 1 { t.Errorf("Expected 1 result for 'secret_password_typed' with inputOnly + tag='customer-b', got %d", len(results)) }
	if len(results) > 0 && results[0].SessionUUID != uuidDev { t.Errorf("Expected dev session") }
}
