package baseline

import (
	"github.com/pd241008/sentinelmesh/simulator/internal/node"
	"github.com/pd241008/sentinelmesh/simulator/internal/quorum"
)

type AlertTimeline map[int]map[string][]int

type BaselineResult struct {
	Alerts       AlertTimeline
	TotalDigests int
}

func addAlert(alerts AlertTimeline, round int, category string, corrobs []int) {
	if alerts[round] == nil {
		alerts[round] = make(map[string][]int)
	}
	alerts[round][category] = corrobs
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
			for cat, corrobs := range quorum.Evaluate(n.GetCache(), quorumThreshold, window, round, nil) {
				addAlert(alerts, round, cat, corrobs)
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
		for cat, corrobs := range quorum.Evaluate(centralCache, quorumThreshold, window, round, nil) {
			addAlert(alerts, round, cat, corrobs)
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
