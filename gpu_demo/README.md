# gpu_demo — Parallel & GPU Computing Showcase for SentinelMesh

Four self-contained demos that solve **the same project problem** — per-flow
anomaly scoring (normalized distance between a flow's feature vector and the
benign-traffic centroid, the same math as `simulator/scorer.go`) — using four
parallel-programming models, **all on the project's real UNSW-NB15 data**, plus
a run of the production ML pipeline itself.

## One-command teacher demo

```bash
cd gpu_demo
make demo        # builds + runs everything in sequence (~4 min)
```

Individual targets: `make data` (export real flows) · `make run-openmp` ·
`make run-cuda` · `make run-mpi` · `make run-py` · `make crosscheck-gpu` ·
`make clean`

## How this is wired to the project (not just a demo)

```
data/raw/UNSW_NB15_testing-set.csv            <- the real dataset (82,332 flows)
        |
        v
ml-crosscheck/src/dataset.py  FlowDataset     <- the PROJECT's own loader
        |                (FEATURE_COLUMNS: sbytes, dbytes, spkts, dpkts, rate)
        v
gpu_demo/export_features.py  ->  flows.bin    <- real flows + benign centroid + labels
        |
        +--> openmp_flows.c   (CPU threads on real flows)
        +--> cuda_flows.cu    (GPU kernels on real flows)
        +--> mpi_flows.c      (mesh partitions of real flows)
        |
        +--> gpu_benchmark.py  imports the PRODUCTION AutoencoderScorer
        |                     (ml-crosscheck/src/models/autoencoder.py) —
        |                     no re-implementation; proves weights land in VRAM
        v
make crosscheck-gpu  ->  runs ml-crosscheck train.py generate_crosscheck_report()
                         -> writes the project's real artifacts:
                            results/crosscheck/{summary.csv, per_category.csv,
                            per_flow_scores.csv, report.json}
```

What "wired" means here, concretely:

1. **Same data** — every demo scores flows exported through the production
   `FlowDataset` loader (attack/normal labels included; attacks measurably
   score ~2x normal).
2. **Same math** — benign-centroid distance, matching `simulator/scorer.go`.
3. **Same model class** — the PyTorch track instantiates the shipped
   `AutoencoderScorer` and prints which device its weights physically live on
   (`cuda:0` in Part D).
4. **Same output contract** — `make crosscheck-gpu` executes the real
   pipeline and regenerates `results/crosscheck/`, which the dashboard reads.

The only non-production element is an honest, labeled emulation: real flows
are tiled x250 to emulate multi-round mesh traffic volume (the production
5-feature workload is ~0.8 ms — too small for meaningful per-thread timing).

## The four tracks

| # | Track   | File                                      | Programming model                                      | What it proves                                                                     |
| - | ------- | ----------------------------------------- | ------------------------------------------------------ | ---------------------------------------------------------------------------------- |
| 1 | OpenMP  | `openmp_flows.c`                        | Shared-memory threads (`#pragma omp parallel for`)   | CPU cores scale ~3-4x, then hit the memory-bandwidth wall                          |
| 2 | CUDA    | `cuda_flows.cu`                         | GPU kernels, 1 thread per flow (+ wide-row experiment) | Same math ~60x faster on 2560 GPU cores; PCIe transfers can cost more than compute |
| 3 | MPI     | `mpi_flows.c`                           | Distributed memory (`MPI_Scatterv` / `MPI_Reduce`) | Models the mesh itself: shared-nothing nodes, message-passing aggregation          |
| 4 | PyTorch | `gpu_benchmark.py` + `crosscheck-gpu` | The production model class on GPU                      | Where the GPU genuinely pays off — and where it honestly doesn't                  |

All demos verify their results against a serial checksum — parallelization
changed *speed*, never the *answer*.

## Measured results on this machine

> i9-13900H (20 logical cores) + RTX 4050 Laptop GPU (6 GB, cc 8.9, 2560 CUDA
> cores), CUDA 13.3, PyTorch 2.9.1+cu128, OpenMPI. Timings are representative
> ranges — they fluctuate a little every run, which is itself worth mentioning.

**Track 1 — OpenMP** (20.6M real flow-scorings, 20.6M x 5 features):

|    Threads |      Time |          Speedup |
| ---------: | --------: | ---------------: |
| 1 (serial) |   ~130 ms |             1.0x |
|          2 |    ~42 ms |              ~3x |
|          4 |    ~29 ms |              ~4x |
|       8-20 | ~30-43 ms | ~3-4x (flattens) |

Real-data signal: mean anomaly score attacks ~61,835 vs normal ~29,640.

**Track 2 — CUDA** (same real flows, x250 emulation):

| Implementation               |    Time |  vs CPU serial |
| ---------------------------- | ------: | -------------: |
| CPU serial (1 core)          | ~160 ms |           1.0x |
| K1: thread-per-flow          | ~2.7 ms | **~60x** |
| K2: warp-per-flow, coalesced |  ~15 ms |           ~11x |
| K3: warp-per-flow, strided   |  ~15 ms |    ~1.0x vs K2 |

Teaching points baked into the output:

- At only **5 production features** the naive K1 kernel wins: 20M independent
  threads hide memory latency better than a warp-cooperative scheme. A
  clearly-labeled wide-row experiment (F=64, synthetic) demonstrates the
  coalescing penalty that appears once feature vectors grow.
- **PCIe:** uploading the 412 MB flow matrix took **~150 ms** vs a ~3-15 ms
  kernel — data movement dominates; that is why training batches live in VRAM.
- Attacks score ~2x normal on the real data — the GPU is doing the actual IDS
  work, not a toy kernel.

**Track 3 — MPI** (same real flows, 8 ranks):

```
1. scatter  (move 412 MB of real flows) :  ~340 ms
2. scoring  (slowest of 8 ranks)        :  ~25 ms   <- pure compute wall time
3. reduce   (quorum aggregation)        :  ~10 ms
compute-only speedup with 8 ranks       :  ~3-8x (ideal 8.0x)
```

The killer observation: **moving the raw flows cost ~13x more than computing
on them** (341 ms scatter vs 26 ms scoring). This is the empirical justification
of SentinelMesh's core design claim — why mesh nodes gossip constant-size
*digests* instead of raw flows, and why a centralized SIEM (which hoovers all
raw logs to one place) is structurally expensive.

**Track 4 — PyTorch, production class** (real `AutoencoderScorer`, 82,332 flows):

| Workload                                  |      CPU |                   GPU |                      Speedup |
| ----------------------------------------- | -------: | --------------------: | ---------------------------: |
| [A] Shipped 5-feature pipeline, 15 epochs |    ~14 s |                 ~36 s |  **0.4x — GPU LOSES** |
| [A] Batch-score all 82k flows             |   ~15 ms |                ~27 ms |                        ~0.6x |
| [B] Same class, 39 features               |    ~18 s |                 ~39 s |                        ~0.5x |
| [C] Single 8192² matmul                  | ~4500 ms |               ~143 ms | **~31x** (7.7 TFLOP/s) |
| [D] Residency proof                       |       — | weights on `cuda:0` |                           — |

Part A/B are deliberately reported honestly: the shipped scorer network is
tiny (5 features, hidden layer of 4) and the training loop is dominated by
per-batch kernel-launch overhead, so the GPU is *slower* at this scale. This
is the most important slide of the demo — **GPUs are not automatically faster;
they amortize launch cost over massive parallel work.** Part C shows the payoff
curve: the larger the matrices, the more the 2560-core architecture wins
(6x -> 31x as size grows 1024 -> 8192).

`make crosscheck-gpu` then runs the **production** pipeline
(`generate_crosscheck_report`) end-to-end on the GPU and regenerates the
project's `results/crosscheck/` artifacts — the exact files the dashboard
visualizes. (The metric values there are the project's own, at its current
thresholds; the demo does not alter them.)

## How the GPU is incorporated into the project — and the justification

**Where it is incorporated (already in the codebase, not demo-only):**

- `ml-crosscheck/src/models/autoencoder.py` — `AutoencoderScorer` auto-selects
  `cuda` when available; model, training batches and scoring tensors are placed
  on the GPU via `.to(self.device)`. PyTorch dispatches every layer's GEMM to
  cuBLAS on the GPU automatically — no code change between CPU and GPU, only
  the device. `gpu_benchmark.py` Part D proves the weights physically reside
  in VRAM.
- `make crosscheck-gpu` runs that production path and refreshes the real
  `results/crosscheck/` artifacts.
- The simulator's local EWMA scorer stays on the CPU **on purpose**: edge IDS
  nodes are resource-constrained micro-services doing O(1) work per flow, and
  there is no GPU at the network edge. That is an architecture decision the
  demo can defend, not an oversight.

**Why the GPU is justified (the argument to present):**

1. **The workload is data-parallel.** Scoring N flows = N independent distance
   computations; training = dense matrix multiplies. Both map naturally onto
   thousands of GPU threads (Track 2: measured ~60x on real flows).
2. **The batch must be large enough to amortize overhead.** Track 4 shows the
   crossover explicitly: tiny nets lose (0.4x), large batched ops win (~31x).
   Knowing *where* the crossover is, is the competence being demonstrated.
3. **The pipeline grows into the GPU's sweet spot.** Full-feature models,
   larger latent dimensions, hyperparameter sweeps over the mesh grid
   (N, f, k, q) and batch re-scoring are the growth path — Track 4 Part B/C
   measures exactly that direction.
4. **It matches the deployment story.** The centralized analyst side
   (retraining the cross-check model over the full UNSW-NB15 set) has GPU
   capacity; the edge node deliberately does not. SentinelMesh is a project
   where the CPU/GPU *split* is itself the design.

**The cross-model comparison (one sentence per model, for the viva):**

- OpenMP: add parallelism to existing CPU code with one pragma; limited by
  core count and shared memory bandwidth (~3-4x here).
- MPI: scale across machines/nodes with explicit message passing; we used it
  to *model the mesh*, and it demonstrated that data movement dominates
  (scatter 341 ms vs compute 26 ms) — the empirical case for gossiping digests.
- CUDA: explicit control of thousands of threads and the memory hierarchy;
  biggest win (~60x) but demands coalesced layouts and transfer management.
- PyTorch: the practical GPU path — CUDA without writing CUDA; the shipped
  model class runs on either device unchanged, and the framework handles
  kernels, cuBLAS and memory for you.

## 5-minute teacher demo script

1. **`nvidia-smi -l 1`** in a second terminal — show the idle GPU, leave it running.
2. **`make run-openmp`** — "Real UNSW-NB15 flows on CPU threads: 130 -> 30 ms.
   Note the flattening: shared memory bus. Attacks score 2x normal."
3. **`make run-cuda`** — "Same real flows, ~60x faster. K2/K3 + the labeled
   wide-row experiment is the coalescing lesson. PCIe: 150 ms to move what the
   kernel reads in 3."
4. **`make run-mpi`** — "This is our mesh: 8 shared-nothing nodes over real
   flow partitions, aggregation by message passing. Scatter 341 ms vs compute
   26 ms is *why* we gossip digests instead of raw flows."
5. **`make run-py`** — "The production AutoencoderScorer class, imported — not
   re-implemented. GPU loses on our tiny shipped net (honest!) and wins ~31x
   on the matmuls the pipeline grows into. Part D shows its weights living in
   VRAM."
6. **`make crosscheck-gpu`** — "And this is the production pipeline itself
   running on the GPU, regenerating the real results the dashboard shows."
7. Close: "The GPU is justified for the ML cross-check track by measurement,
   and deliberately absent at the simulated edge — both by design."

## Notes for classmates on other laptops

- Needs: NVIDIA GPU + CUDA toolkit (`nvcc`), gcc, and for the MPI track
  `openmpi` (Arch: `sudo pacman -S openmpi`, Debian: `sudo apt install openmpi-bin libopenmpi-dev`).
- No GPU? Tracks 1 and 3 still run anywhere (CPU only); the Makefile skips
  what it cannot build.
- Running as root (WSL/containers): the Makefile auto-adds
  `mpirun --allow-run-as-root`.
- Dataset expected at `../data/raw/UNSW_NB15_testing-set.csv`
  (`./data/scripts/fetch_dataset.sh` from the repo root fetches it).
- `flows.bin` (2 MB) is generated — `make data` recreates it; `make clean`
  removes it.
