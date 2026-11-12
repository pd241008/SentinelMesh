package baseline

import (
	"testing"

	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
	"github.com/pd241008/sentinelmesh/simulator/internal/node"
	"github.com/pd241008/sentinelmesh/simulator/internal/scorer"
)

func defaultThresholds() scorer.Thresholds {
	return scorer.Thresholds{
		Recon: 1.66,
		DoS:   0.82,
		Ewma:  0.89,
	}
}

func makeBaselineNodes(t *testing.T, n int, flowsPerNode int, alpha float64) []*node.Node {
	t.Helper()
	nodes := make([]*node.Node, n)
	for i := 0; i < n; i++ {
		flows := make([]dataset.Flow, flowsPerNode)
		for j := 0; j < flowsPerNode; j++ {
			flows[j] = dataset.Flow{
				ID:        i*flowsPerNode + j + 1,
				Category:  "normal",
				Sbytes:    100,
				Dbytes:    200,
				Spkts:     2,
				Dpkts:     4,
				Rate:      10.0,
				Timestamp: float64(i*flowsPerNode + j + 1),
			}
		}
		nodes[i] = node.New(i, flows, alpha, defaultThresholds(), 5)
	}
	return nodes
}

func TestRunIndependent(t *testing.T) {
	nodes := makeBaselineNodes(t, 4, 10, 0.3)
	result := RunIndependent(nodes, 5, 5, 1)
	if result.Alerts == nil {
		t.Fatal("expected non-nil alerts")
	}
	if result.TotalDigests < 0 {
		t.Fatalf("expected non-negative total digests, got %d", result.TotalDigests)
	}
}

func TestRunIndependentNoAlerts(t *testing.T) {
	nodes := makeBaselineNodes(t, 4, 5, 0.3)
	result := RunIndependent(nodes, 5, 5, 99)
	for round, cats := range result.Alerts {
		t.Errorf("unexpected alert at round %d: %v", round, cats)
	}
}

func TestRunCentralized(t *testing.T) {
	nodes := makeBaselineNodes(t, 4, 10, 0.3)
	result := RunCentralized(nodes, 5, 5, 1)
	if result.Alerts == nil {
		t.Fatal("expected non-nil alerts")
	}
	if result.TotalDigests < 0 {
		t.Fatalf("expected non-negative total digests, got %d", result.TotalDigests)
	}
}

func TestRunCentralizedNoAlerts(t *testing.T) {
	nodes := makeBaselineNodes(t, 4, 5, 0.3)
	result := RunCentralized(nodes, 5, 5, 99)
	for round, cats := range result.Alerts {
		t.Errorf("unexpected alert at round %d: %v", round, cats)
	}
}
