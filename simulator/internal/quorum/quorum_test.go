package quorum

import (
	"testing"

	"github.com/pd241008/sentinelmesh/simulator/internal/node"
)

func TestEvaluateBelowThreshold(t *testing.T) {
	cache := map[int][]node.Digest{
		1: {{NodeID: 1, Score: 0.3, Category: "dos", Round: 1}},
		2: {{NodeID: 2, Score: 0.4, Category: "dos", Round: 1}},
	}
	alerts := Evaluate(cache, 3, 5, 1, nil) // quorum=3 > 2 nodes, no alert
	if len(alerts) != 0 {
		t.Fatalf("expected no alerts, got %v", alerts)
	}
}

func TestEvaluateQuorumMet(t *testing.T) {
	cache := map[int][]node.Digest{
		1: {{NodeID: 1, Score: 0.7, Category: "dos", Round: 1}},
		2: {{NodeID: 2, Score: 0.8, Category: "dos", Round: 1}},
		3: {{NodeID: 3, Score: 0.6, Category: "dos", Round: 1}},
	}
	alerts := Evaluate(cache, 3, 5, 1, nil)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert category, got %v", alerts)
	}
	if dos, ok := alerts["dos"]; !ok || len(dos) != 3 {
		t.Fatalf("expected 3 dos corroborators, got %v", dos)
	}
}

func TestEvaluateMultipleCategories(t *testing.T) {
	cache := map[int][]node.Digest{
		1: {{NodeID: 1, Score: 0.8, Category: "dos", Round: 1}},
		2: {{NodeID: 2, Score: 0.9, Category: "dos", Round: 1}},
		3: {{NodeID: 3, Score: 0.8, Category: "fuzzers", Round: 1}},
		4: {{NodeID: 4, Score: 0.9, Category: "fuzzers", Round: 1}},
	}
	alerts := Evaluate(cache, 2, 5, 1, nil)
	if len(alerts) != 2 {
		t.Fatalf("expected 2 alert categories, got %v", alerts)
	}
	if _, ok := alerts["dos"]; !ok {
		t.Fatal("expected dos alert")
	}
	if _, ok := alerts["fuzzers"]; !ok {
		t.Fatal("expected fuzzers alert")
	}
}

func TestEvaluateWindowExcluded(t *testing.T) {
	cache := map[int][]node.Digest{
		1: {{NodeID: 1, Score: 0.9, Category: "dos", Round: 1}},
		2: {{NodeID: 2, Score: 0.9, Category: "dos", Round: 1}},
	}
	alerts := Evaluate(cache, 2, 1, 10, nil)
	if len(alerts) != 0 {
		t.Fatalf("expected 0 alerts (outside window), got %v", alerts)
	}
}

func TestEvaluateEmptyCache(t *testing.T) {
	alerts := Evaluate(map[int][]node.Digest{}, 2, 5, 1, nil)
	if len(alerts) != 0 {
		t.Fatalf("expected 0 alerts for empty cache, got %v", alerts)
	}
}

func TestEvaluateDedupBySource(t *testing.T) {
	cache := map[int][]node.Digest{
		1: {
			{NodeID: 1, Score: 0.9, Category: "dos", Round: 1},
			{NodeID: 1, Score: 0.9, Category: "dos", Round: 1}, // duplicate same node, same round
		},
	}
	alerts := Evaluate(cache, 2, 5, 1, nil)
	if len(alerts) != 0 {
		t.Fatalf("expected no alerts (dedup by source), got %v", alerts)
	}
}

func TestEvaluateMultipleRounds(t *testing.T) {
	cache := map[int][]node.Digest{
		1: {{NodeID: 1, Score: 0.9, Category: "dos", Round: 1}},
		2: {{NodeID: 2, Score: 0.9, Category: "dos", Round: 2}},
	}
	// Round 1: node 1 (diff 0 <= 5) only, no quorum of 3
	alerts := Evaluate(cache, 3, 5, 1, nil)
	if len(alerts) != 0 {
		t.Fatalf("expected no alerts at round 1, got %v", alerts)
	}
	// Round 2: node 1 (diff 1 <= 5) and node 2 (diff 0 <= 5), still only 2 sources < quorum 3
	alerts = Evaluate(cache, 3, 5, 2, nil)
	if len(alerts) != 0 {
		t.Fatalf("expected no alerts at round 2 (only 2 sources < quorum 3), got %v", alerts)
	}
	// With quorum=2, round 2 should trigger
	alerts = Evaluate(cache, 2, 5, 2, nil)
	if _, ok := alerts["dos"]; !ok || len(alerts["dos"]) != 2 {
		t.Fatalf("expected 2 dos corroborators at round 2 with quorum=2, got %v", alerts["dos"])
	}
}
