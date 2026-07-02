package fragment

import (
	"hash/fnv"
	"sort"
	"strings"

	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
)

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

func DistributeFlows(flows []dataset.Flow, numNodes int, k int, attackCategories []string) [][]dataset.Flow {
	partitions := make([][]dataset.Flow, numNodes)
	
	attackCatMap := make(map[string]bool)
	for _, c := range attackCategories {
		attackCatMap[strings.ToLower(c)] = true
	}

	rrCount := 0

	for _, flow := range flows {
		cat := strings.ToLower(flow.Category)
		isTargetAttack := flow.IsAttack && attackCatMap[cat]
		
		var assignedNode int
		if isTargetAttack {
			actualK := k
			if actualK > numNodes {
				actualK = numNodes
			}
			assignedNode = rrCount % actualK
			rrCount++
		} else {
			assignedNode = int(hashID(flow.ID) % uint32(numNodes))
		}
		
		partitions[assignedNode] = append(partitions[assignedNode], flow)
	}

	for i := 0; i < numNodes; i++ {
		sort.SliceStable(partitions[i], func(a, b int) bool {
			return partitions[i][a].Timestamp < partitions[i][b].Timestamp
		})
	}

	return partitions
}
