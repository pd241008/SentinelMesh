package dataset

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Flow represents a single network flow derived from the dataset
type Flow struct {
	ID        int
	Timestamp float64
	Category  string
	IsAttack  bool

	// Features used by the local scorer
	Sbytes int
	Dbytes int
	Spkts  int
	Dpkts  int
	Rate   float64
}

// LoadCSV parses a UNSW-NB15 dataset CSV file. It handles dynamic column mapping
// based on the header row to extract relevant features for the simulation.
func LoadCSV(path string) ([]Flow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("empty dataset")
	}

	header := records[0]
	colIdx := make(map[string]int)
	for i, h := range header {
		colIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	var flows []Flow
	for i, row := range records[1:] {
		flow := Flow{
			ID: i + 1,
		}

		if idx, ok := colIdx["sttime"]; ok {
			flow.Timestamp, _ = strconv.ParseFloat(row[idx], 64)
		} else if idx, ok := colIdx["timestamp"]; ok {
			flow.Timestamp, _ = strconv.ParseFloat(row[idx], 64)
		}

		if idx, ok := colIdx["attack_cat"]; ok {
			cat := strings.TrimSpace(row[idx])
			if cat == "" || strings.ToLower(cat) == "normal" {
				flow.Category = "normal"
				flow.IsAttack = false
			} else {
				flow.Category = strings.ToLower(cat)
				flow.IsAttack = true
			}
		} else {
			flow.Category = "normal"
		}

		if idx, ok := colIdx["sbytes"]; ok {
			flow.Sbytes, _ = strconv.Atoi(row[idx])
		}
		if idx, ok := colIdx["dbytes"]; ok {
			flow.Dbytes, _ = strconv.Atoi(row[idx])
		}
		if idx, ok := colIdx["spkts"]; ok {
			flow.Spkts, _ = strconv.Atoi(row[idx])
		}
		if idx, ok := colIdx["dpkts"]; ok {
			flow.Dpkts, _ = strconv.Atoi(row[idx])
		}
		if idx, ok := colIdx["rate"]; ok {
			flow.Rate, _ = strconv.ParseFloat(row[idx], 64)
		}

		flows = append(flows, flow)
	}

	return flows, nil
}
