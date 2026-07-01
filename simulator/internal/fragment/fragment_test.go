package fragment

import (
	"testing"

	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
)

func makeTestFlows() []dataset.Flow {
	return []dataset.Flow{
		{ID: 1, Category: "normal", IsAttack: false},
		{ID: 2, Category: "normal", IsAttack: false},
		{ID: 3, Category: "dos", IsAttack: true},
		{ID: 4, Category: "normal", IsAttack: false},
		{ID: 5, Category: "fuzzers", IsAttack: true},
		{ID: 6, Category: "normal", IsAttack: false},
		{ID: 7, Category: "exploits", IsAttack: true},
		{ID: 8, Category: "dos", IsAttack: true},
		{ID: 9, Category: "normal", IsAttack: false},
		{ID: 10, Category: "fuzzers", IsAttack: true},
	}
}

func TestDistributeFlows(t *testing.T) {
	flows := makeTestFlows()
	attackCats := []string{"dos", "fuzzers", "exploits"}

	partitions := DistributeFlows(flows, 4, 2, attackCats)

	if len(partitions) != 4 {
		t.Fatalf("expected 4 partitions, got %d", len(partitions))
	}

	total := 0
	for i, p := range partitions {
		total += len(p)
		t.Logf("partition[%d]: %d flows", i, len(p))
	}
	if total != len(flows) {
		t.Fatalf("expected %d total flows, got %d", len(flows), total)
	}
}

func TestDistributeFlowsFragmentation(t *testing.T) {
	flows := makeTestFlows()
	attackCats := []string{"dos", "fuzzers", "exploits"}

	partitions := DistributeFlows(flows, 8, 3, attackCats)

	dosIn0 := false
	dosIn1 := false
	for _, f := range partitions[0] {
		if f.Category == "dos" {
			dosIn0 = true
		}
	}
	for _, f := range partitions[1] {
		if f.Category == "dos" {
			dosIn1 = true
		}
	}

	hasFrags := dosIn0 || dosIn1
	if !hasFrags {
		t.Error("expected fragmented attack flows in first k partitions")
	}
}

func TestDistributeFlowsPreservesCount(t *testing.T) {
	flows := makeTestFlows()
	attackCats := []string{"dos", "fuzzers", "exploits"}

	partitions := DistributeFlows(flows, 4, 2, attackCats)

	attackCount := 0
	for _, p := range partitions {
		for _, f := range p {
			if f.IsAttack {
				attackCount++
			}
		}
	}
	if attackCount != 5 {
		t.Fatalf("expected 5 attack flows total, got %d", attackCount)
	}
}
