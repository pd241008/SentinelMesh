import json
import tempfile
from pathlib import Path

import numpy as np
import pandas as pd
import pytest

from dataset import FlowDataset
from models import AutoencoderScorer, GoEWMAScorer, IsolationForestScorer
from train import (
    evaluate_scorer,
    generate_crosscheck_report,
    run_validation,
)

TESTDATA = Path(__file__).resolve().parent.parent.parent / "simulator" / "testdata" / "testdata.csv"


@pytest.fixture
def ds() -> FlowDataset:
    return FlowDataset.load_csv(TESTDATA)


class TestFullPipeline:
    def test_run_validation_returns_all_scorers(self, ds: FlowDataset):
        results = run_validation(TESTDATA, num_nodes=1, include_go_ewma=True)
        assert len(results) == 3
        names = [r.model_name for r in results]
        assert "go_ewma_node0" in names
        assert "if_node0" in names
        assert "ae_node0" in names

    def test_run_validation_multiple_nodes(self, ds: FlowDataset):
        ds.flows["id"] = range(len(ds))
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "data.csv"
            ds.flows.to_csv(path, index=False)
            results = run_validation(path, num_nodes=2, include_go_ewma=False)
            assert len(results) == 4
            names = [r.model_name for r in results]
            assert "if_node0" in names
            assert "if_node1" in names
            assert "ae_node0" in names
            assert "ae_node1" in names

    def test_each_result_has_per_category_metrics(self, ds: FlowDataset):
        results = run_validation(TESTDATA, num_nodes=1, include_go_ewma=True)
        for r in results:
            assert len(r.per_category) > 0
            for pc in r.per_category:
                assert 0.0 <= pc.precision <= 1.0
                assert 0.0 <= pc.recall <= 1.0
                assert 0.0 <= pc.f1_score <= 1.0
                assert pc.total > 0

    def test_scores_match_labels_shape(self, ds: FlowDataset):
        results = run_validation(TESTDATA, num_nodes=1)
        for r in results:
            assert r.scores.shape == r.labels.shape

    def test_metrics_are_in_range(self, ds: FlowDataset):
        results = run_validation(TESTDATA, num_nodes=1)
        for r in results:
            assert 0.0 <= r.accuracy <= 1.0
            assert 0.0 <= r.precision <= 1.0
            assert 0.0 <= r.recall <= 1.0
            assert 0.0 <= r.f1_score <= 1.0

    def test_go_ewma_matches_expected_range(self, ds: FlowDataset):
        scorer = GoEWMAScorer()
        scorer.train(ds)
        scores = scorer.score(ds)
        assert scores.shape == (15,)
        assert scores.min() >= 0.0
        assert scores.max() <= 1.0


class TestCrosscheckReport:
    def test_generates_all_output_files(self, ds: FlowDataset):
        ds.flows["id"] = range(len(ds))
        with tempfile.TemporaryDirectory() as tmp:
            data_path = Path(tmp) / "data.csv"
            ds.flows.to_csv(data_path, index=False)
            results = generate_crosscheck_report(data_path, num_nodes=1, output_dir=tmp)

            assert len(results) == 3

            summary_path = Path(tmp) / "summary.csv"
            assert summary_path.exists()
            summary_df = pd.read_csv(summary_path)
            assert list(summary_df.columns) == [
                "model", "accuracy", "precision", "recall", "f1",
                "avg_score_attack", "avg_score_normal",
            ]
            assert len(summary_df) == 3

            per_cat_path = Path(tmp) / "per_category.csv"
            assert per_cat_path.exists()
            per_cat_df = pd.read_csv(per_cat_path)
            assert list(per_cat_df.columns) == [
                "model", "category", "total", "detected",
                "precision", "recall", "f1", "avg_score",
            ]
            assert len(per_cat_df) > 0

            flow_scores_path = Path(tmp) / "per_flow_scores.csv"
            assert flow_scores_path.exists()
            flow_df = pd.read_csv(flow_scores_path)
            assert "flow_id" in flow_df.columns
            assert "category" in flow_df.columns
            assert "label" in flow_df.columns
            assert "go_ewma_node0" in flow_df.columns
            assert "if_node0" in flow_df.columns
            assert "ae_node0" in flow_df.columns
            assert len(flow_df) == 15

            json_path = Path(tmp) / "report.json"
            assert json_path.exists()
            with open(json_path) as f:
                report = json.load(f)
            assert len(report) == 3
            assert "model" in report[0]
            assert "overall" in report[0]
            assert "per_category" in report[0]

    def test_no_id_column_still_works(self, ds: FlowDataset):
        with tempfile.TemporaryDirectory() as tmp:
            data_path = Path(tmp) / "data.csv"
            ds.flows.drop(columns=["id"], errors="ignore").to_csv(data_path, index=False)
            results = generate_crosscheck_report(data_path, num_nodes=1, output_dir=tmp)
            assert len(results) == 3
            summary_df = pd.read_csv(Path(tmp) / "summary.csv")
            assert len(summary_df) == 3

    def test_go_ewma_disabled(self, ds: FlowDataset):
        with tempfile.TemporaryDirectory() as tmp:
            data_path = Path(tmp) / "data.csv"
            ds.flows.to_csv(data_path, index=False)
            results = generate_crosscheck_report(data_path, num_nodes=1, output_dir=tmp, threshold=0.5)
            names = [r.model_name for r in results]
            assert "go_ewma_node0" in names
            assert "if_node0" in names
            assert "ae_node0" in names


class TestPerCategoryMetrics:
    def test_normal_category_all_detected_as_normal(self, ds: FlowDataset):
        results = run_validation(TESTDATA, num_nodes=1, include_go_ewma=True)
        for r in results:
            for pc in r.per_category:
                if pc.category == "normal":
                    assert pc.detected <= pc.total
                    break

    def test_attack_categories_have_detections(self, ds: FlowDataset):
        results = run_validation(TESTDATA, num_nodes=1, include_go_ewma=False)
        for r in results:
            attack_cats = [pc for pc in r.per_category if pc.category != "normal"]
            detected_any = any(pc.detected > 0 for pc in attack_cats)
            assert detected_any, f"{r.model_name} detected no attack categories"
