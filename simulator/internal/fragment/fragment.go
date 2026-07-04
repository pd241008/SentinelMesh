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

func DistributeFlows(flows []dataset.Flow, numNodes int, k int, attackCategories []string, clustered bool) ([][]dataset.Flow, []Campaign) {
	return distributeFlowsInternal(flows, nil, numNodes, k, attackCategories, false, clustered)
}

func DistributeFlowsControl(flows []dataset.Flow, normalPool []dataset.Flow, numNodes int, k int, attackCategories []string, clustered bool) ([][]dataset.Flow, []Campaign) {
	return distributeFlowsInternal(flows, normalPool, numNodes, k, attackCategories, true, clustered)
}

func distributeFlowsInternal(flows []dataset.Flow, normalPool []dataset.Flow, numNodes int, k int, attackCategories []string, isControl bool, clustered bool) ([][]dataset.Flow, []Campaign) {
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
				if rrCount % 5 != 0 {
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

			if isControl {
				// Replace the flow with a normal flow from the pool
				if len(normalPool) > 0 {
					replacementFlow := normalPool[normalPoolIdx%len(normalPool)]
					normalPoolIdx++
					// Assign using hash based on original ID to ensure deterministic but unstructured scatter
					assignedNode = int(hashID(flow.ID) % uint32(numNodes))
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

	return partitions, campaigns
}
