# ============================================================================
# Makefile — SentinelMesh GPU / parallel-computing demo
#
# One command for the teacher demo:   make demo
#
# Targets:
#   make help      - this overview (default)
#   make all       - build every binary the available toolchain allows
#   make cuda      - build cuda_flows      (nvcc, runs on the GPU)
#   make openmp    - build openmp_flows    (gcc -fopenmp, multicore CPU)
#   make mpi       - build mpi_flows       (mpicc, distributed memory)
#   make py        - check the Python GPU stack (torch + CUDA)
#   make run-cuda  - run the CUDA demo
#   make run-openmp- run the OpenMP demo
#   make run-mpi   - run the MPI demo (NPROC ranks)
#   make run-py    - run the PyTorch CPU-vs-GPU benchmark on UNSW-NB15
#   make demo      - BUILD + RUN everything in sequence (teacher demo)
#   make clean     - remove binaries
# ============================================================================

CXX        := nvcc
NVCC_ARCH  := -arch=native
NVCC_FLAGS := -O3 $(NVCC_ARCH)
CFLAGS     := -O3 -fopenmp -Wall
MPI_FLAGS  := -O3 -Wall
PY         ?= python3

CUDA_BIN   := cuda_flows
OPENMP_BIN := openmp_flows
MPI_BIN    := mpi_flows

# Dataset used by every track (same file the Go simulator reads)
DATA ?= ../data/raw/UNSW_NB15_testing-set.csv

# Real project data, exported through ml-crosscheck's FlowDataset loader into
# a flat binary the C/CUDA/MPI demos read (real flows, benign centroid, labels).
flows.bin: export_features.py
	@if [ ! -f flows.bin ] || [ $(DATA) -nt flows.bin ]; then \
		$(PY) export_features.py --data $(DATA); \
	else \
		echo "[data] flows.bin up to date"; \
	fi

data: flows.bin

# MPI ranks for the demo (this machine has 20 logical cores)
NPROC ?= 8

# Toolchain availability checks (demo degrades gracefully if MPI is missing)
NVCC_OK  := $(shell command -v $(CXX) 2>/dev/null)
GCC_OK   := $(shell command -v gcc 2>/dev/null)
MPICC_OK := $(shell command -v mpicc 2>/dev/null)

.PHONY: help all cuda openmp mpi py data run-cuda run-openmp run-mpi run-py crosscheck-gpu demo clean

help:
	@echo "SentinelMesh parallel/GPU demo — targets:"
	@echo "  make all        build everything the toolchain allows"
	@echo "  make cuda       build the CUDA GPU demo"
	@echo "  make openmp     build the OpenMP CPU demo"
	@echo "  make mpi        build the MPI demo (skipped if mpicc missing)"
	@echo "  make py         verify PyTorch + CUDA are usable"
	@echo "  make data       export real UNSW-NB15 flows -> flows.bin (all C demos read it)"
	@echo "  make run-cuda   run the CUDA demo (real flows)"
	@echo "  make run-openmp run the OpenMP demo (real flows)"
	@echo "  make run-mpi    run the MPI demo with $(NPROC) ranks (real flows)"
	@echo "  make run-py     run the PRODUCTION AutoencoderScorer CPU-vs-GPU"
	@echo "  make crosscheck-gpu  run the production train.py pipeline -> results/crosscheck/"
	@echo "  make demo       BUILD + RUN the whole sequence (teacher demo)"
	@echo "  make clean      remove built binaries and flows.bin"

# ---- build ----------------------------------------------------------------
all: cuda openmp mpi

cuda:
ifdef NVCC_OK
	$(CXX) $(NVCC_FLAGS) cuda_flows.cu -o $(CUDA_BIN)
	@echo "[built] $(CUDA_BIN)"
else
	@echo "[skip] nvcc not found — install the CUDA toolkit to build the GPU demo"
endif

openmp:
ifdef GCC_OK
	gcc $(CFLAGS) openmp_flows.c -o $(OPENMP_BIN) -lm
	@echo "[built] $(OPENMP_BIN)"
else
	@echo "[skip] gcc not found"
endif

mpi:
ifdef MPICC_OK
	mpicc $(MPI_FLAGS) mpi_flows.c -o $(MPI_BIN) -lm
	@echo "[built] $(MPI_BIN)"
else
	@echo "[skip] mpicc not found — install OpenMPI:  sudo pacman -S openmpi  (or: apt install openmpi-bin libopenmpi-dev)"
endif

py:
	$(PY) -c "import torch; assert torch.cuda.is_available(), 'CUDA unavailable'; print('PyTorch', torch.__version__, '| GPU:', torch.cuda.get_device_name(0))"

# ---- run -------------------------------------------------------------------
run-cuda: cuda flows.bin
ifdef NVCC_OK
	./$(CUDA_BIN)
else
	@echo "[skip] $(CUDA_BIN) not built (no nvcc)"
endif

run-openmp: openmp flows.bin
ifdef GCC_OK
	./$(OPENMP_BIN)
endif

# --allow-run-as-root: only needed when running as root (containers/WSL)
RUN_AS_ROOT := $(shell [ "`id -u`" = "0" ] && echo --allow-run-as-root || true)

run-mpi: mpi flows.bin
ifdef MPICC_OK
	mpirun $(RUN_AS_ROOT) -np $(NPROC) ./$(MPI_BIN)
else
	@echo "[skip] $(MPI_BIN) not built (no mpicc)"
endif

run-py: py
	$(PY) gpu_benchmark.py --data $(DATA)

# Run the PRODUCTION crosscheck pipeline (train.py) — writes the project's
# own artifacts to results/crosscheck/ (summary.csv, report.json, ...).
# This is the real pipeline the dashboard reads, accelerated by the GPU
# auto-selected inside the shipped AutoencoderScorer.
crosscheck-gpu: py
	$(PY) -c "import sys; sys.path.insert(0, '../ml-crosscheck/src'); \
		from train import generate_crosscheck_report, print_results; \
		results = generate_crosscheck_report('$(DATA)'); \
		print_results(results); \
		print('wrote results/crosscheck/{summary.csv,per_category.csv,per_flow_scores.csv,report.json}')"

# ---- the one-command teacher demo ------------------------------------------
demo: all
	@echo ""
	@echo "############################################################"
	@echo "#  1/5  OpenMP — the REAL UNSW-NB15 flows on CPU cores     #"
	@echo "############################################################"
	@$(MAKE) --no-print-directory run-openmp
	@echo ""
	@echo "############################################################"
	@echo "#  2/5  CUDA — the same real flows on the GPU              #"
	@echo "############################################################"
	@$(MAKE) --no-print-directory run-cuda
	@echo ""
	@echo "############################################################"
	@echo "#  3/5  MPI — the mesh itself: partitions of real flows    #"
	@echo "############################################################"
	@$(MAKE) --no-print-directory run-mpi
	@echo ""
	@echo "############################################################"
	@echo "#  4/5  PyTorch — the production AutoencoderScorer on GPU  #"
	@echo "############################################################"
	@$(MAKE) --no-print-directory run-py
	@echo ""
	@echo "############################################################"
	@echo "#  5/5  PRODUCTION crosscheck pipeline (writes results/)   #"
	@echo "############################################################"
	@$(MAKE) --no-print-directory crosscheck-gpu
	@echo ""
	@echo "Demo complete. See gpu_demo/README.md for the justification write-up."

clean:
	rm -f $(CUDA_BIN) $(OPENMP_BIN) $(MPI_BIN) flows.bin
	@echo "[clean] removed binaries and flows.bin"
