package sweep

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", "configs", "sweep_default.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if len(cfg.Sweep.N) != 4 {
		t.Fatalf("expected 4 N values, got %d", len(cfg.Sweep.N))
	}
	if cfg.Sweep.W != 5 {
		t.Fatalf("expected W=5, got %d", cfg.Sweep.W)
	}
	if len(cfg.Sweep.F) != 3 {
		t.Fatalf("expected 3 f values, got %d", len(cfg.Sweep.F))
	}
	if len(cfg.Sweep.K) != 3 {
		t.Fatalf("expected 3 k values, got %d", len(cfg.Sweep.K))
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestRun(t *testing.T) {
	cfg := &Config{}
	cfg.Sweep.N = []int{4}
	cfg.Sweep.F = []int{1}
	cfg.Sweep.K = []int{2}
	cfg.Sweep.W = 3

	allFlows := loadTestFlows()
	if allFlows == nil {
		t.Fatal("failed to load test flows")
	}

	outputDir := t.TempDir()
	if err := Run(cfg, allFlows, 0.3, 0.3, 99, outputDir, false, false, 0); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	csvPath := filepath.Join(outputDir, "sweep_results.csv")
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		t.Fatalf("expected CSV at %s", csvPath)
	}
	data, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("CSV is empty")
	}
}

func loadTestFlows() []dataset.Flow {
	// Use the test dataset
	flows, err := dataset.LoadCSV(filepath.Join("..", "..", "testdata", "testdata.csv"))
	if err != nil {
		return nil
	}
	return flows
}
