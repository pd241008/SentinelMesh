package metrics

import (
	"testing"

	"github.com/pd241008/sentinelmesh/simulator/internal/baseline"
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

func TestComputeFullRecall(t *testing.T) {
	flows := []dataset.Flow{
		{ID: 1, Category: "dos", IsAttack: true, Timestamp: 1.0},
		{ID: 2, Category: "dos", IsAttack: true, Timestamp: 2.0},
	}
	nodes := []*node.Node{
		node.New(0, flows[:1], 0.3, defaultThresholds(), 10),
		node.New(1, flows[1:], 0.3, defaultThresholds(), 10),
	}
	alerts := baseline.AlertTimeline{
		1: {"dos": {0}},
		2: {"dos": {1}},
	}
	result := Compute(nodes, flows, alerts, nil, nil, nil, 2, 10, nil, 2, 2)
	if result.FlowDoSRecall != 1.0 {
		t.Fatalf("expected recall 1.0, got %f", result.FlowDoSRecall)
	}
}

func TestComputeZeroRecall(t *testing.T) {
	flows := []dataset.Flow{
		{ID: 1, Category: "dos", IsAttack: true, Timestamp: 1.0},
		{ID: 2, Category: "fuzzers", IsAttack: true, Timestamp: 2.0},
	}
	nodes := []*node.Node{
		node.New(0, flows[:1], 0.3, defaultThresholds(), 10),
		node.New(1, flows[1:], 0.3, defaultThresholds(), 10),
	}
	alerts := baseline.AlertTimeline{}
	result := Compute(nodes, flows, alerts, nil, nil, nil, 0, 10, nil, 2, 2)
	if result.FlowDoSRecall != 0.0 {
		t.Fatalf("expected recall 0.0, got %f", result.FlowDoSRecall)
	}
	if result.FlowReconRecall != 0.0 {
		t.Fatalf("expected recon recall 0.0, got %f", result.FlowReconRecall)
	}
}

func TestComputePartialRecall(t *testing.T) {
	flows := []dataset.Flow{
		{ID: 1, Category: "dos", IsAttack: true, Timestamp: 1.0},
		{ID: 2, Category: "dos", IsAttack: true, Timestamp: 2.0},
		{ID: 3, Category: "fuzzers", IsAttack: true, Timestamp: 3.0},
		{ID: 4, Category: "fuzzers", IsAttack: true, Timestamp: 4.0},
	}
	nodes := []*node.Node{
		node.New(0, flows[:2], 0.3, defaultThresholds(), 10),
		node.New(1, flows[2:], 0.3, defaultThresholds(), 10),
	}
	alerts := baseline.AlertTimeline{
		1: {"dos": {0}},
		2: {"dos": {1}},
	}
	result := Compute(nodes, flows, alerts, nil, nil, nil, 2, 10, nil, 2, 2)
	if result.FlowDoSRecall != 1.0 {
		t.Fatalf("expected dos recall 1.0, got %f", result.FlowDoSRecall)
	}
}

func TestComputeNoAttackFlows(t *testing.T) {
	flows := []dataset.Flow{
		{ID: 1, Category: "normal", IsAttack: false, Timestamp: 1.0},
	}
	nodes := []*node.Node{
		node.New(0, flows, 0.3, defaultThresholds(), 10),
	}
	alerts := baseline.AlertTimeline{}
	result := Compute(nodes, flows, alerts, nil, nil, nil, 0, 10, nil, 1, 1)
	if result.FlowDoSRecall != 0.0 {
		t.Fatalf("expected recall 0.0 for no attacks, got %f", result.FlowDoSRecall)
	}
}

func TestComputeBandwidth(t *testing.T) {
	flows := []dataset.Flow{}
	nodes := []*node.Node{
		node.New(0, nil, 0.3, defaultThresholds(), 10),
	}
	alerts := baseline.AlertTimeline{}
	// 100 digests * 32 bytes / 1 node / 1 round / 1024 = 3.125 KB/s
	result := Compute(nodes, flows, alerts, nil, nil, nil, 100, 1, nil, 1, 1)
	if result.BandwidthKBps != 3.125 {
		t.Fatalf("expected bandwidth 3.125 KB/s, got %f", result.BandwidthKBps)
	}
}

func TestComputeCorrectedRecall(t *testing.T) {
	flows := []dataset.Flow{
		{ID: 1, Category: "dos", IsAttack: true, Timestamp: 1.0},
		{ID: 2, Category: "dos", IsAttack: true, Timestamp: 2.0},
	}
	nodes := []*node.Node{
		node.New(0, flows, 0.3, defaultThresholds(), 10),
	}
	tAlerts := baseline.AlertTimeline{
		1: {"dos": {0}},
		2: {"dos": {0}},
	}
	cAlerts := baseline.AlertTimeline{
		1: {"dos": {0}}, // control matches treatment at round 1
	}
	result := Compute(nodes, flows, tAlerts, nil, cAlerts, nil, 2, 10, nil, 2, 1)
	// Corrected recall: flow 2 at round 2 has no matching control → corrected detected
	// Flow 1 at round 1: control matches → exact match, not corrected
	if result.CorrectedFlowDoSRecall != 0.5 {
		t.Fatalf("expected corrected recall 0.5, got %f", result.CorrectedFlowDoSRecall)
	}
}
