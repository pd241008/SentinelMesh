package node

import (
	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
	"github.com/pd241008/sentinelmesh/simulator/internal/scorer"
)

// Digest is the constant-size message exchanged between nodes via Gossip.
type Digest struct {
	NodeID   int
	Score    float64
	Category string
	Round    int // Discrete gossip round at which this digest was generated
}

// Node represents an independent IDS sensor in the mesh.
type Node struct {
	ID          int
	Scorer      *scorer.Scorer
	Flows       []dataset.Flow
	DigestCache map[int]Digest // PeerNodeID -> latest Digest
	CurrentFlow int
}

// New creates a new IDS node with the given flow partition.
func New(id int, flows []dataset.Flow, alpha float64) *Node {
	return &Node{
		ID:          id,
		Scorer:      scorer.New(alpha),
		Flows:       flows,
		DigestCache: make(map[int]Digest),
	}
}

// ProcessFlowsForRound simulates the node processing traffic for the current round.
// It generates a new digest to be gossiped for the current round, if there are flows left.
func (n *Node) ProcessFlowsForRound(round int) (*Digest, bool) {
	if n.CurrentFlow >= len(n.Flows) {
		return nil, false
	}

	flow := n.Flows[n.CurrentFlow]
	n.CurrentFlow++

	score := n.Scorer.ScoreFlow(flow)

	digest := Digest{
		NodeID:   n.ID,
		Score:    score,
		Category: flow.Category,
		Round:    round,
	}

	// Update own digest in cache
	n.DigestCache[n.ID] = digest

	return &digest, true
}

// ReceiveDigest updates the node's local digest cache with an incoming digest.
func (n *Node) ReceiveDigest(d Digest) {
	// Retain the most recent digest per source node (highest round)
	if existing, ok := n.DigestCache[d.NodeID]; !ok || d.Round > existing.Round {
		n.DigestCache[d.NodeID] = d
	}
}

// GetCache returns the current digest cache for quorum evaluation.
func (n *Node) GetCache() map[int]Digest {
	return n.DigestCache
}
