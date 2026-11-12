package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pd241008/sentinelmesh/simulator/internal/baseline"
	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
	"github.com/pd241008/sentinelmesh/simulator/internal/fragment"
	"github.com/pd241008/sentinelmesh/simulator/internal/gossip"
	"github.com/pd241008/sentinelmesh/simulator/internal/metrics"
	"github.com/pd241008/sentinelmesh/simulator/internal/node"
	"github.com/pd241008/sentinelmesh/simulator/internal/quorum"
	"github.com/pd241008/sentinelmesh/simulator/internal/scorer"
	"github.com/pd241008/sentinelmesh/simulator/internal/sweep"
)

var testdataPath = filepath.Join("..", "testdata", "testdata.csv")

func defaultThresholds() scorer.Thresholds {
	return scorer.Thresholds{
		Recon: 1.66,
		DoS:   0.82,
		Ewma:  0.89,
	}
}

func TestFullPipeline(t *testing.T) {
	allFlows, err := dataset.LoadCSV(testdataPath)
	if err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}
	if len(allFlows) != 15 {
		t.Fatalf("expected 15 flows, got %d", len(allFlows))
	}

	attackCategories := []string{"fuzzers", "reconnaissance", "dos", "exploits", "generic", "analysis", "backdoor"}
	partitions, _ := fragment.DistributeFlows(allFlows, 4, 2, attackCategories, false, 0)
	if len(partitions) != 4 {
		t.Fatalf("expected 4 partitions, got %d", len(partitions))
	}

	totalRounds := 0
	for _, p := range partitions {
		if len(p) > totalRounds {
			totalRounds = len(p)
		}
	}
	t.Logf("totalRounds: %d", totalRounds)

	nodes := make([]*node.Node, 4)
	for i := 0; i < 4; i++ {
		nodes[i] = node.New(i, partitions[i], 0.3, defaultThresholds(), 5)
	}

	g := gossip.New(nodes, 2, 5, 42)
	alerts := make(map[int]map[string][]int)

	roundAlerts := 0
	for round := 1; round <= totalRounds; round++ {
		g.Round(round)
		for _, n := range nodes {
			cats := quorum.Evaluate(n.GetCache(), 2, 5, round, nil)
			for cat, corrobs := range cats {
				if alerts[round] == nil {
					alerts[round] = make(map[string][]int)
				}
				alerts[round][cat] = corrobs
				roundAlerts += len(corrobs)
			}
		}
	}

	t.Logf("Total round alerts: %d", roundAlerts)

	// Run metrics computation
	result := metrics.Compute(nodes, allFlows, alerts, nil, nil, nil, totalRounds*4*2, 5, nil, totalRounds, 4)
	if result.FlowDoSRecall < 0 || result.FlowDoSRecall > 1 {
		t.Fatalf("invalid recall: %f", result.FlowDoSRecall)
	}
	t.Logf("DoS recall: %.4f, Bandwidth: %.4f KB/s, Latency: %.2f rounds", result.FlowDoSRecall, result.BandwidthKBps, result.AvgLatency)
}

func TestSweepIntegration(t *testing.T) {
	allFlows, err := dataset.LoadCSV(testdataPath)
	if err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}

	cfg := &sweep.Config{}
	cfg.Sweep.N = []int{4}
	cfg.Sweep.F = []int{2}
	cfg.Sweep.K = []int{2}
	cfg.Sweep.W = 3

	outputDir := t.TempDir()
	err = sweep.Run(cfg, allFlows, 0.3, 0.3, 42, outputDir, false, false, 0)
	if err != nil {
		t.Fatalf("sweep.Run: %v", err)
	}

	csvPath := filepath.Join(outputDir, "sweep_results.csv")
	data, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("CSV is empty")
	}
	t.Logf("CSV output:\n%s", string(data))
}

func TestEndToEndWithBaselines(t *testing.T) {
	allFlows, err := dataset.LoadCSV(testdataPath)
	if err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}

	attackCategories := []string{"fuzzers", "reconnaissance", "dos", "exploits", "generic", "analysis", "backdoor"}
	partitions, _ := fragment.DistributeFlows(allFlows, 4, 2, attackCategories, false, 0)

	totalRounds := 0
	for _, p := range partitions {
		if len(p) > totalRounds {
			totalRounds = len(p)
		}
	}

	makeNodes := func() []*node.Node {
		nodes := make([]*node.Node, 4)
		for i := 0; i < 4; i++ {
			nodes[i] = node.New(i, partitions[i], 0.3, defaultThresholds(), 5)
		}
		return nodes
	}

	gossipNodes := makeNodes()
	g := gossip.New(gossipNodes, 2, 5, 42)
	alerts := make(map[int]map[string][]int)
	for round := 1; round <= totalRounds; round++ {
		g.Round(round)
		for _, n := range gossipNodes {
			cats := quorum.Evaluate(n.GetCache(), 2, 5, round, nil)
			for cat, corrobs := range cats {
				if alerts[round] == nil {
					alerts[round] = make(map[string][]int)
				}
				alerts[round][cat] = corrobs
			}
		}
	}

	indepNodes := makeNodes()
	indepResult := baseline.RunIndependent(indepNodes, 5, totalRounds, 1)

	centNodes := makeNodes()
	centResult := baseline.RunCentralized(centNodes, 5, totalRounds, 1)

	gResult := metrics.Compute(gossipNodes, allFlows, alerts, nil, nil, nil, totalRounds*4*2, 5, nil, totalRounds, 4)
	iResult := metrics.Compute(indepNodes, allFlows, indepResult.Alerts, nil, nil, nil, indepResult.TotalDigests, 5, nil, totalRounds, 4)
	cResult := metrics.Compute(centNodes, allFlows, centResult.Alerts, nil, nil, nil, centResult.TotalDigests, 5, nil, totalRounds, 4)

	t.Logf("Gossip:      DoS recall=%.4f  BW=%.4f KB/s  Lat=%.2f rounds", gResult.FlowDoSRecall, gResult.BandwidthKBps, gResult.AvgLatency)
	t.Logf("Independent: DoS recall=%.4f  BW=%.4f KB/s  Lat=%.2f rounds", iResult.FlowDoSRecall, iResult.BandwidthKBps, iResult.AvgLatency)
	t.Logf("Centralized: DoS recall=%.4f  BW=%.4f KB/s  Lat=%.2f rounds", cResult.FlowDoSRecall, cResult.BandwidthKBps, cResult.AvgLatency)
}
