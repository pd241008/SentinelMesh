import numpy as np

from dataset import FEATURE_COLUMNS, FlowDataset

from .base import BaseScorer


class _FeatureEWMA:
    def __init__(self, alpha: float):
        self.alpha = alpha
        self.mean = 0.0
        self.variance = 0.0
        self.count = 0

    def update(self, val: float) -> float:
        if self.count == 0:
            self.mean = val
            self.variance = 0.0
            self.count += 1
            return 0.0

        stddev = np.sqrt(self.variance)
        if stddev > 0:
            z = (val - self.mean) / stddev
        elif val != self.mean:
            z = 3.0
        else:
            z = 0.0

        diff = val - self.mean
        self.mean += self.alpha * diff
        self.variance = (1.0 - self.alpha) * (self.variance + self.alpha * diff * diff)
        self.count += 1

        return abs(z)


class GoEWMAScorer(BaseScorer):
    FEATURE_MAP = {
        "sbytes": "Sbytes",
        "dbytes": "Dbytes",
        "spkts": "Spkts",
        "dpkts": "Dpkts",
        "rate": "Rate",
    }

    def __init__(self, alpha: float = 0.3):
        self.alpha = alpha
        self._features: dict[str, _FeatureEWMA] = {}
        self._trained = False

    def train(self, dataset: FlowDataset) -> None:
        self._features = {
            name: _FeatureEWMA(self.alpha)
            for name in self.FEATURE_MAP.values()
        }
        X = dataset.get_features()
        available_cols = [c for c in FEATURE_COLUMNS if c in dataset.flows.columns]
        for i in range(len(X)):
            row = X[i]
            for j, col in enumerate(available_cols):
                ewma_name = self.FEATURE_MAP[col]
                self._features[ewma_name].update(float(row[j]))
        self._trained = True

    def score(self, dataset: FlowDataset) -> np.ndarray:
        if not self._trained:
            raise RuntimeError("model must be trained before scoring")
        X = dataset.get_features()
        available_cols = [c for c in FEATURE_COLUMNS if c in dataset.flows.columns]
        scores = np.zeros(len(X), dtype=np.float64)
        for i in range(len(X)):
            row = X[i]
            max_z = 0.0
            for j, col in enumerate(available_cols):
                ewma_name = self.FEATURE_MAP[col]
                z = self._features[ewma_name].update(float(row[j]))
                if z > max_z:
                    max_z = z
            raw = max_z / 5.0
            scores[i] = min(raw, 1.0)
        return scores
