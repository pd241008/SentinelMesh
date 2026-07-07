# SentinelMesh Results Summary

This document summarizes the Gossip FPR metrics extracted using the post-alignment-fix pipeline, alongside the corrected recall numbers for the canonical Table 1.

## Table I (N=32, f=3, k=8)

| $q$ | Recon Recall (Corrected) | Recon FPR (Gossip) | DoS Recall (Corrected) | DoS FPR (Gossip) |
|---|---|---|---|---|
| **2** | 50.31% ± 0.00% | **68.55%** ± 0.29% | 79.23% ± 0.47% | **82.77%** ± 0.06% |
| **4** | 44.84% ± 0.28% | **38.25%** ± 0.40% | 74.20% ± 1.03% | **64.35%** ± 0.54% |
| **6** | 27.52% ± 1.44% | **14.09%** ± 0.43% | 52.91% ± 0.87% | **37.22%** ± 0.17% |
| **8** | 22.55% ± 1.06% | **6.24%** ± 0.08% | 27.68% ± 0.91% | **2.66%** ± 0.38% |

### Note on Independent Baseline
The Independent FPR figures (**34.2%** for Recon and **9.9%** for DoS) remain fully unaffected by the alignment fix. These metrics are derived directly from the threshold-calibration percentile analysis and operate independently of the treatment/control window-matching pipeline.
