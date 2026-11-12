// ============================================================================
// mpi_flows.c — SentinelMesh MPI demo (distributed memory) on REAL data
//
// Data:  the actual UNSW-NB15 flows exported by export_features.py (through
//        the project's FlowDataset loader) into flows.bin.
// Model: each MPI rank is a SentinelMesh edge node. Root holds the full
//        traffic matrix; every rank receives ONLY its own partition of real
//        flows (the node-partitioning of simulator/fragment.go), scores it
//        with the same benign-centroid distance math, and the quorum-style
//        aggregation happens purely by message passing.
//
// Build:  mpicc -O3 mpi_flows.c -o mpi_flows -lm
// Run:    mpirun --allow-run-as-root -np 8 ./mpi_flows     (OpenMPI, root)
//         mpirun -np 8 ./mpi_flows                         (MPICH / user)
// ============================================================================

#include <mpi.h>
#include <stdio.h>
#include <stdlib.h>
#include <math.h>
#include <stdint.h>
#include <string.h>

static float* load_flows_bin_root(const char* path, int* n_out, int* f_out,
                                  float** centroid_out)
{
    FILE* fh = fopen(path, "rb");
    if (!fh) {
        fprintf(stderr, "error: %s not found.\n"
                "  Run `make data` first (export_features.py) — it exports the real\n"
                "  UNSW-NB15 flows through the project's FlowDataset loader.\n", path);
        MPI_Abort(MPI_COMM_WORLD, 1);
    }
    char magic[4];
    int n = 0, f = 0;
    if (fread(magic, 1, 4, fh) != 4 || magic[0] != 'U' || magic[1] != 'N' ||
        fread(&n, sizeof(int), 1, fh) != 1 || fread(&f, sizeof(int), 1, fh) != 1 ||
        n <= 0 || f <= 0) {
        fprintf(stderr, "error: %s is not a valid flows.bin\n", path);
        MPI_Abort(MPI_COMM_WORLD, 1);
    }
    float* X = malloc((size_t)n * f * sizeof(float));
    float* c = malloc((size_t)f * sizeof(float));
    if (!X || !c) { fprintf(stderr, "malloc failed\n"); MPI_Abort(MPI_COMM_WORLD, 1); }
    if (fread(X, sizeof(float), (size_t)n * f, fh) != (size_t)n * f ||
        fread(c, sizeof(float), (size_t)f, fh) != (size_t)f) {
        fprintf(stderr, "error: truncated %s\n", path);
        MPI_Abort(MPI_COMM_WORLD, 1);
    }
    fclose(fh);
    *n_out = n; *f_out = f; *centroid_out = c;
    return X;
}

int main(int argc, char **argv)
{
    MPI_Init(&argc, &argv);
    int rank = 0, size = 1;
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);
    MPI_Comm_size(MPI_COMM_WORLD, &size);

    int n = 0, f = 0;
    float *all_flows = NULL, *centroid = NULL;
    int *sendcounts = NULL, *displs = NULL;
    double *rank_ms = NULL;

    /* Root loads the REAL dataset; every rank needs the centroid, which ships
       inside flows.bin — broadcast it so no rank re-derives anything. */
    if (rank == 0) {
        all_flows = load_flows_bin_root("flows.bin", &n, &f, &centroid);

        /* Emulate mesh traffic volume: tile the REAL flows x<scale> so the
           demo models what a larger mesh sees per round. */
        int scale = (argc > 1) ? atoi(argv[1]) : 250;
        if (scale < 1) scale = 1;
        if (scale > 1) {
            const int n0 = n;
            float* tiled = malloc((size_t)n0 * scale * f * sizeof(float));
            if (!tiled) MPI_Abort(MPI_COMM_WORLD, 1);
            for (int s = 0; s < scale; ++s)
                memcpy(tiled + (size_t)s * n0 * f, all_flows,
                       (size_t)n0 * f * sizeof(float));
            free(all_flows);
            all_flows = tiled;
            n = n0 * scale;
            printf("Emulation       : real flows x%d = %d flow-scorings (multi-round mesh traffic)\n",
                   scale, n);
        }

        sendcounts = malloc(size * sizeof(int));
        displs     = malloc(size * sizeof(int));
        rank_ms    = malloc(size * sizeof(double));
        if (!sendcounts || !displs || !rank_ms) MPI_Abort(MPI_COMM_WORLD, 1);
        const int base = n / size, rem = n % size;
        int off = 0;
        for (int r = 0; r < size; ++r) {
            const int rows = base + (r == size - 1 ? rem : 0);
            sendcounts[r] = rows * f;
            displs[r]     = off;
            off += rows * f;
        }
    }
    MPI_Bcast(&n, 1, MPI_INT, 0, MPI_COMM_WORLD);
    MPI_Bcast(&f, 1, MPI_INT, 0, MPI_COMM_WORLD);

    const int base = n / size, rem = n % size;
    const int my_rows = base + (rank == size - 1 ? rem : 0);

    float *recv   = malloc((size_t)my_rows * f * sizeof(float));
    float *scores = malloc((size_t)my_rows * sizeof(float));
    if (rank != 0) centroid = malloc((size_t)f * sizeof(float));
    if (!recv || !scores || !centroid) { fprintf(stderr, "rank %d: malloc failed\n", rank); MPI_Abort(MPI_COMM_WORLD, 1); }
    MPI_Bcast(centroid, f, MPI_FLOAT, 0, MPI_COMM_WORLD);

    MPI_Barrier(MPI_COMM_WORLD);
    const double t0 = MPI_Wtime();

    /* ---- phase 1: distribute the real flow partitions ---------------------- */
    MPI_Scatterv(all_flows, sendcounts, displs, MPI_FLOAT,
                 recv, my_rows * f, MPI_FLOAT, 0, MPI_COMM_WORLD);
    const double t_scatter = MPI_Wtime();

    /* ---- phase 2: each rank scores ONLY its own real flows ----------------- */
    double local_sum = 0.0;
    for (int i = 0; i < my_rows; ++i) {
        float acc = 0.0f;
        for (int j = 0; j < f; ++j) {
            const float d = recv[(size_t)i * f + j] - centroid[j];
            acc += d * d;
        }
        scores[i] = sqrtf(acc / (float)f);
        local_sum += scores[i];
    }
    const double my_ms = (MPI_Wtime() - t_scatter) * 1e3;

    /* ---- phase 3: quorum-style aggregation via message passing ------------- */
    double total_sum = 0.0;
    MPI_Reduce(&local_sum, &total_sum, 1, MPI_DOUBLE, MPI_SUM, 0, MPI_COMM_WORLD);
    const double t_reduce = MPI_Wtime();

    double compute_wall_ms = 0.0;
    MPI_Reduce(&my_ms, &compute_wall_ms, 1, MPI_DOUBLE, MPI_MAX, 0, MPI_COMM_WORLD);

    const double scatter_ms = (t_scatter - t0) * 1e3;
    const double reduce_ms  = (t_reduce - t_scatter - my_ms / 1e3) * 1e3;
    const double total_ms   = (t_reduce - t0) * 1e3;

    MPI_Gather(&my_ms, 1, MPI_DOUBLE, rank_ms, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD);

    if (rank == 0) {
        /* serial baseline on the root for comparison (same real flows) */
        const double s0 = MPI_Wtime();
        double ref_sum = 0.0;
        for (int i = 0; i < n; ++i) {
            float acc = 0.0f;
            for (int j = 0; j < f; ++j) {
                const float d = all_flows[(size_t)i * f + j] - centroid[j];
                acc += d * d;
            }
            ref_sum += sqrtf(acc / (float)f);
        }
        const double serial_ms = (MPI_Wtime() - s0) * 1e3;

        printf("=== SentinelMesh MPI demo: %d ranks, %d REAL flows x %d features ===\n",
               size, n, f);
        printf("Data source : flows.bin (exported via ml-crosscheck FlowDataset)\n");
        printf("phase breakdown (root view):\n");
        printf("  1. scatter  (move %.0f MB of real flows) : %8.1f ms\n",
               (double)n * f * sizeof(float) / 1e6, scatter_ms);
        printf("  2. scoring  (slowest of %d ranks)        : %8.1f ms   <- pure compute wall time\n",
               size, compute_wall_ms);
        printf("  3. reduce   (quorum aggregation)          : %8.1f ms\n", reduce_ms);
        printf("  end-to-end                                : %8.1f ms\n", total_ms);
        printf("serial baseline (1 process, same compute)   : %8.1f ms\n", serial_ms);
        printf("compute-only speedup with %d ranks          : %5.1fx (ideal %d.0x)\n",
               size, serial_ms / compute_wall_ms, size);
        printf("per-rank compute time (ms)                  :");
        for (int r = 0; r < size; ++r) printf(" %.1f", rank_ms[r]);
        printf("\nchecksum: reduced sum=%.3f serial sum=%.3f -> %s\n",
               total_sum, ref_sum,
               fabs(total_sum - ref_sum) < 1e-2 * (double)n ? "PASS" : "FAIL");
        printf("\n--- what to notice ---\n");
        printf("1. Each rank saw only 1/%d of the REAL flows — shared nothing, like\n", size);
        printf("   the mesh nodes. Coordination happened ONLY via MPI messages.\n");
        printf("2. MPI_Reduce is the 'quorum aggregation': partial results merge at root.\n");
        printf("3. Compare phases: moving raw flows vs computing on them. This ratio is\n");
        printf("   exactly why SentinelMesh nodes gossip constant-size digests instead\n");
        printf("   of raw flows — and why a centralized SIEM is structurally expensive.\n");
        printf("4. Compute speedup is sub-linear: ranks contend for shared memory\n");
        printf("   bandwidth on one machine; across real machines MPI scales further.\n");
    }

    free(recv); free(scores); free(centroid);
    if (rank == 0) { free(all_flows); free(sendcounts); free(displs); free(rank_ms); }
    MPI_Finalize();
    return 0;
}
