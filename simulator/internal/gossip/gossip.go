package gossip

import (
	"math/rand"

	"github.com/pd241008/sentinelmesh/simulator/internal/node"
)

type Gossiper struct {
	nodes        []*node.Node
	fanout       int
	window       int
	rng          *rand.Rand
	MessagesSent map[int]int
}

func New(nodes []*node.Node, fanout int, window int, seed int64) *Gossiper {
	return &Gossiper{
		nodes:        nodes,
		fanout:       fanout,
		window:       window,
		rng:          rand.New(rand.NewSource(seed)),
		MessagesSent: make(map[int]int),
	}
}

func (g *Gossiper) Round(round int) {
	for _, sender := range g.nodes {
		// UNCONDITIONALLY advance RNG state so Treatment and Control remain synchronized
		indices := g.rng.Perm(len(g.nodes))

		digest, ok := sender.ProcessFlowsForRound(round)
		if !ok || digest == nil {
			continue
		}

		peers := g.selectPeersFromIndices(sender.ID, indices)
		for _, peer := range peers {
			peer.ReceiveDigest(*digest)
		}
		g.MessagesSent[sender.ID] += len(peers)
	}

	g.evictStale(round)
}

func (g *Gossiper) selectPeersFromIndices(senderID int, indices []int) []*node.Node {
	var selected []*node.Node
	for _, idx := range indices {
		if g.nodes[idx].ID != senderID {
			selected = append(selected, g.nodes[idx])
			if len(selected) == g.fanout {
				break
			}
		}
	}
	return selected
}

func (g *Gossiper) evictStale(round int) {
	for _, n := range g.nodes {
		n.PurgeStale(round)
	}
}
