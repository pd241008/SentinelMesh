package node

import (
	"testing"

	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
)

func TestNew(t *testing.T) {
	n := New(0, []dataset.Flow{{ID: 1, Category: "normal"}}, 0.3)
	if n.ID != 0 {
		t.Fatalf("expected ID 0, got %d", n.ID)
	}
	if n.DigestCache == nil {
		t.Fatal("DigestCache should be initialized")
	}
}

func TestProcessFlowsForRound(t *testing.T) {
	n := New(0, []dataset.Flow{
		{ID: 1, Category: "normal", Sbytes: 100},
		{ID: 2, Category: "dos", IsAttack: true, Sbytes: 5000},
	}, 0.3)

	d, ok := n.ProcessFlowsForRound(1)
	if !ok || d == nil {
		t.Fatal("expected digest for round 1")
	}
	if d.NodeID != 0 || d.Round != 1 || d.Category != "normal" {
		t.Errorf("bad digest: %+v", d)
	}

	d2, ok := n.ProcessFlowsForRound(2)
	if !ok || d2 == nil {
		t.Fatal("expected digest for round 2")
	}
	if d2.Category != "dos" {
		t.Errorf("expected dos category, got %s", d2.Category)
	}

	d3, ok := n.ProcessFlowsForRound(3)
	if ok || d3 != nil {
		t.Fatal("expected nil after last flow")
	}
}

func TestReceiveDigest(t *testing.T) {
	n := New(0, nil, 0.3)
	d1 := Digest{NodeID: 1, Score: 0.5, Category: "dos", Round: 1}
	d2 := Digest{NodeID: 1, Score: 0.8, Category: "dos", Round: 2}

	n.ReceiveDigest(d1)
	if n.DigestCache[1].Score != 0.5 {
		t.Fatal("expected score 0.5 after first receive")
	}

	n.ReceiveDigest(d2)
	if n.DigestCache[1].Score != 0.8 {
		t.Fatal("expected score 0.8 after newer receive")
	}

	stale := Digest{NodeID: 1, Score: 0.3, Category: "dos", Round: 1}
	n.ReceiveDigest(stale)
	if n.DigestCache[1].Score != 0.8 {
		t.Fatal("should not overwrite with older round")
	}
}

func TestGetCache(t *testing.T) {
	n := New(0, nil, 0.3)
	cache := n.GetCache()
	if cache == nil {
		t.Fatal("GetCache returned nil")
	}
	if len(cache) != 0 {
		t.Fatal("expected empty cache")
	}
}
