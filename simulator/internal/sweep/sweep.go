package sweep

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/pd241008/sentinelmesh/simulator/internal/baseline"
	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
	"github.com/pd241008/sentinelmesh/simulator/internal/fragment"
	"github.com/pd241008/sentinelmesh/simulator/internal/gossip"
	"github.com/pd241008/sentinelmesh/simulator/internal/metrics"
	"github.com/pd241008/sentinelmesh/simulator/internal/node"
	"github.com/pd241008/sentinelmesh/simulator/internal/quorum"
	"github.com/pd241008/sentinelmesh/simulator/internal/scorer"
	"gopkg.in/yaml.v3"
)

var AttackCategories = []string{
	"analysis", "backdoor", "dos", "exploits", "fuzzers",
	"generic", "reconnaissance", "shellcode", "worms",
}

type Config struct {
	Sweep struct {
		N []int `yaml:"N"`
		F []int `yaml:"f"`
		Q []int `yaml:"q"`
		K []int `yaml:"k"`
		W int   `yaml:"W"`
		Thresholds struct {
			Recon float64 `yaml:"threshold_recon"`
			DoS   float64 `yaml:"threshold_dos"`
			Ewma  float64 `yaml:"threshold_ewma"`
		} `yaml:"thresholds"`
	} `yaml:"sweep"`
}

type RunResult struct {
	N              int
	F              int
	Q              int
	K              int
	W              int
	GossipFlowReconRecall   float64
	GossipFlowDoSRecall     float64
	GossipWindowReconRecall float64
	GossipWindowDoSRecall   float64
	GossipReconFPR          float64
	GossipDoSFPR            float64
	GossipBandwidth         int
	GossipLatency           float64
	IndepFlowReconRecall    float64
	IndepFlowDoSRecall      float64
	IndepWindowReconRecall  float64
	IndepWindowDoSRecall    float64
	IndepReconFPR           float64
	IndepDoSFPR             float64
	IndepBandwidth          int
	IndepLatency            float64
	CentFlowReconRecall     float64
	CentFlowDoSRecall       float64
	CentWindowReconRecall   float64
	CentWindowDoSRecall     float64
	CentReconFPR            float64
	CentDoSFPR              float64
	CentBandwidth           int
	CentLatency             float64
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Run(cfg *Config, allFlows []dataset.Flow, alpha float64, threshold float64, seed int64, outputDir string) error {
	var results []RunResult
	idx := int64(0)

	for _, N := range cfg.Sweep.N {
		for _, k := range cfg.Sweep.K {
			partitions := fragment.DistributeFlows(allFlows, N, k, AttackCategories)
			totalRounds := 0
			for _, p := range partitions {
				if len(p) > totalRounds {
					totalRounds = len(p)
				}
			}

			for _, f := range cfg.Sweep.F {
				for _, q := range cfg.Sweep.Q {
					idx++
					result := RunResult{
						N: N, F: f, Q: q, K: k, W: cfg.Sweep.W,
					}

					thresh := scorer.Thresholds{
						Recon: cfg.Sweep.Thresholds.Recon,
						DoS:   cfg.Sweep.Thresholds.DoS,
						Ewma:  cfg.Sweep.Thresholds.Ewma,
					}

					gossipNodes := makeNodes(N, partitions, alpha, cfg.Sweep.W, thresh)
					gResult := runGossip(gossipNodes, f, cfg.Sweep.W, q, totalRounds, seed+idx)
					gMetrics := metrics.Compute(gossipNodes, allFlows, gResult.alerts, gResult.totalDigests, cfg.Sweep.W)
					result.GossipFlowReconRecall = gMetrics.FlowReconRecall
					result.GossipFlowDoSRecall = gMetrics.FlowDoSRecall
					result.GossipWindowReconRecall = gMetrics.WindowReconRecall
					result.GossipWindowDoSRecall = gMetrics.WindowDoSRecall
					result.GossipReconFPR = gMetrics.ReconFPR
					result.GossipDoSFPR = gMetrics.DoSFPR
					result.GossipBandwidth = gMetrics.Bandwidth
					result.GossipLatency = gMetrics.AvgLatency

					indepNodes := makeNodes(N, partitions, alpha, cfg.Sweep.W, thresh)
					iResult := baseline.RunIndependent(indepNodes, cfg.Sweep.W, totalRounds, q)
					iMetrics := metrics.Compute(indepNodes, allFlows, iResult.Alerts, iResult.TotalDigests, cfg.Sweep.W)
					result.IndepFlowReconRecall = iMetrics.FlowReconRecall
					result.IndepFlowDoSRecall = iMetrics.FlowDoSRecall
					result.IndepWindowReconRecall = iMetrics.WindowReconRecall
					result.IndepWindowDoSRecall = iMetrics.WindowDoSRecall
					result.IndepReconFPR = iMetrics.ReconFPR
					result.IndepDoSFPR = iMetrics.DoSFPR
					result.IndepBandwidth = 0
					result.IndepLatency = iMetrics.AvgLatency

					centNodes := makeNodes(N, partitions, alpha, cfg.Sweep.W, thresh)
					cResult := baseline.RunCentralized(centNodes, cfg.Sweep.W, totalRounds, q)
					cMetrics := metrics.Compute(centNodes, allFlows, cResult.Alerts, cResult.TotalDigests, cfg.Sweep.W)
					result.CentFlowReconRecall = cMetrics.FlowReconRecall
					result.CentFlowDoSRecall = cMetrics.FlowDoSRecall
					result.CentWindowReconRecall = cMetrics.WindowReconRecall
					result.CentWindowDoSRecall = cMetrics.WindowDoSRecall
					result.CentReconFPR = cMetrics.ReconFPR
					result.CentDoSFPR = cMetrics.DoSFPR
					result.CentBandwidth = 0
					result.CentLatency = cMetrics.AvgLatency

					results = append(results, result)
				}
			}
		}
	}

	return writeCSV(results, outputDir)
}

type gossipResult struct {
	alerts        baseline.AlertTimeline
	totalDigests  int
}

func runGossip(nodes []*node.Node, f, w int, q int, rounds int, seed int64) gossipResult {
	g := gossip.New(nodes, f, w, seed)
	alerts := make(baseline.AlertTimeline)
	totalDigests := 0

	for round := 1; round <= rounds; round++ {
		g.Round(round)
		totalDigests += f * len(nodes)

		for _, n := range nodes {
			for _, cat := range quorum.Evaluate(n.GetCache(), q, w, round) {
				if alerts[round] == nil {
					alerts[round] = make(map[string]bool)
				}
				alerts[round][cat] = true
			}
		}
	}

	return gossipResult{alerts: alerts, totalDigests: totalDigests}
}

func makeNodes(n int, partitions [][]dataset.Flow, alpha float64, window int, thresh scorer.Thresholds) []*node.Node {
	nodes := make([]*node.Node, n)
	for i := 0; i < n; i++ {
		nodes[i] = node.New(i, partitions[i], alpha, thresh, window)
	}
	return nodes
}

func writeCSV(results []RunResult, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}
	path := filepath.Join(outputDir, "sweep_results.csv")

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"N", "f", "q", "k", "W",
		"gossip_flow_recon_recall", "gossip_flow_dos_recall", "gossip_window_recon_recall", "gossip_window_dos_recall", "gossip_recon_fpr", "gossip_dos_fpr", "gossip_bandwidth", "gossip_latency",
		"indep_flow_recon_recall", "indep_flow_dos_recall", "indep_window_recon_recall", "indep_window_dos_recall", "indep_recon_fpr", "indep_dos_fpr", "indep_bandwidth", "indep_latency",
		"cent_flow_recon_recall", "cent_flow_dos_recall", "cent_window_recon_recall", "cent_window_dos_recall", "cent_recon_fpr", "cent_dos_fpr", "cent_bandwidth", "cent_latency",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, r := range results {
		row := []string{
			intStr(r.N), intStr(r.F), intStr(r.Q), intStr(r.K), intStr(r.W),
			floatStr(r.GossipFlowReconRecall), floatStr(r.GossipFlowDoSRecall), floatStr(r.GossipWindowReconRecall), floatStr(r.GossipWindowDoSRecall), floatStr(r.GossipReconFPR), floatStr(r.GossipDoSFPR), intStr(r.GossipBandwidth), floatStr(r.GossipLatency),
			floatStr(r.IndepFlowReconRecall), floatStr(r.IndepFlowDoSRecall), floatStr(r.IndepWindowReconRecall), floatStr(r.IndepWindowDoSRecall), floatStr(r.IndepReconFPR), floatStr(r.IndepDoSFPR), intStr(r.IndepBandwidth), floatStr(r.IndepLatency),
			floatStr(r.CentFlowReconRecall), floatStr(r.CentFlowDoSRecall), floatStr(r.CentWindowReconRecall), floatStr(r.CentWindowDoSRecall), floatStr(r.CentReconFPR), floatStr(r.CentDoSFPR), intStr(r.CentBandwidth), floatStr(r.CentLatency),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	fmt.Printf("Wrote %d results to %s\n", len(results), path)
	return nil
}

func intStr(i int) string {
	return fmt.Sprintf("%d", i)
}

func floatStr(f float64) string {
	if math.IsNaN(f) {
		return "0"
	}
	return fmt.Sprintf("%.4f", f)
}
