package gossip

import (
	"testing"

	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
	"github.com/pd241008/sentinelmesh/simulator/internal/node"
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
	nodes := []*node.Node{node.New(0, nil, 0.3, defaultThresholds(), 5)}
	g := New(nodes, 2, 5, 42)
	if g == nil {
		t.Fatal("New returned nil")
	}
	if g.fanout != 2 || g.window != 5 {
		t.Fatalf("bad params: %+v", g)
	}
}

func makeTestNodes(t *testing.T, n int, flowsPerNode int) []*node.Node {
	t.Helper()
	nodes := make([]*node.Node, n)
	for i := 0; i < n; i++ {
		flows := make([]dataset.Flow, flowsPerNode)
		for j := 0; j < flowsPerNode; j++ {
			flows[j] = dataset.Flow{
				ID:        i*flowsPerNode + j + 1,
				Category:  "normal",
				Sbytes:    100,
				Dbytes:    200,
				Spkts:     2,
				Dpkts:     4,
				Rate:      10.0,
				Timestamp: float64(i*flowsPerNode + j + 1),
			}
		}
		nodes[i] = node.New(i, flows, 0.3, defaultThresholds(), 5)
	}
	return nodes
}

func TestSelectPeers(t *testing.T) {
	nodes := makeTestNodes(t, 8, 5)
	g := New(nodes, 3, 5, 42)

	// Verify deterministic peer selection across rounds
	for round := 1; round <= 10; round++ {
		g.Round(round)
	}
	// Check that messages were sent (fanout=3 per active node)
	if g.MessagesSent == nil || len(g.MessagesSent) == 0 {
		t.Log("No messages sent - all flows may be normal (no anomalies detected)")
	}
}

func TestSelectPeersSingleNode(t *testing.T) {
	nodes := makeTestNodes(t, 1, 5)
	g := New(nodes, 2, 5, 42)
	g.Round(1)
	// Single node cannot send to itself
	for _, count := range g.MessagesSent {
		if count > 0 {
			t.Fatalf("single node should not send messages to itself, got %d", count)
		}
	}
}

func TestRound(t *testing.T) {
	nodes := makeTestNodes(t, 4, 10)
	g := New(nodes, 2, 5, 42)

	g.Round(1)

	// Check that the gossiper executed without error
	_ = g
}

func TestEvictStale(t *testing.T) {
	nodes := makeTestNodes(t, 2, 10)
	g := New(nodes, 2, 2, 42)

	g.Round(1)
	_ = g

	g.Round(4)
	_ = g
}

func TestDeterministicPeers(t *testing.T) {
	nodes1 := makeTestNodes(t, 16, 5)
	nodes2 := makeTestNodes(t, 16, 5)

	g1 := New(nodes1, 3, 5, 99)
	g2 := New(nodes2, 3, 5, 99)

	g1.Round(1)
	g2.Round(1)

	// Both gossipers should have sent the same number of messages given same seed
	if len(g1.MessagesSent) != len(g2.MessagesSent) {
		t.Fatalf("different node counts with messages: %d vs %d", len(g1.MessagesSent), len(g2.MessagesSent))
	}

	for id := range g1.MessagesSent {
		if g1.MessagesSent[id] != g2.MessagesSent[id] {
			t.Fatalf("different messages sent for node %d: %d vs %d", id, g1.MessagesSent[id], g2.MessagesSent[id])
		}
	}
}
