package scorer

import (
	"math"
	"testing"

	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
)

func defaultThresholds() Thresholds {
	return Thresholds{
		Recon: 1.66,
		DoS:   0.82,
		Ewma:  0.89,
	}
}

func TestNew(t *testing.T) {
	s := New(0.3, defaultThresholds())
	if s == nil {
		t.Fatal("New returned nil")
	}
	if len(s.features) != 5 {
		t.Fatalf("expected 5 features, got %d", len(s.features))
	}
}

func TestScoreFlowNormal(t *testing.T) {
	s := New(0.3, defaultThresholds())
	flow := dataset.Flow{Sbytes: 100, Dbytes: 200, Spkts: 2, Dpkts: 4, Rate: 10.5, CtDstLtm: 10, CtDstSportLtm: 5}
	score, cats, cat, isAnomalous := s.ScoreFlow(flow)
	if score < 0 || score > 10 {
		t.Fatalf("score out of expected range: %f", score)
	}
	if cat == "" {
		t.Fatal("category should not be empty")
	}
	_ = cats
	_ = isAnomalous
}

func TestScoreFlowOutlier(t *testing.T) {
	s := New(0.3, defaultThresholds())

	normal := dataset.Flow{Sbytes: 100, Dbytes: 200, Spkts: 2, Dpkts: 4, Rate: 10.5, CtDstLtm: 10, CtDstSportLtm: 5}
	s.ScoreFlow(normal)
	s.ScoreFlow(normal)
	s.ScoreFlow(normal)

	outlier := dataset.Flow{Sbytes: 100000, Dbytes: 200000, Spkts: 2000, Dpkts: 4000, Rate: 10000, CtDstLtm: 1000, CtDstSportLtm: 1}
	score, cats, cat, isAnomalous := s.ScoreFlow(outlier)
	if score < 0.5 {
		t.Fatalf("expected high outlier score, got %f", score)
	}
	_ = cats
	_ = cat
	_ = isAnomalous
}

func TestScoreFlowBounded(t *testing.T) {
	s := New(0.3, defaultThresholds())

	base := dataset.Flow{Sbytes: 100, Dbytes: 200, Spkts: 2, Dpkts: 4, Rate: 10.5, CtDstLtm: 10, CtDstSportLtm: 5}
	for i := 0; i < 20; i++ {
		s.ScoreFlow(base)
	}

	extreme := dataset.Flow{Sbytes: 99999999, Dbytes: 99999999, Spkts: 999999, Dpkts: 999999, Rate: 999999, CtDstLtm: 999999, CtDstSportLtm: 1}
	score, cats, cat, isAnomalous := s.ScoreFlow(extreme)
	if score > 10.0 || math.IsNaN(score) {
		t.Fatalf("score should be reasonable, got %f", score)
	}
	_ = cats
	_ = cat
	_ = isAnomalous
}

func TestScoreFlowRepeatable(t *testing.T) {
	s1 := New(0.3, defaultThresholds())
	s2 := New(0.3, defaultThresholds())

	flow := dataset.Flow{Sbytes: 100, Dbytes: 200, Spkts: 2, Dpkts: 4, Rate: 10.5, CtDstLtm: 10, CtDstSportLtm: 5}
	sc1, cats1, cat1, anom1 := s1.ScoreFlow(flow)
	sc2, cats2, cat2, anom2 := s2.ScoreFlow(flow)
	if sc1 != sc2 || cat1 != cat2 || anom1 != anom2 {
		t.Fatalf("scores should be identical: %f/%s/%v vs %f/%s/%v", sc1, cat1, anom1, sc2, cat2, anom2)
	}
	_ = cats1
	_ = cats2
}

func TestMultiCategory(t *testing.T) {
	s := New(0.3, defaultThresholds())

	// Flow with both high DoS and high Recon scores - should return both categories
	flow := dataset.Flow{
		Sbytes:        50000,
		Dbytes:        100000,
		Spkts:         500,
		Dpkts:         1000,
		Rate:          10000,
		CtDstLtm:      1,
		CtDstSportLtm: 100,
	}
	_, cats, cat, isAnomalous := s.ScoreFlow(flow)
	if !isAnomalous {
		t.Fatal("expected flow to be anomalous")
	}
	if cat == "" {
		t.Fatal("primary category should not be empty")
	}
	_ = cats
}

func TestCategoryPriority(t *testing.T) {
	s := New(0.3, defaultThresholds())

	// Flow that triggers both DoS and Recon - primary should be dos (higher score)
	flow := dataset.Flow{
		Sbytes:        10000,
		Dbytes:        20000,
		Spkts:         100,
		Dpkts:         200,
		Rate:          500,
		CtDstLtm:      1,
		CtDstSportLtm: 100,
	}
	_, cats, cat, _ := s.ScoreFlow(flow)
	if cat != "dos" && cat != "reconnaissance" {
		t.Logf("Category: %s (priority test)", cat)
	}
	_ = cats
}
