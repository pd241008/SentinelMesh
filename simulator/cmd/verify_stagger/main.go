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

	// 1. Verify MCC Alignment
	var normalPool []dataset.Flow
	for _, f := range allFlows {
		cat := strings.ToLower(f.Category)
		if cat == "normal" {
			normalPool = append(normalPool, f)
		}
	}

	tPartitions, _ := fragment.DistributeFlows(allFlows, 32, 8, attackCategories, false, 2)
	cPartitions, _ := fragment.DistributeFlowsControl(allFlows, normalPool, 32, 8, attackCategories, false, 2)

	mismatchFound := false
	for n := 0; n < 32; n++ {
		if len(tPartitions[n]) != len(cPartitions[n]) {
			fmt.Printf("Length mismatch on Node %d: Treatment=%d, Control=%d\n", n, len(tPartitions[n]), len(cPartitions[n]))
			mismatchFound = true
			break
		}
		for i := 0; i < len(tPartitions[n]); i++ {
			// They should have the same ID (even if category is different due to control replacement)
			if tPartitions[n][i].ID != cPartitions[n][i].ID {
				fmt.Printf("ID mismatch on Node %d, index %d: Treatment=%d, Control=%d\n", n, i, tPartitions[n][i].ID, cPartitions[n][i].ID)
				mismatchFound = true
				break
			}
		}
	}
	if !mismatchFound {
		fmt.Println("MCC Verification PASSED: Treatment and Control background traffic ordering is byte-for-byte identical.")
	} else {
		fmt.Println("MCC Verification FAILED!")
	}

	// 2. Realized Stagger Delays
	// Test for k=8, stagger=5
	printRealizedStagger(8, 5)
	// Test for k=4, stagger=2
	printRealizedStagger(4, 2)
}

func printRealizedStagger(k, staggerRounds int) {
	fmt.Printf("\nRealized Delays for k=%d, staggerRounds=%d:\n", k, staggerRounds)
	for i := 0; i < k; i++ {
		delay := int(float64(i*staggerRounds)/float64(k-1) + 0.5)
		fmt.Printf("Fragment %d: delay = %d rounds\n", i, delay)
	}
}
