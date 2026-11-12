package quorum

import (
	"github.com/pd241008/sentinelmesh/simulator/internal/node"
)

// Evaluate applies the dual-threshold quorum rule from Eq. 2 of the paper:
//
//	Alert(c, t) = 1[ s̄_c(t) >= γ_high ] OR 1[ |S_c| >= q ]
//
// where S_c is the set of distinct source nodes reporting category c within window W.
// A high-confidence unilateral threshold (gammaHigh) triggers immediate escalation
// regardless of quorum, implementing the paper's dual-threshold rule.
func Evaluate(cache map[int][]node.Digest, quorum int, window int, currentRound int, gammaHigh map[string]float64) map[string][]int {
	categoryCount := make(map[string]int)

	for _, digests := range cache {
		// Dedup node's votes: a node gets maximum 1 vote per category within the window W.
		nodeCats := make(map[string]bool)
		for _, d := range digests {
			if currentRound-d.Round <= window {
				for _, cat := range d.Categories {
					nodeCats[cat] = true
				}
				// Backward compat: fall back to single Category field
				if d.Category != "" && d.Category != "normal" {
					nodeCats[d.Category] = true
				}
			}
		}

		// Increment global count only once per category per source node
		for cat := range nodeCats {
			categoryCount[cat]++
		}
	}

	alerts := make(map[string][]int)
	for cat, count := range categoryCount {
		quorumMet := count >= quorum

		// Dual-threshold: unilateral high-confidence bypass
		unilateralMet := false
		if gammaHigh != nil {
			if high, ok := gammaHigh[cat]; ok && high > 0 {
				// Check if any single node's score breaches the high threshold
				for _, digests := range cache {
					for _, d := range digests {
						if currentRound-d.Round <= window {
							isCat := false
							for _, c := range d.Categories {
								if c == cat {
									isCat = true
									break
								}
							}
							if !isCat && d.Category == cat {
								isCat = true
							}
							if isCat && d.Score >= high {
								unilateralMet = true
								break
							}
						}
					}
					if unilateralMet {
						break
					}
				}
			}
		}

		if quorumMet || unilateralMet {
			var nodeIDs []int
			for srcID, digests := range cache {
				for _, d := range digests {
					if currentRound-d.Round <= window {
						isCat := false
						for _, c := range d.Categories {
							if c == cat {
								isCat = true
								break
							}
						}
						if !isCat && d.Category == cat {
							isCat = true
						}
						if isCat {
							nodeIDs = append(nodeIDs, srcID)
							break
						}
					}
				}
			}
			alerts[cat] = nodeIDs
		}
	}

	return alerts
}

// EvaluateLegacy is the original quorum-only evaluation without dual-threshold.
// It is kept for backward compatibility with callers that don't pass gammaHigh.
func EvaluateLegacy(cache map[int][]node.Digest, quorum int, window int, currentRound int) map[string][]int {
	return Evaluate(cache, quorum, window, currentRound, nil)
}
