package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
	"github.com/pd241008/sentinelmesh/simulator/internal/sweep"
)

func main() {
	configPath := flag.String("config", "configs/sweep_default.yaml", "path to sweep config YAML")
	dataPath := flag.String("data", "", "path to UNSW-NB15 CSV dataset")
	outputDir := flag.String("output", "../../results/sweep", "output directory for results")
	alpha := flag.Float64("alpha", 0.3, "EWMA smoothing factor")
	threshold := flag.Float64("threshold", 0.5, "local anomaly score threshold")
	seed := flag.Int64("seed", 42, "random seed for reproducibility")
	coldStart := flag.Bool("cold-start", false, "clear cache before onsetRound")
	clustered := flag.Bool("clustered", false, "use 80/20 clustered fragmentation")
	flag.Parse()

	if *dataPath == "" {
		fmt.Println("Usage: simulate --data <dataset.csv> [--config <config.yaml>] [--output <dir>]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	cfg, err := sweep.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	allFlows, err := dataset.LoadCSV(*dataPath)
	if err != nil {
		log.Fatalf("failed to load dataset: %v", err)
	}
	log.Printf("Loaded %d flows from %s", len(allFlows), *dataPath)

	var attackFlows int
	for _, f := range allFlows {
		if f.IsAttack {
			attackFlows++
		}
	}
	log.Printf("Attack flows: %d / %d", attackFlows, len(allFlows))

	if err := sweep.Run(cfg, allFlows, *alpha, *threshold, *seed, *outputDir, *coldStart, *clustered); err != nil {
		log.Fatalf("sweep failed: %v", err)
	}
}
