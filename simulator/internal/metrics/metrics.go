package metrics

import (
	"math"

	"github.com/pd241008/sentinelmesh/simulator/internal/baseline"
	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
	"github.com/pd241008/sentinelmesh/simulator/internal/node"
)

type Result struct {
	FlowReconRecall   float64
	FlowDoSRecall     float64
	WindowReconRecall float64
	WindowDoSRecall   float64
	ReconFPR          float64
	DoSFPR            float64
	Bandwidth         int
	AvgLatency        float64
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Compute(nodes []*node.Node, allFlows []dataset.Flow, alerts baseline.AlertTimeline, totalDigests int, window int) Result {
	firstAlertRound := make(map[string]int)

	maxRound := 0
	for _, n := range nodes {
		if len(n.Flows) > maxRound {
			maxRound = len(n.Flows)
		}
	}
	for r := range alerts {
		if r > maxRound {
			maxRound = r
		}
	}

	for round := 1; round <= maxRound; round++ {
		if cats, ok := alerts[round]; ok {
			for cat := range cats {
				if _, exists := firstAlertRound[cat]; !exists {
					firstAlertRound[cat] = round
				}
			}
		}
	}

	// Flow-level recall [r, r+W]
	totalReconFlows := 0
	detectedReconFlows := 0
	totalDosFlows := 0
	detectedDosFlows := 0

	for r := 1; r <= maxRound; r++ {
		for _, n := range nodes {
			if r-1 < len(n.Flows) {
				flow := n.Flows[r-1]
				if flow.Category == "reconnaissance" || flow.Category == "dos" {
					detected := false
					for rr := r; rr <= min(r+window, maxRound); rr++ {
						if alerts[rr] != nil && alerts[rr][flow.Category] {
							detected = true
							break
						}
					}
					
					if flow.Category == "reconnaissance" {
						totalReconFlows++
						if detected {
							detectedReconFlows++
						}
					} else if flow.Category == "dos" {
						totalDosFlows++
						if detected {
							detectedDosFlows++
						}
					}
				}
			}
		}
	}

	var flowReconRecall float64
	if totalReconFlows > 0 {
		flowReconRecall = float64(detectedReconFlows) / float64(totalReconFlows)
	}
	var flowDosRecall float64
	if totalDosFlows > 0 {
		flowDosRecall = float64(detectedDosFlows) / float64(totalDosFlows)
	}

	// Window-level Recall & FPR
	totalReconActiveWindows := 0
	truePosReconWindows := 0
	totalReconNormalWindows := 0
	falsePosReconWindows := 0

	totalDoSActiveWindows := 0
	truePosDoSWindows := 0
	totalDoSNormalWindows := 0
	falsePosDoSWindows := 0

	for r := 1; r <= maxRound; r++ {
		hasReconFlow := false
		hasDoSFlow := false
		for _, n := range nodes {
			if r-1 < len(n.Flows) {
				cat := n.Flows[r-1].Category
				if cat == "reconnaissance" {
					hasReconFlow = true
				}
				if cat == "dos" {
					hasDoSFlow = true
				}
			}
		}
		
		if hasReconFlow {
			totalReconActiveWindows++
			if alerts[r] != nil && alerts[r]["reconnaissance"] {
				truePosReconWindows++
			}
		} else {
			totalReconNormalWindows++
			if alerts[r] != nil && alerts[r]["reconnaissance"] {
				falsePosReconWindows++
			}
		}
		
		if hasDoSFlow {
			totalDoSActiveWindows++
			if alerts[r] != nil && alerts[r]["dos"] {
				truePosDoSWindows++
			}
		} else {
			totalDoSNormalWindows++
			if alerts[r] != nil && alerts[r]["dos"] {
				falsePosDoSWindows++
			}
		}
	}
	
	var windowReconRecall float64
	if totalReconActiveWindows > 0 {
		windowReconRecall = float64(truePosReconWindows) / float64(totalReconActiveWindows)
	}
	var windowDosRecall float64
	if totalDoSActiveWindows > 0 {
		windowDosRecall = float64(truePosDoSWindows) / float64(totalDoSActiveWindows)
	}

	var reconFPR float64
	if totalReconNormalWindows > 0 {
		reconFPR = float64(falsePosReconWindows) / float64(totalReconNormalWindows)
	}
	var dosFPR float64
	if totalDoSNormalWindows > 0 {
		dosFPR = float64(falsePosDoSWindows) / float64(totalDoSNormalWindows)
	}

	var avgLatency float64
	if len(firstAlertRound) > 0 {
		avgLatency = float64(maxRound) / float64(len(firstAlertRound))
	}

	return Result{
		FlowReconRecall:   math.Round(flowReconRecall*10000) / 10000,
		FlowDoSRecall:     math.Round(flowDosRecall*10000) / 10000,
		WindowReconRecall: math.Round(windowReconRecall*10000) / 10000,
		WindowDoSRecall:   math.Round(windowDosRecall*10000) / 10000,
		ReconFPR:          math.Round(reconFPR*10000) / 10000,
		DoSFPR:            math.Round(dosFPR*10000) / 10000,
		Bandwidth:         totalDigests,
		AvgLatency:        math.Round(avgLatency*100) / 100,
	}
}
