# SentinelMesh Project Progress

This document tracks the completion status of the SentinelMesh project across its three distinct tracks: Simulator (Go), ML Validation (Python), and Dashboard (Next.js).

## Track 1: Core Simulator (Go)
**Status: Phase 1 Complete**

### Phase 1: Core Simulator Architecture
- [x] **Sub-phase 1.1: Project Setup & Data Modeling**
  - [x] Scaffolded multi-track mono-repo directory structure.
  - [x] Initialized Go module (`github.com/pd241008/sentinelmesh/simulator`).
  - [x] Created automated dataset fetch script (`data/scripts/fetch_dataset.sh`).
  - [x] Implemented `dataset.go`: Parsing UNSW-NB15 flows with strict labeling.
  - [x] Implemented `fragment.go`: Pseudo-random deterministic node partitioning (by Flow ID) and k-way targeted campaign splitting (round-robin).
- [x] **Sub-phase 1.2: Local Detection & Node Foundation**
  - [x] Configured sweep parameters (`configs/sweep_default.yaml`).
  - [x] Implemented `scorer.go`: $O(1)$ multi-feature EWMA z-score calculator (strict evaluation, no arbitrary decay).
  - [x] Implemented `node.go`: Independent node logic, digest caching by discrete round, and per-round flow ingestion.
- [x] **Sub-phase 1.3: Distributed Mechanism (Gossip & Quorum)**
  - [x] Implement `quorum.go`: Strict Equation 2 escalation rule evaluation.
  - [x] Implement `gossip.go`: Epidemic push-based exchange logic and random peer selection.
- [x] **Sub-phase 1.4: Orchestration, Baselines & Metrics**
  - [x] Implement `baseline/`: Independent (isolated) and Centralized aggregator simulation baselines.
  - [x] Implement `metrics/`: Logic for recall, bandwidth, and convergence latency tracking.
  - [x] Implement `sweep/`: Automated execution loop for N/f/q/k parameters.
  - [x] Implement `cmd/simulate/main.go`: CLI entry point.

## Track 2: ML Crosscheck (Python)
**Status: In Progress — Phase 2**

### Phase 2: ML Validation Pipeline
- [x] **Sub-phase 2.1: Project Setup & Data Pipeline**
  - [x] Setup `pyproject.toml`, virtual env, and testing frameworks (pytest).
  - [x] Implement data loader for partitioned UNSW-NB15 CSV outputs (reuses dataset format from Track 1).
  - [x] Write unit tests for data loader and preprocessing.
- [x] **Sub-phase 2.2: Scorer Models & Training**
  - [x] Implement Isolation Forest scorer with configurable hyperparameters (`n_estimators`, `max_samples`, `contamination`, `random_state`).
  - [x] Implement Autoencoder-based scorer (PyTorch) with configurable `latent_dim`, `learning_rate`, `epochs`, `batch_size`.
  - [x] Train both models on normal traffic and evaluate on all flows — scores normalized to [0, 1].
  - [x] Write unit tests for model scoring, inference, shape, range, and error handling (15 tests).
- [x] **Sub-phase 2.3: Validation & Reporting**
  - [x] Implement automated validation runner (`run_validation`, `generate_crosscheck_report`) comparing Go EWMA scorer vs ML models (Isolation Forest, Autoencoder).
  - [x] Generate comparison reports — per-category and overall precision/recall/F1, output as CSV + JSON.
  - [x] Write integration tests for the full validation pipeline (11 tests covering pipeline execution, multi-node, per-category metrics, report output).
  - [x] Output results to `results/crosscheck/` for dashboard consumption (`summary.csv`, `per_category.csv`, `per_flow_scores.csv`, `report.json`).

## Track 3: Dashboard (Next.js)
**Status: In Progress — Phase 3**

### Phase 3: Visualization & Frontend
- [ ] **Sub-phase 3.1: Scaffolding & Build Config**
  - [ ] Initialize Next.js project with TypeScript and Tailwind CSS.
  - [ ] Configure ESLint, Prettier, and project directory structure.
  - [ ] Create shared TypeScript types for sweep and crosscheck results.
  - [ ] Set up basic layout (navbar, sidebar, page routing).
- [ ] **Sub-phase 3.2: Data Layer & API**
  - [ ] Implement data loader for `results/sweep/` CSV output.
  - [ ] Implement data loader for `results/crosscheck/` comparison reports.
  - [ ] Create client-side data fetching hooks with caching.
  - [ ] Write unit tests for data parsing and transformation.
- [ ] **Sub-phase 3.3: Interactive Visualization Components**
  - [ ] Implement sweep results chart (recall vs N/f/q/k with toggles).
  - [ ] Implement bandwidth overhead comparison chart (gossip vs centralized).
  - [ ] Implement convergence latency heatmap.
  - [ ] Write component tests using React Testing Library.
- [ ] **Sub-phase 3.4: Integration & Views**
  - [ ] Build sweep overview page with parameter selection UI.
  - [ ] Build ML crosscheck comparison view (Go scorer vs ML models).
  - [ ] Add responsive design and dark mode support.
  - [ ] Write end-to-end tests for key user flows.
