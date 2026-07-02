# Simulator Tests

Go-based unit and integration tests for the core simulator (Track 1).

## Stats
- **41 tests** across 10 packages
- All tests pass

## Packages tested
| Package | Focus |
|---|---|
| `internal/dataset` | UNSW-NB15 CSV parsing and flow extraction |
| `internal/fragment` | Node partitioning and k-way campaign splitting |
| `internal/scorer` | EWMA z-score computation and flow scoring |
| `internal/node` | Node logic, digest caching, per-round ingestion |
| `internal/gossip` | Epidemic push exchange, random peer selection |
| `internal/quorum` | Equation 2 escalation rule evaluation |
| `internal/baseline` | Independent and Centralized aggregator baselines |
| `internal/metrics` | Recall, bandwidth, convergence latency |
| `internal/sweep` | Parameter sweep execution loop |
| `tests/` | Full pipeline integration + sweep integration + end-to-end baselines |

## Run
```bash
cd simulator
go test ./... -v -count=1
```

## Run from project root
```bash
cd simulator && go test ./... -count=1
```
