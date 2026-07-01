package dataset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCSV(t *testing.T) {
	flows, err := LoadCSV(filepath.Join("..", "..", "testdata", "testdata.csv"))
	if err != nil {
		t.Fatalf("LoadCSV failed: %v", err)
	}
	if len(flows) != 15 {
		t.Fatalf("expected 15 flows, got %d", len(flows))
	}
	if flows[0].ID != 1 || flows[0].Category != "normal" || flows[0].IsAttack {
		t.Errorf("bad first flow: %+v", flows[0])
	}
	if !flows[2].IsAttack || flows[2].Category != "fuzzers" {
		t.Errorf("expected attack flow at index 2: %+v", flows[2])
	}
	if flows[2].Sbytes != 500 || flows[2].Dbytes != 800 {
		t.Errorf("bad sbytes/dbytes: %d %d", flows[2].Sbytes, flows[2].Dbytes)
	}
}

func TestLoadCSVEmpty(t *testing.T) {
	f, err := os.CreateTemp("", "empty*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("sttime,attack_cat,sbytes,dbytes,spkts,dpkts,rate\n")
	f.Close()

	_, err = LoadCSV(f.Name())
	if err == nil {
		t.Fatal("expected error for empty dataset")
	}
}

func TestLoadCSVNoFile(t *testing.T) {
	_, err := LoadCSV("/nonexistent/path.csv")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
