package fragment

import (
	"hash/fnv"
	"sort"
	"strings"

	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
)

// hashID returns a deterministic uint32 for an integer ID
func hashID(id int) uint32 {
	h := fnv.New32a()
	// Convert int to bytes simply
	b := []byte{
		byte(id >> 24),
		byte(id >> 16),
		byte(id >> 8),
		byte(id),
	}
	h.Write(b)
	return h.Sum32()
}

// DistributeFlows processes all flows, assigning normal traffic to nodes via IP hashing,
// and fragmenting targeted attack categories across k nodes using a round-robin strategy.
func DistributeFlows(flows []dataset.Flow, numNodes int, k int, attackCategories []string) [][]dataset.Flow {
	partitions := make([][]dataset.Flow, numNodes)
	
	attackCatMap := make(map[string]bool)
	for _, c := range attackCategories {
		attackCatMap[strings.ToLower(c)] = true
	}

	// Round-robin counter for targeted attack flows
	rrCount := 0

	for _, flow := range flows {
		cat := strings.ToLower(flow.Category)
		isTargetAttack := flow.IsAttack && attackCatMap[cat]
		
		var assignedNode int
		if isTargetAttack {
			// Fragmented across first k nodes round-robin (campaign splitting)
			assignedNode = rrCount % k
			rrCount++
		} else {
			// Deterministic assignment by pseudo-random Flow ID
			assignedNode = int(hashID(flow.ID) % uint32(numNodes))
		}
		
		partitions[assignedNode] = append(partitions[assignedNode], flow)
	}

	// Re-sort all partitions by timestamp to ensure chronological ingestion per node
	for i := 0; i < numNodes; i++ {
		sort.SliceStable(partitions[i], func(a, b int) bool {
			return partitions[i][a].Timestamp < partitions[i][b].Timestamp
		})
	}

	return partitions
}
