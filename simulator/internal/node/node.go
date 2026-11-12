package node

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"

	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
	"github.com/pd241008/sentinelmesh/simulator/internal/scorer"
)

type Digest struct {
	NodeID     int
	Score      float64
	Category   string   // Primary (highest-scoring) category for backward compatibility
	Categories []string // All breached categories (paper's C_i(t) multi-category set)
	Round      int      // Discrete gossip round at which this digest was generated
	Signature  []byte   // 16-byte truncated HMAC-SHA256 origin signature
}

type Node struct {
	ID          int
	Scorer      *scorer.Scorer
	Flows       []dataset.Flow
	DigestCache map[int][]Digest // PeerNodeID -> list of Digests within W
	CurrentFlow int
	Window      int
	signingKey  []byte
}

func New(id int, flows []dataset.Flow, alpha float64, thresh scorer.Thresholds, window int) *Node {
	return &Node{
		ID:          id,
		Scorer:      scorer.New(alpha, thresh),
		Flows:       flows,
		DigestCache: make(map[int][]Digest),
		Window:      window,
		signingKey:  make([]byte, 16),
	}
}

func (n *Node) ProcessFlowsForRound(round int) (*Digest, bool) {
	if n.CurrentFlow >= len(n.Flows) {
		return nil, false
	}

	flow := n.Flows[n.CurrentFlow]
	n.CurrentFlow++

	score, categories, primaryCategory, isAnomalous := n.Scorer.ScoreFlow(flow)

	n.PurgeStale(round)

	if !isAnomalous {
		return nil, true
	}

	// Construct 32-byte wire-format digest:
	//   2B node_id | 2B category bitmask | 6B float16 score tuple | 6B timestamp | 16B HMAC-SHA256 tag
	sig := n.signDigest(round, categories)

	digest := Digest{
		NodeID:     n.ID,
		Score:      score,
		Category:   primaryCategory,
		Categories: categories,
		Round:      round,
		Signature:  sig,
	}

	n.DigestCache[n.ID] = append(n.DigestCache[n.ID], digest)

	return &digest, true
}

func (n *Node) signDigest(round int, categories []string) []byte {
	mac := hmac.New(sha256.New, n.signingKey)
	var buf [16]byte
	binary.LittleEndian.PutUint16(buf[0:2], uint16(n.ID))
	binary.LittleEndian.PutUint16(buf[2:4], categoryBitmask(categories))
	binary.LittleEndian.PutUint64(buf[4:12], uint64(round))
	mac.Write(buf[:12])
	mac.Write([]byte{byte(round)})
	return mac.Sum(nil)[:16]
}

func categoryBitmask(categories []string) uint16 {
	var mask uint16
	for _, c := range categories {
		switch c {
		case "dos":
			mask |= 1 << 0
		case "reconnaissance":
			mask |= 1 << 1
		case "generic":
			mask |= 1 << 2
		}
	}
	return mask
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

func (n *Node) ClearCache() {
	n.DigestCache = make(map[int][]Digest)
}
