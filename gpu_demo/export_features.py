#!/usr/bin/env python3
"""
export_features.py — bridge between the real project pipeline and the C/CUDA demos.

Loads UNSW-NB15 through the project's OWN FlowDataset (ml-crosscheck/src/
dataset.py), extracts the SAME FEATURE_COLUMNS the production scorers use
(sbytes, dbytes, spkts, dpkts, rate), and writes one flat binary the C demos
can fread:

    [int32 magic 'UNSW'] [int32 n_flows] [int32 n_features]
    [n_flows * n_features float32, row-major]   <- real flows, real values
    [n_features float32]                        <- benign-traffic centroid
    [n_flows int32]                             <- 1 = attack, 0 = normal

The centroid is the mean of the NORMAL flows — exactly what the Go scorer
measures distance against — so the C/CUDA/OpenMP/MPI demos score the same
real data with the same real math as the rest of SentinelMesh.

Usage:  python3 export_features.py [--data data/raw/UNSW_NB15_testing-set.csv]
        (run from gpu_demo/ ; output is gpu_demo/flows.bin)
"""

import argparse
import struct
import sys
from pathlib import Path

import numpy as np

REPO = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(REPO / "ml-crosscheck" / "src"))

from dataset import FlowDataset  # noqa: E402  <- the PROJECT's loader


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--data", default=str(REPO / "data" / "raw" / "UNSW_NB15_testing-set.csv"))
    ap.add_argument("--out", default=str(Path(__file__).resolve().parent / "flows.bin"))
    args = ap.parse_args()

    ds = FlowDataset.load_csv(args.data)          # project loader, project columns
    X = ds.get_features().astype(np.float32)      # production FEATURE_COLUMNS
    labels = ds.get_labels().astype(np.int32)     # is_attack

    normal = X[labels == 0]
    centroid = normal.mean(axis=0, dtype=np.float64).astype(np.float32)  # benign centroid

    n, f = X.shape
    with open(args.out, "wb") as fh:
        fh.write(struct.pack("<4sii", b"UNSW", n, f))
        fh.write(X.tobytes())
        fh.write(centroid.tobytes())
        fh.write(labels.tobytes())

    mb = args.out and Path(args.out).stat().st_size / 1e6
    print(f"exported {n:,} real UNSW-NB15 flows x {f} features -> {args.out} ({mb:.0f} MB)")
    print(f"  attacks: {int(labels.sum()):,} | normal: {int((labels == 0).sum()):,}")
    print("  source: FlowDataset (ml-crosscheck/src/dataset.py) — production pipeline")


if __name__ == "__main__":
    main()
