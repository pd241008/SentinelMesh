from pathlib import Path

import numpy as np
import pytest
import torch

from dataset import FlowDataset, FEATURE_COLUMNS
from models import IsolationForestScorer, AutoencoderScorer, BaseScorer

TESTDATA = Path(__file__).resolve().parent.parent.parent / "simulator" / "testdata" / "testdata.csv"


@pytest.fixture
def ds() -> FlowDataset:
    return FlowDataset.load_csv(TESTDATA)


@pytest.fixture
def normal_ds(ds: FlowDataset) -> FlowDataset:
    return FlowDataset(ds.normal_flows)


class TestBaseScorer:
    def test_predict_uses_score(self):
        class DummyScorer(BaseScorer):
            def train(self, dataset):
                pass
            def score(self, dataset):
                return np.array([0.1, 0.9, 0.6, 0.4])

        scorer = DummyScorer()
        preds = scorer.predict(None, threshold=0.5)
        assert preds.tolist() == [False, True, True, False]


class TestIsolationForestScorer:
    def test_train_and_score_returns_expected_shape(self, ds: FlowDataset):
        scorer = IsolationForestScorer(random_state=42)
        scorer.train(ds)
        scores = scorer.score(ds)
        assert scores.shape == (15,)
        assert scores.dtype == np.float64

    def test_scores_in_0_1_range(self, ds: FlowDataset):
        scorer = IsolationForestScorer(random_state=42)
        scorer.train(ds)
        scores = scorer.score(ds)
        assert scores.min() >= 0.0
        assert scores.max() <= 1.0

    def test_attack_flows_get_higher_scores(self, ds: FlowDataset):
        scorer = IsolationForestScorer(random_state=42)
        scorer.train(ds)
        scores = scorer.score(ds)
        labels = ds.get_labels()
        attack_mean = scores[labels].mean()
        normal_mean = scores[~labels].mean()
        assert attack_mean >= normal_mean

    def test_score_before_train_raises(self, ds: FlowDataset):
        scorer = IsolationForestScorer()
        with pytest.raises(RuntimeError, match="train"):
            scorer.score(ds)

    def test_configurable_hyperparameters(self):
        scorer = IsolationForestScorer(n_estimators=50, max_samples=0.5, contamination=0.05)
        assert scorer.n_estimators == 50
        assert scorer.max_samples == 0.5
        assert scorer.contamination == 0.05

    def test_train_on_normal_only(self, ds: FlowDataset):
        scorer = IsolationForestScorer(random_state=42)
        normal = FlowDataset(ds.normal_flows)
        scorer.train(normal)
        scores = scorer.score(ds)
        assert scores.shape == (15,)


class TestAutoencoderScorer:
    def test_train_and_score_returns_expected_shape(self, ds: FlowDataset):
        scorer = AutoencoderScorer(epochs=10)
        scorer.train(ds)
        scores = scorer.score(ds)
        assert scores.shape == (15,)
        assert scores.dtype in (np.float32, np.float64)

    def test_scores_in_0_1_range(self, ds: FlowDataset):
        scorer = AutoencoderScorer(epochs=10)
        scorer.train(ds)
        scores = scorer.score(ds)
        assert scores.min() >= 0.0
        assert scores.max() <= 1.0

    def test_attack_flows_get_higher_scores(self, ds: FlowDataset):
        scorer = AutoencoderScorer(epochs=50)
        scorer.train(ds)
        scores = scorer.score(ds)
        labels = ds.get_labels()
        attack_mean = scores[labels].mean()
        normal_mean = scores[~labels].mean()
        assert attack_mean >= normal_mean

    def test_score_before_train_raises(self, ds: FlowDataset):
        scorer = AutoencoderScorer()
        with pytest.raises(RuntimeError, match="train"):
            scorer.score(ds)

    def test_configurable_hyperparameters(self):
        scorer = AutoencoderScorer(latent_dim=3, epochs=200, learning_rate=1e-4)
        assert scorer.latent_dim == 3
        assert scorer.epochs == 200
        assert scorer.learning_rate == 1e-4

    def test_device_property(self):
        scorer = AutoencoderScorer()
        assert scorer.device in ("cpu", "cuda")

    def test_train_on_normal_only(self, ds: FlowDataset):
        scorer = AutoencoderScorer(epochs=10)
        normal = FlowDataset(ds.normal_flows)
        scorer.train(normal)
        scores = scorer.score(ds)
        assert scores.shape == (15,)

    def test_model_is_on_correct_device(self, ds: FlowDataset):
        scorer = AutoencoderScorer(epochs=5)
        scorer.train(ds)
        params = list(scorer._model.parameters())
        assert all(p.device.type == scorer.device for p in params)
