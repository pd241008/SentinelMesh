from abc import ABC, abstractmethod

import numpy as np

from dataset import FlowDataset


class BaseScorer(ABC):
    @abstractmethod
    def train(self, dataset: FlowDataset) -> None:
        ...

    @abstractmethod
    def score(self, dataset: FlowDataset) -> np.ndarray:
        ...

    def predict(self, dataset: FlowDataset, threshold: float = 0.5) -> np.ndarray:
        return self.score(dataset) >= threshold
