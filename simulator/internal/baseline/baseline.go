package baseline

import (
	"github.com/pd241008/sentinelmesh/simulator/internal/node"
	"github.com/pd241008/sentinelmesh/simulator/internal/quorum"
)

type AlertTimeline map[int]map[string]bool

type BaselineResult struct {
	Alerts       AlertTimeline
	TotalDigests int
}

func addAlert(alerts AlertTimeline, round int, category string) {
	if alerts[round] == nil {
		alerts[round] = make(map[string]bool)
	}
	alerts[round][category] = true
}

func RunIndependent(nodes []*node.Node, window int, totalRounds int, quorumThreshold int) BaselineResult {
	alerts := make(AlertTimeline)
	totalDigests := 0
	for round := 1; round <= totalRounds; round++ {
		for _, n := range nodes {
			d, ok := n.ProcessFlowsForRound(round)
			if !ok {
				continue
			}
			if d != nil {
				totalDigests++
			}
		}
		for _, n := range nodes {
			for _, cat := range quorum.Evaluate(n.GetCache(), 1, window, round) {
				addAlert(alerts, round, cat)
			}
		}
	}
	return BaselineResult{Alerts: alerts, TotalDigests: totalDigests}
}

func RunCentralized(nodes []*node.Node, window int, totalRounds int, quorumThreshold int) BaselineResult {
	centralCache := make(map[int][]node.Digest)
	alerts := make(AlertTimeline)
	totalDigests := 0

	for round := 1; round <= totalRounds; round++ {
		for _, n := range nodes {
			d, ok := n.ProcessFlowsForRound(round)
			if !ok {
				continue
			}
			if d != nil {
				centralCache[n.ID] = append(centralCache[n.ID], *d)
				totalDigests++
			}
		}
		for _, cat := range quorum.Evaluate(centralCache, quorumThreshold, window, round) {
			addAlert(alerts, round, cat)
		}
		for id, digests := range centralCache {
			var fresh []node.Digest
			for _, d := range digests {
				if round-d.Round <= window {
					fresh = append(fresh, d)
				}
			}
			if len(fresh) > 0 {
				centralCache[id] = fresh
			} else {
				delete(centralCache, id)
			}
		}
	}
	return BaselineResult{Alerts: alerts, TotalDigests: totalDigests}
}
