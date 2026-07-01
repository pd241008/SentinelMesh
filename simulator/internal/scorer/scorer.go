package scorer

import (
	"math"

	"github.com/pd241008/sentinelmesh/simulator/internal/dataset"
)

// FeatureEWMA tracks EWMA statistics for a single feature
type FeatureEWMA struct {
	alpha    float64
	mean     float64
	variance float64
	count    int
}

func (f *FeatureEWMA) update(val float64) float64 {
	if f.count == 0 {
		f.mean = val
		f.variance = 0
		f.count++
		return 0
	}

	stddev := math.Sqrt(f.variance)
	var z float64
	if stddev > 0 {
		z = (val - f.mean) / stddev
	} else if val != f.mean {
		z = 3.0 // Arbitrary significant deviation if variance was 0
	}

	diff := val - f.mean
	f.mean += f.alpha * diff
	// Incremental variance update for EWMA
	f.variance = (1-f.alpha)*(f.variance + f.alpha*diff*diff)
	f.count++

	return math.Abs(z)
}

// Scorer calculates the aggregate z-score for a flow
type Scorer struct {
	alpha    float64
	features map[string]*FeatureEWMA
}

// New returns an initialized Scorer with the given alpha smoothing factor
func New(alpha float64) *Scorer {
	return &Scorer{
		alpha: alpha,
		features: map[string]*FeatureEWMA{
			"Sbytes": {alpha: alpha},
			"Dbytes": {alpha: alpha},
			"Spkts":  {alpha: alpha},
			"Dpkts":  {alpha: alpha},
			"Rate":   {alpha: alpha},
		},
	}
}

// ScoreFlow computes the anomaly score in [0, 1] range based on z-score deviation.
func (s *Scorer) ScoreFlow(flow dataset.Flow) float64 {
	zSbytes := s.features["Sbytes"].update(float64(flow.Sbytes))
	zDbytes := s.features["Dbytes"].update(float64(flow.Dbytes))
	zSpkts := s.features["Spkts"].update(float64(flow.Spkts))
	zDpkts := s.features["Dpkts"].update(float64(flow.Dpkts))
	zRate := s.features["Rate"].update(flow.Rate)

	// Combine scores using max to be sensitive to single-feature anomalies
	maxZ := zSbytes
	if zDbytes > maxZ {
		maxZ = zDbytes
	}
	if zSpkts > maxZ {
		maxZ = zSpkts
	}
	if zDpkts > maxZ {
		maxZ = zDpkts
	}
	if zRate > maxZ {
		maxZ = zRate
	}

	// Normalize z-score to a [0, 1] bounded score (e.g. capping at z=5)
	score := maxZ / 5.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}
