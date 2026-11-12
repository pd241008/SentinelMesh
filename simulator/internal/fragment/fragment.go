package fragment

import (
	"hash/fnv"
	"sort"
	"strings"

	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
)

type Campaign struct {
	Category string
	FlowIDs  []int
}

func hashID(id int) uint32 {
	h := fnv.New32a()
	b := []byte{
		byte(id >> 24),
		byte(id >> 16),
		byte(id >> 8),
		byte(id),
	}
	h.Write(b)
	return h.Sum32()
}

func DistributeFlows(flows []dataset.Flow, numNodes int, k int, attackCategories []string, clustered bool, staggerRounds int) ([][]dataset.Flow, []Campaign) {
	return distributeFlowsInternal(flows, nil, numNodes, k, attackCategories, "", false, clustered, staggerRounds)
}

func DistributeFlowsControl(flows []dataset.Flow, normalPool []dataset.Flow, numNodes int, k int, attackCategories []string, targetCategory string, clustered bool, staggerRounds int) ([][]dataset.Flow, []Campaign) {
	return distributeFlowsInternal(flows, normalPool, numNodes, k, attackCategories, targetCategory, true, clustered, staggerRounds)
}

func distributeFlowsInternal(flows []dataset.Flow, normalPool []dataset.Flow, numNodes int, k int, attackCategories []string, targetCategory string, isControl bool, clustered bool, staggerRounds int) ([][]dataset.Flow, []Campaign) {
	partitions := make([][]dataset.Flow, numNodes)

	attackCatMap := make(map[string]bool)
	for _, c := range attackCategories {
		attackCatMap[strings.ToLower(c)] = true
	}

	rrCount := 0
	normalPoolIdx := 0
	var campaigns []Campaign
	var currentCampaign *Campaign

	for _, flow := range flows {
		cat := strings.ToLower(flow.Category)
		isTargetAttack := flow.IsAttack && attackCatMap[cat]

		var assignedNode int
		if isTargetAttack {
			actualK := k
			if actualK > numNodes {
				actualK = numNodes
			}
			if clustered {
				if rrCount%5 != 0 {
					assignedNode = 0
				} else {
					if actualK > 1 {
						assignedNode = 1 + ((rrCount / 5) % (actualK - 1))
					} else {
						assignedNode = 0
					}
				}
			} else {
				assignedNode = rrCount % actualK
			}
			rrCount++

			if currentCampaign == nil || currentCampaign.Category != cat {
				if currentCampaign != nil {
					campaigns = append(campaigns, *currentCampaign)
				}
				currentCampaign = &Campaign{Category: cat}
			}
			currentCampaign.FlowIDs = append(currentCampaign.FlowIDs, flow.ID)

			if isControl && cat == targetCategory {
				// Replace the flow with a normal flow from the pool
				if len(normalPool) > 0 {
					replacementFlow := normalPool[normalPoolIdx%len(normalPool)]
					normalPoolIdx++
					// Crucial: preserve original ID and Timestamp so metrics matching and sorting still align
					replacementFlow.ID = flow.ID
					replacementFlow.Timestamp = flow.Timestamp
					flow = replacementFlow
				}
			}
		} else {
			assignedNode = int(hashID(flow.ID) % uint32(numNodes))
		}

		partitions[assignedNode] = append(partitions[assignedNode], flow)
	}
	if currentCampaign != nil {
		campaigns = append(campaigns, *currentCampaign)
	}

	for i := 0; i < numNodes; i++ {
		sort.SliceStable(partitions[i], func(a, b int) bool {
			return partitions[i][a].Timestamp < partitions[i][b].Timestamp
		})
	}

	if staggerRounds > 0 {
		partitions = applyStagger(partitions, campaigns, staggerRounds, numNodes)
	}

	return partitions, campaigns
}

func applyStagger(partitions [][]dataset.Flow, campaigns []Campaign, staggerRounds int, numNodes int) [][]dataset.Flow {
	for _, camp := range campaigns {
		if len(camp.FlowIDs) <= 1 {
			continue
		}
		for i, flowID := range camp.FlowIDs {
			// Rounding to nearest integer for better distribution
			delay := int(float64(i*staggerRounds)/float64(len(camp.FlowIDs)-1) + 0.5)
			if delay == 0 {
				continue
			}

			for n := 0; n < numNodes; n++ {
				found := false
				for pIdx, f := range partitions[n] {
					if f.ID == flowID {
						part := partitions[n]
						targetIdx := pIdx + delay

						if targetIdx >= len(part) {
							padding := targetIdx - len(part) + 1
							for p := 0; p < padding; p++ {
								part = append(part, dataset.Flow{Category: "normal", Timestamp: f.Timestamp + float64(p+1)})
							}
						}

						theFlow := part[pIdx]
						// Remove from pIdx
						part = append(part[:pIdx], part[pIdx+1:]...)
						// Insert at targetIdx
						part = append(part[:targetIdx], append([]dataset.Flow{theFlow}, part[targetIdx:]...)...)

						partitions[n] = part
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}
	}
	return partitions
}
