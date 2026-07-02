from pathlib import Path

import numpy as np
import pytest
from sklearn.preprocessing import StandardScaler

from dataset import FEATURE_COLUMNS, FlowDataset

TESTDATA = Path(__file__).resolve().parent.parent.parent / "simulator" / "testdata" / "testdata.csv"


@pytest.fixture
def ds() -> FlowDataset:
    return FlowDataset.load_csv(TESTDATA)


class TestLoadCSV:
    def test_loads_correct_count(self, ds: FlowDataset):
        assert len(ds) == 15

    def test_loads_all_flows_have_ids(self, ds: FlowDataset):
        ids = ds.flows["id"].tolist() if "id" in ds.flows.columns else list(range(len(ds)))
        assert len(ids) == 15

    def test_missing_file_raises(self):
        with pytest.raises(FileNotFoundError):
            FlowDataset.load_csv("/nonexistent/path.csv")


class TestLabelAttacks:
    def test_normal_flow_labeled_correctly(self, ds: FlowDataset):
        normal = ds.flows[ds.flows["attack_cat"] == "normal"]
        assert not normal["is_attack"].any()

    def test_attack_flow_labeled_correctly(self, ds: FlowDataset):
        attacks = ds.flows[ds.flows["attack_cat"] != "normal"]
        assert attacks["is_attack"].all()

    def test_attack_count(self, ds: FlowDataset):
        assert ds.attack_flows.shape[0] == 7
        assert ds.normal_flows.shape[0] == 8


class TestGetFeatures:
    def test_returns_expected_shape(self, ds: FlowDataset):
        X = ds.get_features()
        assert X.shape == (15, 5)

    def test_columns_match_feature_list(self, ds: FlowDataset):
        X = ds.get_features()
        assert X.shape[1] == len(FEATURE_COLUMNS)

    def test_returns_float64(self, ds: FlowDataset):
        X = ds.get_features()
        assert X.dtype == np.float64

    def test_known_attack_high_sbytes(self, ds: FlowDataset):
        X = ds.get_features()
        attack_mask = ds.get_labels()
        attack_sbytes = X[attack_mask, 0]
        normal_sbytes = X[~attack_mask, 0]
        assert attack_sbytes.max() > normal_sbytes.max()

    def test_empty_feature_set(self):
        import pandas as pd
        ds = FlowDataset(pd.DataFrame({"a": [1, 2, 3]}))
        with pytest.raises(ValueError, match="no feature columns"):
            ds.get_features()


class TestGetLabels:
    def test_labels_match_is_attack(self, ds: FlowDataset):
        labels = ds.get_labels()
        assert labels.dtype == bool
        assert labels.sum() == 7

    def test_all_attack_flows_true(self, ds: FlowDataset):
        labels = ds.get_labels()
        attack_cats = ds.flows["attack_cat"].values
        for i, cat in enumerate(attack_cats):
            if cat != "normal":
                assert labels[i], f"expected attack=True for {cat} at index {i}"
            else:
                assert not labels[i], f"expected attack=False for normal at index {i}"


class TestGetCategories:
    def test_returns_all_categories(self, ds: FlowDataset):
        cats = ds.get_categories()
        assert len(cats) == 15
        assert "dos" in cats
        assert "normal" in cats

    def test_known_categories_present(self, ds: FlowDataset):
        cats = set(ds.get_categories())
        assert cats.issuperset({"normal", "fuzzers", "dos", "exploits", "generic", "analysis", "backdoor"})


class TestTrainTestSplit:
    def test_split_contains_all_attacks(self, ds: FlowDataset):
        train, test = ds.train_test_split(train_ratio=0.8)
        assert len(train) + len(test) == len(ds)
        assert test.attack_flows.shape[0] == 7

    def test_train_has_no_attacks(self, ds: FlowDataset):
        train, _ = ds.train_test_split(train_ratio=0.8)
        assert train.attack_flows.shape[0] == 0

    def test_normal_only_in_train(self, ds: FlowDataset):
        train, test = ds.train_test_split(train_ratio=0.5)
        assert len(train) <= len(ds.normal_flows)
        assert len(test) >= len(ds.attack_flows)


class TestNormalize:
    def test_normalize_produces_zero_mean(self, ds: FlowDataset):
        norm_ds, scaler = ds.normalize()
        X = norm_ds.get_features()
        means = X.mean(axis=0)
        assert np.allclose(means, 0, atol=1e-10)

    def test_normalize_produces_unit_variance(self, ds: FlowDataset):
        norm_ds, scaler = ds.normalize()
        X = norm_ds.get_features()
        stds = X.std(axis=0, ddof=0)
        assert np.allclose(stds, 1, atol=1e-10)

    def test_normalize_with_external_scaler(self, ds: FlowDataset):
        norm_ds1, scaler = ds.normalize()
        norm_ds2, _ = ds.normalize(scaler=scaler)
        np.testing.assert_array_almost_equal(
            norm_ds1.get_features(), norm_ds2.get_features(), decimal=5
        )


class TestPartitionByNode:
    def test_partition_into_4_nodes(self, ds: FlowDataset):
        ds.flows["id"] = range(len(ds))
        parts = ds.partition_by_node(4)
        assert len(parts) == 4
        total = sum(len(p) for p in parts)
        assert total == 15
        for p in parts:
            assert len(p) > 0

    def test_partition_without_id_column(self):
        import pandas as pd
        ds2 = FlowDataset(pd.DataFrame({"a": [1, 2, 3]}))
        parts = ds2.partition_by_node(4)
        assert len(parts) == 1
        assert len(parts[0]) == 3
