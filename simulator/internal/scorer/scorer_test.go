package scorer

import (
	"math"
	"testing"

	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
)

func TestNew(t *testing.T) {
	s := New(0.3)
	if s == nil {
		t.Fatal("New returned nil")
	}
	if len(s.features) != 5 {
		t.Fatalf("expected 5 features, got %d", len(s.features))
	}
}

func TestScoreFlowNormal(t *testing.T) {
	s := New(0.3)
	flow := dataset.Flow{Sbytes: 100, Dbytes: 200, Spkts: 2, Dpkts: 4, Rate: 10.5}
	score := s.ScoreFlow(flow)
	if score < 0 || score > 1 {
		t.Fatalf("score out of range [0,1]: %f", score)
	}
}

func TestScoreFlowOutlier(t *testing.T) {
	s := New(0.3)

	normal := dataset.Flow{Sbytes: 100, Dbytes: 200, Spkts: 2, Dpkts: 4, Rate: 10.5}
	s.ScoreFlow(normal)
	s.ScoreFlow(normal)
	s.ScoreFlow(normal)

	outlier := dataset.Flow{Sbytes: 100000, Dbytes: 200000, Spkts: 2000, Dpkts: 4000, Rate: 10000}
	score := s.ScoreFlow(outlier)
	if score < 0.5 {
		t.Fatalf("expected high outlier score, got %f", score)
	}
}

func TestScoreFlowBounded(t *testing.T) {
	s := New(0.3)

	base := dataset.Flow{Sbytes: 100, Dbytes: 200, Spkts: 2, Dpkts: 4, Rate: 10.5}
	for i := 0; i < 20; i++ {
		s.ScoreFlow(base)
	}

	extreme := dataset.Flow{Sbytes: 99999999, Dbytes: 99999999, Spkts: 999999, Dpkts: 999999, Rate: 999999}
	score := s.ScoreFlow(extreme)
	if score > 1.0 || math.IsNaN(score) {
		t.Fatalf("score should be clamped to [0,1], got %f", score)
	}
}

func TestScoreFlowRepeatable(t *testing.T) {
	s1 := New(0.3)
	s2 := New(0.3)

	flow := dataset.Flow{Sbytes: 100, Dbytes: 200, Spkts: 2, Dpkts: 4, Rate: 10.5}
	sc1 := s1.ScoreFlow(flow)
	sc2 := s2.ScoreFlow(flow)
	if sc1 != sc2 {
		t.Fatalf("scores should be identical: %f vs %f", sc1, sc2)
	}
}
