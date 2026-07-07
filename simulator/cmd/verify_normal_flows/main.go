package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
	"github.com/pd241008/sentinelmesh/simulator/internal/fragment"
)

func main() {
	allFlows, err := dataset.LoadCSV("../data/raw/UNSW_NB15_testing-set.csv")
	if err != nil {
		log.Fatal(err)
	}

	attackCategories := []string{"reconnaissance", "dos"}

	var normalPool []dataset.Flow
	for _, f := range allFlows {
		cat := strings.ToLower(f.Category)
		if cat == "normal" {
			normalPool = append(normalPool, f)
		}
	}

	tPartitions, _ := fragment.DistributeFlows(allFlows, 32, 8, attackCategories, false, 0)
	cPartitions, _ := fragment.DistributeFlowsControl(allFlows, normalPool, 32, 8, attackCategories, false, 0)

	// Pick 20 untouched Normal flows from the dataset (they have IsAttack == false and cat == "normal")
	var sampleFlows []dataset.Flow
	for _, f := range allFlows {
		if strings.ToLower(f.Category) == "normal" {
			sampleFlows = append(sampleFlows, f)
			if len(sampleFlows) == 10020 { sampleFlows = sampleFlows[10000:]; break 
				break
			}
		}
	}

	fmt.Println("Untouched Normal Flows: Node & Round Index Comparison (Treatment vs Control)")
	fmt.Printf("%-8s | %-15s | %-15s | %-15s | %-15s | %-10s\n", "Flow ID", "T-Node", "C-Node", "T-Round", "C-Round", "Round Diff")
	fmt.Println(strings.Repeat("-", 90))

	for _, sf := range sampleFlows {
		tNode, tRound := findFlow(tPartitions, sf.ID)
		cNode, cRound := findFlow(cPartitions, sf.ID)
		
		diff := tRound - cRound

		fmt.Printf("%-8d | %-15d | %-15d | %-15d | %-15d | %+10d\n", sf.ID, tNode, cNode, tRound, cRound, diff)
	}
}

func findFlow(partitions [][]dataset.Flow, id int) (int, int) {
	for n := 0; n < len(partitions); n++ {
		for r := 0; r < len(partitions[n]); r++ {
			if partitions[n][r].ID == id {
				return n, r
			}
		}
	}
	return -1, -1
}
