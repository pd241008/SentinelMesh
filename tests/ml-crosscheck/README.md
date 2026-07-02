# ML Crosscheck Tests

Python-based tests for the ML crosscheck pipeline (Track 2).

## Stats
- **49 tests** across 3 test files
- All tests pass

## Test suites
| File | Tests | Focus |
|---|---|---|
| `test_dataset.py` | 23 | Data loading, attack labeling, feature extraction, train/test split, normalization, node partitioning |
| `test_models.py` | 15 | Isolation Forest scorer, Autoencoder scorer, base interface, score bounds, error handling |
| `test_integration.py` | 11 | Full pipeline, multi-node validation, per-category metrics, crosscheck report output (CSV + JSON) |

## Run from project root
```bash
python -m pytest tests/ml-crosscheck/ -v
```

## Run from track directory
```bash
cd ml-crosscheck
pytest tests/ -v
```
