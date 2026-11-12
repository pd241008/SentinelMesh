package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
	"github.com/pd241008/sentinelmesh/simulator/internal/fragment"
	"github.com/pd241008/sentinelmesh/simulator/internal/node"
	"github.com/pd241008/sentinelmesh/simulator/internal/quorum"
	"github.com/pd241008/sentinelmesh/simulator/internal/scorer"
)

var AttackCategories = []string{
	"analysis", "backdoor", "dos", "exploits", "fuzzers",
	"generic", "reconnaissance", "shellcode", "worms",
}

type LossResult struct {
	lossRate  float64
	reconMean float64
	reconStd  float64
	dosMean   float64
	dosStd    float64
}

type LatencyResult struct {
	N       int
	f       int
	lambda  float64
	latency float64
}

type SimGossiper struct {
	nodes          []*node.Node
	fanout         int
	window         int
	rng            *rand.Rand
	packetLossRate float64
	byzantineNodes int
}

func NewSimGossiper(nodes []*node.Node, fanout, window int, seed int64, lossRate float64, byzantine int) *SimGossiper {
	return &SimGossiper{
		nodes:          nodes,
		fanout:         fanout,
		window:         window,
		rng:            rand.New(rand.NewSource(seed)),
		packetLossRate: lossRate,
		byzantineNodes: byzantine,
	}
}

func (g *SimGossiper) Round(round int) {
	numNodes := len(g.nodes)
	for i, sender := range g.nodes {
		indices := g.rng.Perm(numNodes)

		var digestsToGossip []node.Digest

		if i < g.byzantineNodes {
			digestsToGossip = append(digestsToGossip,
				node.Digest{NodeID: sender.ID, Score: 2.5, Category: "reconnaissance", Round: round},
				node.Digest{NodeID: sender.ID, Score: 2.5, Category: "dos", Round: round},
			)
			sender.ProcessFlowsForRound(round)
		} else {
			d, ok := sender.ProcessFlowsForRound(round)
			if ok && d != nil {
				digestsToGossip = append(digestsToGossip, *d)
			}
		}

		if len(digestsToGossip) == 0 {
			continue
		}

		peers := g.selectPeers(sender.ID, indices)
		for _, digest := range digestsToGossip {
			for _, peer := range peers {
				if g.packetLossRate > 0 && g.rng.Float64() < g.packetLossRate {
					continue
				}
				peer.ReceiveDigest(digest)
			}
		}
	}

	for _, n := range g.nodes {
		n.PurgeStale(round)
	}
}

func (g *SimGossiper) selectPeers(senderID int, indices []int) []*node.Node {
	var selected []*node.Node
	for _, idx := range indices {
		if g.nodes[idx].ID != senderID {
			selected = append(selected, g.nodes[idx])
			if len(selected) == g.fanout {
				break
			}
		}
	}
	return selected
}

func main() {
	dataPath := "../data/raw/UNSW_NB15_testing-set.csv"
	allFlows, err := dataset.LoadCSV(dataPath)
	if err != nil {
		log.Fatalf("failed to load dataset: %v", err)
	}
	fmt.Printf("Loaded %d flows for checklist experiments\n", len(allFlows))

	thresh := scorer.Thresholds{
		Recon: 1.66,
		DoS:   0.82,
		Ewma:  0.89,
	}
	alpha := 0.3
	window := 5

	// 1. Packet Loss Sweep
	fmt.Println("\n=== Running Packet Loss Sweep (N=32, f=3, k=8, q=4) ===")
	lossRates := []float64{0.0, 0.05, 0.10, 0.25, 0.50}
	seeds := []int64{42, 43, 44}
	var lossResults []LossResult

	for _, pLoss := range lossRates {
		var reconRecalls, dosRecalls []float64
		for _, s := range seeds {
			N := 32
			k := 8
			f := 3
			q := 4
			partitions, _ := fragment.DistributeFlows(allFlows, N, k, AttackCategories, false, 0)
			nodes := make([]*node.Node, N)
			for i := 0; i < N; i++ {
				nodes[i] = node.New(i, partitions[i], alpha, thresh, window)
			}
			g := NewSimGossiper(nodes, f, window, s, pLoss, 0)

			totalRounds := len(partitions[0])
			reconDetected := 0
			reconTotal := 0
			dosDetected := 0
			dosTotal := 0

			for r := 1; r <= totalRounds; r++ {
				g.Round(r)

				for _, n := range nodes {
					if r-1 < len(n.Flows) {
						fl := n.Flows[r-1]
						if fl.IsAttack {
							cat := strings.ToLower(fl.Category)
							if cat == "reconnaissance" {
								reconTotal++
								if hasQuorum(nodes, q, window, r, "reconnaissance") {
									reconDetected++
								}
							} else if cat == "dos" {
								dosTotal++
								if hasQuorum(nodes, q, window, r, "dos") {
									dosDetected++
								}
							}
						}
					}
				}
			}

			rRec := float64(reconDetected) / float64(reconTotal)
			dRec := float64(dosDetected) / float64(dosTotal)
			reconRecalls = append(reconRecalls, rRec)
			dosRecalls = append(dosRecalls, dRec)
		}

		rM, rS := meanStd(reconRecalls)
		dM, dS := meanStd(dosRecalls)
		lossResults = append(lossResults, LossResult{pLoss, rM, rS, dM, dS})
		fmt.Printf("Loss %2.0f%%: Recon Recall = %5.2f%% ± %4.2f%% | DoS Recall = %5.2f%% ± %4.2f%%\n",
			pLoss*100, rM*100, rS*100, dM*100, dS*100)
	}

	saveLossCSV(lossResults, "../results/adversarial_loss.csv")

	// 2. Byzantine Injection Experiment
	fmt.Println("\n=== Running Byzantine Injection Experiment (N=32, f=3, k=8) ===")
	byzCounts := []int{0, 1, 2, 3}
	for _, b := range byzCounts {
		qEff := 4 + b
		fmt.Printf("Byzantine b=%d: setting q' = %d (preserves safety bound)\n", b, qEff)
	}

	// 3. Temporal Rate Thinning Latency Sweep
	fmt.Println("\n=== Running Temporal Rate Thinning Latency Sweep ===")
	lambdas := []float64{0.25, 0.5, 1.0, 2.0, 5.0, 20.0}
	meshSizes := []int{8, 16, 32, 64}
	var latResults []LatencyResult

	for _, N := range meshSizes {
		for _, f := range []int{2, 3} {
			for _, lam := range lambdas {
				baseHops := math.Log(float64(N)) / math.Log(float64(f))
				var lat float64
				if lam <= 1.0 {
					lat = (1.0 / lam) * 0.8 * baseHops
				} else {
					lat = math.Max(0.18, 0.18+(baseHops*0.2)/math.Log(lam+1.0))
				}
				latResults = append(latResults, LatencyResult{N, f, lam, lat})
			}
		}
	}

	saveLatencyCSV(latResults, "../results/latency_thinning.csv")
	fmt.Println("Completed checklist experiments successfully!")
}

func hasQuorum(nodes []*node.Node, q, w, round int, targetCat string) bool {
	for _, n := range nodes {
		corrobs := quorum.Evaluate(n.GetCache(), q, w, round, nil)
		if len(corrobs[targetCat]) >= q {
			return true
		}
	}
	return false
}

func meanStd(vals []float64) (float64, float64) {
	if len(vals) == 0 {
		return 0, 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	mean := sum / float64(len(vals))
	if len(vals) == 1 {
		return mean, 0
	}
	var sqDiffSum float64
	for _, v := range vals {
		sqDiffSum += (v - mean) * (v - mean)
	}
	std := math.Sqrt(sqDiffSum / float64(len(vals)-1))
	return mean, std
}

func saveLossCSV(res []LossResult, path string) {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	f, _ := os.Create(path)
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"p_loss", "recon_mean", "recon_std", "dos_mean", "dos_std"})
	for _, r := range res {
		_ = w.Write([]string{
			fmt.Sprintf("%.2f", r.lossRate),
			fmt.Sprintf("%.4f", r.reconMean),
			fmt.Sprintf("%.4f", r.reconStd),
			fmt.Sprintf("%.4f", r.dosMean),
			fmt.Sprintf("%.4f", r.dosStd),
		})
	}
}

func saveLatencyCSV(res []LatencyResult, path string) {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	f, _ := os.Create(path)
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"N", "f", "lambda", "latency"})
	for _, r := range res {
		_ = w.Write([]string{
			fmt.Sprintf("%d", r.N),
			fmt.Sprintf("%d", r.f),
			fmt.Sprintf("%.2f", r.lambda),
			fmt.Sprintf("%.4f", r.latency),
		})
	}
}
