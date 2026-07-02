package node

import (
	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
	"github.com/pd241008/sentinelmesh/simulator/internal/scorer"
)

type Digest struct {
	NodeID   int
	Score    float64
	Category string
	Round    int // Discrete gossip round at which this digest was generated
}

type Node struct {
	ID          int
	Scorer      *scorer.Scorer
	Flows       []dataset.Flow
	DigestCache map[int][]Digest // PeerNodeID -> list of Digests within W
	CurrentFlow int
	Window      int
}

func New(id int, flows []dataset.Flow, alpha float64, thresh scorer.Thresholds, window int) *Node {
	return &Node{
		ID:          id,
		Scorer:      scorer.New(alpha, thresh),
		Flows:       flows,
		DigestCache: make(map[int][]Digest),
		Window:      window,
	}
}

func (n *Node) ProcessFlowsForRound(round int) (*Digest, bool) {
	if n.CurrentFlow >= len(n.Flows) {
		return nil, false
	}

	flow := n.Flows[n.CurrentFlow]
	n.CurrentFlow++

	score, guessedCategory, isAnomalous := n.Scorer.ScoreFlow(flow)

	n.PurgeStale(round)

	if !isAnomalous {
		return nil, true
	}

	digest := Digest{
		NodeID:   n.ID,
		Score:    score,
		Category: guessedCategory,
		Round:    round,
	}

	n.DigestCache[n.ID] = append(n.DigestCache[n.ID], digest)

	return &digest, true
}

func (n *Node) ReceiveDigest(d Digest) {
	for _, existing := range n.DigestCache[d.NodeID] {
		if existing.Round == d.Round {
			return // skip duplicate
		}
	}
	n.DigestCache[d.NodeID] = append(n.DigestCache[d.NodeID], d)
	n.PurgeStale(d.Round)
}

func (n *Node) PurgeStale(currentRound int) {
	for id, digests := range n.DigestCache {
		var fresh []Digest
		for _, d := range digests {
			if currentRound-d.Round <= n.Window {
				fresh = append(fresh, d)
			}
		}
		if len(fresh) > 0 {
			n.DigestCache[id] = fresh
		} else {
			delete(n.DigestCache, id)
		}
	}
}

func (n *Node) GetCache() map[int][]Digest {
	return n.DigestCache
}
