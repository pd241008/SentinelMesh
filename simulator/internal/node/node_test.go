package node

import (
	"testing"

	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
	"github.com/pd241008/sentinelmesh/simulator/internal/scorer"
)

func defaultThresholds() scorer.Thresholds {
	return scorer.Thresholds{
		Recon: 1.66,
		DoS:   0.82,
		Ewma:  0.89,
	}
}

func TestNew(t *testing.T) {
	n := New(0, []dataset.Flow{{ID: 1, Category: "normal", Timestamp: 1.0}}, 0.3, defaultThresholds(), 10)
	if n.ID != 0 {
		t.Fatalf("expected ID 0, got %d", n.ID)
	}
	if n.DigestCache == nil {
		t.Fatal("DigestCache should be initialized")
	}
	if n.Window != 10 {
		t.Fatalf("expected Window 10, got %d", n.Window)
	}
}

func TestProcessFlowsForRound(t *testing.T) {
	thresh := defaultThresholds()
	n := New(0, []dataset.Flow{
		{ID: 1, Category: "dos", IsAttack: true, Sbytes: 5000, Dbytes: 10000, Spkts: 50, Dpkts: 100, Rate: 500, CtDstLtm: 1, CtDstSportLtm: 1, Timestamp: 1.0},
		{ID: 2, Category: "dos", IsAttack: true, Sbytes: 6000, Dbytes: 12000, Spkts: 60, Dpkts: 120, Rate: 600, CtDstLtm: 1, CtDstSportLtm: 1, Timestamp: 2.0},
	}, 0.3, thresh, 10)

	d, ok := n.ProcessFlowsForRound(1)
	if !ok || d == nil {
		t.Fatal("expected digest for round 1")
	}
	if d.NodeID != 0 || d.Round != 1 {
		t.Errorf("bad digest: %+v", d)
	}

	d2, ok := n.ProcessFlowsForRound(2)
	if !ok || d2 == nil {
		t.Fatal("expected digest for round 2")
	}
	if d2.Category != "reconnaissance" && d2.Category != "dos" {
		t.Errorf("expected reconnaissance or dos category, got %s", d2.Category)
	}

	d3, ok := n.ProcessFlowsForRound(3)
	if ok || d3 != nil {
		t.Fatal("expected nil after last flow")
	}
}

func TestReceiveDigest(t *testing.T) {
	thresh := defaultThresholds()
	n := New(0, nil, 0.3, thresh, 10)
	d1 := Digest{NodeID: 1, Score: 0.5, Category: "dos", Round: 1}
	d2 := Digest{NodeID: 1, Score: 0.8, Category: "dos", Round: 2}

	n.ReceiveDigest(d1)
	if len(n.DigestCache[1]) != 1 || n.DigestCache[1][0].Score != 0.5 {
		t.Fatal("expected score 0.5 after first receive")
	}

	n.ReceiveDigest(d2)
	if len(n.DigestCache[1]) != 2 || n.DigestCache[1][1].Score != 0.8 {
		t.Fatal("expected two digests after second receive")
	}

	stale := Digest{NodeID: 1, Score: 0.3, Category: "dos", Round: 1}
	n.ReceiveDigest(stale)
	if len(n.DigestCache[1]) != 2 {
		t.Fatal("should not add duplicate round from same source")
	}
}

func TestPurgeStale(t *testing.T) {
	thresh := defaultThresholds()
	n := New(0, nil, 0.3, thresh, 2)
	n.ReceiveDigest(Digest{NodeID: 1, Score: 0.5, Category: "dos", Round: 1})
	n.ReceiveDigest(Digest{NodeID: 2, Score: 0.6, Category: "dos", Round: 2})
	n.ReceiveDigest(Digest{NodeID: 1, Score: 0.7, Category: "dos", Round: 3})

	n.PurgeStale(4)

	// Node 1: round 1 purged (4-1=3 > window 2), round 3 kept (4-3=1 <= 2)
	if len(n.DigestCache[1]) != 1 || n.DigestCache[1][0].Round != 3 {
		t.Errorf("expected node 1 to keep only round 3 digest, got %d digests", len(n.DigestCache[1]))
	}
	// Node 2: round 2 kept (4-2=2 <= window 2)
	if len(n.DigestCache[2]) != 1 || n.DigestCache[2][0].Round != 2 {
		t.Errorf("expected node 2 to keep round 2 digest, got %d digests", len(n.DigestCache[2]))
	}
}

func TestGetCache(t *testing.T) {
	thresh := defaultThresholds()
	n := New(0, nil, 0.3, thresh, 10)
	cache := n.GetCache()
	if cache == nil {
		t.Fatal("GetCache returned nil")
	}
	if len(cache) != 0 {
		t.Fatal("expected empty cache")
	}
}
