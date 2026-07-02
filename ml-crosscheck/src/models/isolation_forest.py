import numpy as np
from sklearn.ensemble import IsolationForest

from dataset import FlowDataset

from .base import BaseScorer


class IsolationForestScorer(BaseScorer):
    def __init__(
        self,
        n_estimators: int = 100,
        max_samples: float | int = "auto",
        contamination: float = 0.1,
        random_state: int = 42,
    ):
        self.n_estimators = n_estimators
        self.max_samples = max_samples
        self.contamination = contamination
        self.random_state = random_state
        self._model: IsolationForest | None = None

    def train(self, dataset: FlowDataset) -> None:
        X = dataset.get_features(dataset.normal_flows)
        self._model = IsolationForest(
            n_estimators=self.n_estimators,
            max_samples=self.max_samples,
            contamination=self.contamination,
            random_state=self.random_state,
        )
        self._model.fit(X)

    def score(self, dataset: FlowDataset) -> np.ndarray:
        if self._model is None:
            raise RuntimeError("model must be trained before scoring")
        X = dataset.get_features()
        raw = self._model.decision_function(X)
        raw = -raw
        mn, mx = raw.min(), raw.max()
        if mx - mn < 1e-12:
            return np.full_like(raw, 0.0)
        return (raw - mn) / (mx - mn)
