#!/usr/bin/env python3
"""
EWMA Threshold Calibration — percentile-based derivation matching the method
used for τ_recon (1.66) and τ_dos (0.82).

Loads UNSW-NB15 training data, runs the Go-matching EWMA scorer (no clipping)
over normal flows to establish the score distribution, then picks the threshold
at the 95th percentile. Also reports scores for the residual categories
(analysis, backdoor, exploits, fuzzers, generic, shellcode, worms) to verify
separation.
"""
import sys
import numpy as np
import pandas as pd

sys.path.insert(0, "/root/workspace/workspace/03-Code/Projects/Legacy/SentinalMesh/ml-crosscheck/src")

from dataset import FlowDataset, FEATURE_COLUMNS
from models.go_ewma import GoEWMAScorer, _FeatureEWMA

# ---------------------------------------------------------------------------
# Go-matching EWMA scorer — no clipping (unlike the Python go_ewma.py which clips to 1.0)
# ---------------------------------------------------------------------------
class GoMatchingEWMAScorer(GoEWMAScorer):
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
                z = self._features[ewma_name].update(np.log1p(float(row[j])))
                if z > max_z:
                    max_z = z
            scores[i] = max_z / 5.0  # NO clipping — matches Go scorer.go:85
        return scores


def main():
    data_path = "/root/workspace/workspace/03-Code/Projects/Legacy/centralized_datasets/raw/unsw_nb15/UNSW_NB15_training-set.csv"
    print(f"Loading training data from {data_path}...")
    ds = FlowDataset.load_csv(data_path)
    print(f"  Total flows: {len(ds)}")
    print(f"  Attack flows: {len(ds.attack_flows)}")
    print(f"  Normal flows: {len(ds.normal_flows)}")

    # Categories present in the data
    print(f"\nCategory distribution:")
    cat_col = "attack_cat"
    if cat_col in ds.flows.columns:
        cats = ds.flows[cat_col].astype(str).str.strip().str.lower()
        # Replace 'normal' with 'normal' and nan with 'normal'
        cats = cats.fillna("normal").replace(["", "nan"], "normal")
        print(cats.value_counts())
    else:
        print("  (no attack_cat column)")

    # Categories we care about for EWMA: residual = everything except dos, reconnaissance, normal
    residual_categories = ["analysis", "backdoor", "exploits", "fuzzers", "generic", "shellcode", "worms"]

    # -----------------------------------------------------------------------
    # Phase 1: Train on ALL normal flows, scoring each one (online/streaming)
    # -----------------------------------------------------------------------
    print(f"\n{'='*70}")
    print("Phase 1: EWMA score distribution over NORMAL traffic")
    print(f"{'='*70}")

    normal_ds = ds.normal_flows
    normal_scores = []

    # Fresh EWMA scorer, train on normal flows
    def _init_features(scorer_obj):
        scorer_obj._features = {
            name: _FeatureEWMA(scorer_obj.alpha)
            for name in scorer_obj.FEATURE_MAP.values()
        }

    # Fresh EWMA scorer — initialize features manually, then score online
    scorer = GoMatchingEWMAScorer(alpha=0.3)
    _init_features(scorer)
    features = normal_ds[FEATURE_COLUMNS].fillna(0).to_numpy(dtype=np.float64)
    for i in range(len(normal_ds)):
        row = features[i]
        max_z = 0.0
        for j, col in enumerate(FEATURE_COLUMNS):
            ewma_name = scorer.FEATURE_MAP[col]
            z = scorer._features[ewma_name].update(np.log1p(float(row[j])))
            if z > max_z:
                max_z = z
        score = max_z / 5.0
        normal_scores.append(score)

    normal_scores = np.array(normal_scores)
    print(f"  Normal flows scored: {len(normal_scores)}")

    # Percentiles
    for p in [50, 75, 90, 95, 97.5, 99, 99.5, 99.9]:
        val = np.percentile(normal_scores, p)
        print(f"  P{p:5.1f} = {val:.4f}")

    print(f"\n  Normal EWMA score stats:")
    print(f"    Mean:    {np.mean(normal_scores):.4f}")
    print(f"    Std:     {np.std(normal_scores):.4f}")
    print(f"    Max:     {np.max(normal_scores):.4f}")
    print(f"    Median:  {np.median(normal_scores):.4f}")

    # -----------------------------------------------------------------------
    # Phase 2: Score residual categories (analysis, backdoor, exploits, fuzzers, generic, shellcode, worms)
    # -----------------------------------------------------------------------
    print(f"\n{'='*70}")
    print("Phase 2: EWMA scores for residual attack categories")
    print(f"{'='*70}")

    if cat_col in ds.flows.columns:
        for cat in residual_categories:
            cat_flows = ds.flows[ds.flows[cat_col].astype(str).str.strip().str.lower() == cat]
            if len(cat_flows) == 0:
                print(f"\n  {cat}: NO FLOWS FOUND")
                continue

            cat_scores = []
            scorer3 = GoMatchingEWMAScorer(alpha=0.3)
            _init_features(scorer3)
            cat_features = cat_flows[FEATURE_COLUMNS].fillna(0).to_numpy(dtype=np.float64)
            for i in range(len(cat_flows)):
                row = cat_features[i]
                max_z = 0.0
                for j, col in enumerate(FEATURE_COLUMNS):
                    ewma_name = scorer3.FEATURE_MAP[col]
                    z = scorer3._features[ewma_name].update(np.log1p(float(row[j])))
                    if z > max_z:
                        max_z = z
                score = max_z / 5.0
                cat_scores.append(score)

            cat_scores = np.array(cat_scores)
            print(f"\n  {cat}: {len(cat_scores)} flows")
            print(f"    Mean:    {np.mean(cat_scores):.4f}")
            print(f"    Median:  {np.median(cat_scores):.4f}")
            print(f"    P25-P75: [{np.percentile(cat_scores, 25):.4f}, {np.percentile(cat_scores, 75):.4f}]")
            print(f"    P95:     {np.percentile(cat_scores, 95):.4f}")
            print(f"    Max:     {np.max(cat_scores):.4f}")
            # Fraction above P95-normal threshold
            above_p95 = np.sum(cat_scores > np.percentile(normal_scores, 95))
            print(f"    % > Normal P95: {above_p95 / len(cat_scores) * 100:.1f}%")
            above_p99 = np.sum(cat_scores > np.percentile(normal_scores, 99))
            print(f"    % > Normal P99: {above_p99 / len(cat_scores) * 100:.1f}%")

    # -----------------------------------------------------------------------
    # Phase 3: Score DoS and Recon flows for comparison
    # -----------------------------------------------------------------------
    print(f"\n{'='*70}")
    print("Phase 3: EWMA scores for DoS and Recon (for comparison)")
    print(f"{'='*70}")

    for cat in ["dos", "reconnaissance"]:
        cat_flows = ds.flows[ds.flows[cat_col].astype(str).str.strip().str.lower() == cat]
        if len(cat_flows) == 0:
            print(f"\n  {cat}: NO FLOWS FOUND")
            continue

        cat_scores = []
        scorer4 = GoMatchingEWMAScorer(alpha=0.3)
        _init_features(scorer4)
        cat_features = cat_flows[FEATURE_COLUMNS].fillna(0).to_numpy(dtype=np.float64)
        for i in range(len(cat_flows)):
            row = cat_features[i]
            max_z = 0.0
            for j, col in enumerate(FEATURE_COLUMNS):
                ewma_name = scorer4.FEATURE_MAP[col]
                z = scorer4._features[ewma_name].update(np.log1p(float(row[j])))
                if z > max_z:
                    max_z = z
            score = max_z / 5.0
            cat_scores.append(score)

        cat_scores = np.array(cat_scores)
        print(f"\n  {cat}: {len(cat_scores)} flows")
        print(f"    Mean:    {np.mean(cat_scores):.4f}")
        print(f"    Median:  {np.median(cat_scores):.4f}")
        print(f"    P25-P75: [{np.percentile(cat_scores, 25):.4f}, {np.percentile(cat_scores, 75):.4f}]")
        print(f"    P95:     {np.percentile(cat_scores, 95):.4f}")
        print(f"    Max:     {np.max(cat_scores):.4f}")
        above_p95 = np.sum(cat_scores > np.percentile(normal_scores, 95))
        print(f"    % > Normal P95: {above_p95 / len(cat_scores) * 100:.1f}%")

    # -----------------------------------------------------------------------
    # Recommendation
    # -----------------------------------------------------------------------
    p95_normal = np.percentile(normal_scores, 95)
    p99_normal = np.percentile(normal_scores, 99)
    print(f"\n{'='*70}")
    print(f"RECOMMENDATION")
    print(f"{'='*70}")
    print(f"  Current threshold_ewma: 0.8")
    print(f"  Normal P95:             {p95_normal:.4f}")
    print(f"  Normal P99:             {p99_normal:.4f}")
    print(f"  Normal P99.5:           {np.percentile(normal_scores, 99.5):.4f}")
    print(f"  Normal P99.9:           {np.percentile(normal_scores, 99.9):.4f}")

    # Check if 0.8 is close to any meaningful percentile
    actual_p = np.sum(normal_scores <= 0.8) / len(normal_scores) * 100
    print(f"  Current 0.8 is at P{actual_p:.1f} of normal scores")
    print(f"\n  => Set threshold_ewma to P95 ({p95_normal:.4f}) to match τ_recon/τ_dos calibration method")


if __name__ == "__main__":
    main()
