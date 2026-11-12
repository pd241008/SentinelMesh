// ============================================================================
// openmp_flows.c — SentinelMesh OpenMP demo (multicore CPU) on REAL data
//
// Data:  the actual UNSW-NB15 flows exported by export_features.py
//        (through the project's FlowDataset loader) into flows.bin.
// Work:  per-flow anomaly score = normalized L2 distance from the benign
//        centroid — the same math as the CUDA demo and simulator/scorer.go,
//        executed with OpenMP threads scaling 1 -> all cores.
//
// OpenMP constructs: #pragma omp parallel for, reduction, schedule,
// omp_set_num_threads, omp_get_max_threads, omp_get_wtime.
//
// Build:  gcc -O3 -fopenmp openmp_flows.c -o openmp_flows -lm
// Run:    ./openmp_flows        (reads flows.bin, exported by `make data`)
// ============================================================================

#include <stdio.h>
#include <stdlib.h>
#include <math.h>
#include <time.h>
#include <stdint.h>
#include <string.h>
#include <omp.h>

static double now_ms(void)
{
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return ts.tv_sec * 1e3 + ts.tv_nsec / 1e6;
}

/* Same loader as cuda_flows.cu — one shared real-data format for all demos. */
static float* load_flows_bin(const char* path, int* n_out, int* f_out,
                             float** centroid_out, int32_t** labels_out)
{
    FILE* fh = fopen(path, "rb");
    if (!fh) {
        fprintf(stderr, "error: %s not found.\n"
                "  Run `make data` first (export_features.py) — it exports the real\n"
                "  UNSW-NB15 flows through the project's FlowDataset loader.\n", path);
        exit(1);
    }
    char magic[4];
    int n = 0, f = 0;
    if (fread(magic, 1, 4, fh) != 4 || magic[0] != 'U' || magic[1] != 'N' ||
        fread(&n, sizeof(int), 1, fh) != 1 || fread(&f, sizeof(int), 1, fh) != 1 ||
        n <= 0 || f <= 0) {
        fprintf(stderr, "error: %s is not a valid flows.bin\n", path);
        exit(1);
    }
    float* X = malloc((size_t)n * f * sizeof(float));
    float* c = malloc((size_t)f * sizeof(float));
    int32_t* y = malloc((size_t)n * sizeof(int32_t));
    if (!X || !c || !y) { fprintf(stderr, "malloc failed\n"); exit(1); }
    if (fread(X, sizeof(float), (size_t)n * f, fh) != (size_t)n * f ||
        fread(c, sizeof(float), (size_t)f, fh) != (size_t)f ||
        fread(y, sizeof(int32_t), (size_t)n, fh) != (size_t)n) {
        fprintf(stderr, "error: truncated %s\n", path);
        exit(1);
    }
    fclose(fh);
    *n_out = n; *f_out = f; *centroid_out = c; *labels_out = y;
    return X;
}

int main(int argc, char** argv)
{
    int n = 0, f = 0;
    float *flows = NULL, *centroid = NULL;
    int32_t* labels = NULL;
    flows = load_flows_bin("flows.bin", &n, &f, &centroid, &labels);

    /* Emulate mesh traffic volume: tile the REAL flows x<scale>. Every row is
       still a genuine UNSW-NB15 flow; the mesh of <scale> nodes sees this
       much traffic per round, and per-thread timing needs enough work. */
    int scale = (argc > 1) ? atoi(argv[1]) : 250;
    if (scale < 1) scale = 1;
    if (scale > 1) {
        const int n0 = n;
        float* tiled = malloc((size_t)n0 * scale * f * sizeof(float));
        int32_t* lab = malloc((size_t)n0 * scale * sizeof(int32_t));
        if (!tiled || !lab) { fprintf(stderr, "malloc failed\n"); return 1; }
        for (int s = 0; s < scale; ++s) {
            memcpy(tiled + (size_t)s * n0 * f, flows, (size_t)n0 * f * sizeof(float));
            memcpy(lab + (size_t)s * n0, labels, (size_t)n0 * sizeof(int32_t));
        }
        free(flows); free(labels);
        flows = tiled; labels = lab; n = n0 * scale;
        printf("Emulation      : real flows x%d = %d flow-scorings (multi-round mesh traffic)\n",
               scale, n);
    }

    float* scores = malloc((size_t)n * sizeof(float));
    if (!scores) { fprintf(stderr, "malloc failed\n"); return 1; }

    printf("=== SentinelMesh OpenMP demo: REAL UNSW-NB15 flows, per-flow scoring ===\n");
    printf("Data source    : flows.bin (exported via ml-crosscheck FlowDataset)\n");
    printf("  flows        : %d x %d production feature columns\n", n, f);
    printf("CPUs available : %d logical cores (omp_get_num_procs)\n", omp_get_num_procs());

    // ---- serial baseline (1 core) -----------------------------------------
    const double t0 = now_ms();
    double ref_sum = 0.0;
    for (int i = 0; i < n; ++i) {
        float acc = 0.0f;
        for (int j = 0; j < f; ++j) {
            const float d = flows[(size_t)i * f + j] - centroid[j];
            acc += d * d;
        }
        scores[i] = sqrtf(acc / (float)f);
        ref_sum += scores[i];
    }
    const double serial_ms = now_ms() - t0;
    printf("[ 1 thread (serial)] %8.1f ms   (  1.0x baseline)\n", serial_ms);

    // ---- OpenMP thread scaling --------------------------------------------
    const int max_threads = omp_get_max_threads();
    int tlist[8];
    int n_t = 0;
    const int wanted[] = { 2, 4, 8, 16, 20 };
    for (int k = 0; k < 5; ++k)
        if (wanted[k] < max_threads) tlist[n_t++] = wanted[k];
    tlist[n_t++] = max_threads;

    int prev = -1;
    for (int k = 0; k < n_t; ++k) {
        const int t = tlist[k];
        if (t == prev) continue;
        prev = t;
        omp_set_num_threads(t);

        const double t1 = omp_get_wtime() * 1e3;
        double sum = 0.0;
        #pragma omp parallel for schedule(static) reduction(+:sum)
        for (int i = 0; i < n; ++i) {
            float acc = 0.0f;
            for (int j = 0; j < f; ++j) {
                const float d = flows[(size_t)i * f + j] - centroid[j];
                acc += d * d;
            }
            scores[i] = sqrtf(acc / (float)f);
            sum += scores[i];
        }
        const double ms = omp_get_wtime() * 1e3 - t1;

        const int ok = fabs(sum - ref_sum) < 1e-2 * (double)n;
        printf("[%2d threads]          %8.1f ms   (%5.1fx vs serial)  checksum %s\n",
               t, ms, serial_ms / ms, ok ? "OK" : "MISMATCH");
    }

    // ---- what the scores mean on the real data -----------------------------
    double s_att = 0.0, s_nrm = 0.0;
    int n_att = 0, n_nrm = 0;
    for (int i = 0; i < n; ++i) {
        if (labels[i]) { s_att += scores[i]; ++n_att; }
        else           { s_nrm += scores[i]; ++n_nrm; }
    }
    printf("\nreal-data signal: mean score attacks=%.4f (n=%d) vs normal=%.4f (n=%d)\n",
           n_att ? s_att / n_att : 0.0, n_att, n_nrm ? s_nrm / n_nrm : 0.0, n_nrm);

    printf("\n--- what to notice ---\n");
    printf("1. Real UNSW-NB15 flows (project loader), same math as the CUDA demo —\n");
    printf("   this is the CPU baseline that demo beats.\n");
    printf("2. OpenMP scales until cores saturate, then flattens (memory bandwidth\n");
    printf("   becomes the limit — the same wall the GPU coalescing demo shows).\n");
    printf("3. 'schedule(static)' splits flows into equal chunks; use\n");
    printf("   'schedule(dynamic)' when per-flow cost is irregular.\n");

    free(flows); free(scores); free(centroid); free(labels);
    return 0;
}
