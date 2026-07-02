package gossip

import (
	"math/rand"

	"github.com/pd241008/sentinelmesh/simulator/internal/node"
)

type Gossiper struct {
	nodes  []*node.Node
	fanout int
	window int
	rng    *rand.Rand
}

func New(nodes []*node.Node, fanout int, window int, seed int64) *Gossiper {
	return &Gossiper{
		nodes:  nodes,
		fanout: fanout,
		window: window,
		rng:    rand.New(rand.NewSource(seed)),
	}
}

func (g *Gossiper) Round(round int) {
	for _, sender := range g.nodes {
		digest, ok := sender.ProcessFlowsForRound(round)
		if !ok {
			continue
		}
		if digest == nil {
			continue
		}

		peers := g.selectPeers(sender.ID)
		for _, peer := range peers {
			peer.ReceiveDigest(*digest)
		}
	}

	g.evictStale(round)
}

func (g *Gossiper) selectPeers(senderID int) []*node.Node {
	indices := g.rng.Perm(len(g.nodes))
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
