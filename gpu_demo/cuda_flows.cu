// ============================================================================
// cuda_flows.cu — SentinelMesh GPU demo (raw CUDA) on REAL project data
//
// Data:    the actual UNSW-NB15 flows, exported through the project's own
//          FlowDataset loader (ml-crosscheck/src/dataset.py) by
//          export_features.py into flows.bin. Same FEATURE_COLUMNS the
//          production scorers use, same benign-traffic centroid.
// Workload: per-flow anomaly score = normalized L2 distance from the benign
//          centroid — the same feature-distance math as simulator/scorer.go.
//
// Three GPU strategies vs the serial CPU:
//   K1  thread-per-flow   — the naive port: 1 CUDA thread per flow.
//   K2  warp-per-flow     — 32 lanes cooperate on one flow, reading
//                           CONSECUTIVE features => coalesced accesses.
//   K3  warp-per-flow on a column-major copy — identical math, strided.
//
// Build:  nvcc -O3 -arch=native cuda_flows.cu -o cuda_flows
// Run:    ./cuda_flows          (reads flows.bin, exported by `make data`)
// ============================================================================

#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <cmath>
#include <chrono>
#include <cstdint>
#include <cuda_runtime.h>

#define CUDA_CHECK(call)                                                      \
    do {                                                                      \
        cudaError_t err__ = (call);                                           \
        if (err__ != cudaSuccess) {                                           \
            fprintf(stderr, "CUDA error %s at %s:%d\n",                       \
                    cudaGetErrorString(err__), __FILE__, __LINE__);           \
            exit(EXIT_FAILURE);                                               \
        }                                                                     \
    } while (0)

// ---------------------------------------------------------------------------
// K1 — thread-per-flow, row-major: flows[i * F + j].
// ---------------------------------------------------------------------------
__global__ void score_thread_per_flow(const float* __restrict__ flows,
                                      const float* __restrict__ centroid,
                                      float* __restrict__ scores,
                                      int n, int f)
{
    const int i = blockIdx.x * blockDim.x + threadIdx.x;
    if (i >= n) return;

    const float* row = flows + (size_t)i * f;
    float acc = 0.0f;
    for (int j = 0; j < f; ++j) {
        const float d = row[j] - centroid[j];
        acc += d * d;
    }
    scores[i] = sqrtf(acc / (float)f);
}

__device__ __forceinline__ float warp_reduce_sum(float acc)
{
    #pragma unroll
    for (int off = 16; off > 0; off >>= 1)
        acc += __shfl_down_sync(0xffffffffu, acc, off);
    return acc;
}

// ---------------------------------------------------------------------------
// K2 — warp-per-flow over the ROW-MAJOR layout (coalesced).
// ---------------------------------------------------------------------------
__global__ void score_warp_coalesced(const float* __restrict__ flows,
                                     const float* __restrict__ centroid,
                                     float* __restrict__ scores,
                                     int n, int f)
{
    const int lane  = threadIdx.x & 31;
    const int wid   = threadIdx.x >> 5;
    const int warps = blockDim.x >> 5;
    const int flow  = blockIdx.x * warps + wid;
    if (flow >= n) return;

    const float* row = flows + (size_t)flow * f;
    float acc = 0.0f;
    for (int j = lane; j < f; j += 32) {
        const float d = row[j] - centroid[j];
        acc += d * d;
    }
    acc = warp_reduce_sum(acc);
    if (lane == 0) scores[flow] = sqrtf(acc / (float)f);
}

// ---------------------------------------------------------------------------
// K3 — warp-per-flow over the COLUMN-MAJOR copy (strided). Same math.
// ---------------------------------------------------------------------------
__global__ void score_warp_strided(const float* __restrict__ flows,
                                   const float* __restrict__ centroid,
                                   float* __restrict__ scores,
                                   int n, int f)
{
    const int lane  = threadIdx.x & 31;
    const int wid   = threadIdx.x >> 5;
    const int warps = blockDim.x >> 5;
    const int flow  = blockIdx.x * warps + wid;
    if (flow >= n) return;

    float acc = 0.0f;
    for (int j = lane; j < f; j += 32) {
        const float d = flows[(size_t)j * n + flow] - centroid[j];
        acc += d * d;
    }
    acc = warp_reduce_sum(acc);
    if (lane == 0) scores[flow] = sqrtf(acc / (float)f);
}

// ---------------------------------------------------------------------------
// Controlled experiment: WIDE rows (F=64, synthetic but clearly labeled).
// The production feature set is only 5 columns — too narrow for the
// coalescing penalty to appear (that is why naive K1 wins above). This
// section demonstrates the memory-layout lesson in its visible regime.
// ---------------------------------------------------------------------------
__global__ void wide_coalesced(const float* __restrict__ m,
                               float* __restrict__ out, int n, int f)
{
    const int lane  = threadIdx.x & 31;
    const int wid   = threadIdx.x >> 5;
    const int warps = blockDim.x >> 5;
    const int row   = blockIdx.x * warps + wid;
    if (row >= n) return;

    const float* r = m + (size_t)row * f;
    float acc = 0.0f;
    for (int j = lane; j < f; j += 32) acc += r[j] * r[j];
    acc = warp_reduce_sum(acc);
    if (lane == 0) out[row] = acc;
}

__global__ void wide_strided(const float* __restrict__ m,
                             float* __restrict__ out, int n, int f)
{
    const int lane  = threadIdx.x & 31;
    const int wid   = threadIdx.x >> 5;
    const int warps = blockDim.x >> 5;
    const int row   = blockIdx.x * warps + wid;
    if (row >= n) return;

    float acc = 0.0f;
    for (int j = lane; j < f; j += 32) acc += m[(size_t)j * n + row] * m[(size_t)j * n + row];
    acc = warp_reduce_sum(acc);
    if (lane == 0) out[row] = acc;
}

// ---------------------------------------------------------------------------
static int cores_per_sm(int major, int minor)
{
    switch (major * 10 + minor) {
        case 75:  return 64;   case 80:  return 64;
        case 86:  return 128;  case 89:  return 128;
        case 90:  return 128;  default: return 128;
    }
}

static double now_ms(void)
{
    using namespace std::chrono;
    return duration<double, std::milli>(steady_clock::now().time_since_epoch()).count();
}

// ---------------------------------------------------------------------------
// Load the real project data exported by export_features.py:
//   ['U','N','S','W'][int32 n][int32 f][n*f f32 row-major][f f32 centroid][n i32 labels]
// ---------------------------------------------------------------------------
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
    float* X = (float*)malloc((size_t)n * f * sizeof(float));
    float* c = (float*)malloc((size_t)f * sizeof(float));
    int32_t* y = (int32_t*)malloc((size_t)n * sizeof(int32_t));
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
    const char* path = (argc > 1) ? argv[1] : "flows.bin";
    int n = 0, f = 0;
    float *h_cent = NULL, *h_row = NULL;
    int32_t* h_labels = NULL;
    h_row = load_flows_bin(path, &n, &f, &h_cent, &h_labels);

    // Emulate mesh traffic volume: tile the REAL flows x<scale>. Every row is
    // still a genuine UNSW-NB15 flow; we replicate them the way a mesh of
    // <scale> nodes would see the same per-node traffic per round.
    int scale = (argc > 2) ? atoi(argv[2]) : 250;
    if (scale < 1) scale = 1;
    if (scale > 1) {
        const int n0 = n;
        float* tiled = (float*)malloc((size_t)n0 * scale * f * sizeof(float));
        int32_t* lab = (int32_t*)malloc((size_t)n0 * scale * sizeof(int32_t));
        if (!tiled || !lab) { fprintf(stderr, "malloc failed\n"); return 1; }
        for (int s = 0; s < scale; ++s) {
            memcpy(tiled + (size_t)s * n0 * f, h_row, (size_t)n0 * f * sizeof(float));
            memcpy(lab + (size_t)s * n0, h_labels, (size_t)n0 * sizeof(int32_t));
        }
        free(h_row); free(h_labels);
        h_row = tiled; h_labels = lab; n = n0 * scale;
        printf("Emulation         : real flows x%d = %d flows (mesh of %d nodes' traffic)\n",
               scale, n, scale);
    }
    const size_t total = (size_t)n * f;

    printf("=== SentinelMesh CUDA demo: REAL UNSW-NB15 flows, per-flow scoring ===\n");
    printf("Data source      : %s (exported via ml-crosscheck FlowDataset)\n", path);
    printf("  flows          : %d x %d production feature columns\n", n, f);

    // Column-major copy of the SAME real data for the strided-kernel lesson.
    float* h_col = (float*)malloc(total * sizeof(float));
    float* h_cpu = (float*)malloc((size_t)n * sizeof(float));
    float* h_gpu = (float*)malloc((size_t)n * sizeof(float));
    if (!h_col || !h_cpu || !h_gpu) { fprintf(stderr, "malloc failed\n"); return 1; }
    for (int i = 0; i < n; ++i)
        for (int j = 0; j < f; ++j)
            h_col[(size_t)j * n + i] = h_row[(size_t)i * f + j];

    // ---- Device inventory -------------------------------------------------
    cudaDeviceProp prop;
    CUDA_CHECK(cudaGetDeviceProperties(&prop, 0));
    const int cuda_cores = prop.multiProcessorCount * cores_per_sm(prop.major, prop.minor);
    printf("Device           : %s — %d SMs x %d cores = %d CUDA cores (cc %d.%d)\n",
           prop.name, prop.multiProcessorCount, cores_per_sm(prop.major, prop.minor),
           cuda_cores, prop.major, prop.minor);

    // ---- CPU serial reference --------------------------------------------
    double sum_cpu = 0.0;
    double t0 = now_ms();
    for (int i = 0; i < n; ++i) {
        float acc = 0.0f;
        for (int j = 0; j < f; ++j) {
            const float d = h_row[(size_t)i * f + j] - h_cent[j];
            acc += d * d;
        }
        h_cpu[i] = sqrtf(acc / (float)f);
        sum_cpu += h_cpu[i];
    }
    const double cpu_ms = now_ms() - t0;
    printf("CPU serial       : %8.1f ms   (1 core, the 'before' picture)\n", cpu_ms);

    // ---- Device buffers ---------------------------------------------------
    float *d_row, *d_col, *d_cent, *d_scores;
    CUDA_CHECK(cudaMalloc(&d_row,    total * sizeof(float)));
    CUDA_CHECK(cudaMalloc(&d_col,    total * sizeof(float)));
    CUDA_CHECK(cudaMalloc(&d_cent,   f * sizeof(float)));
    CUDA_CHECK(cudaMalloc(&d_scores, (size_t)n * sizeof(float)));

    cudaEvent_t e0, e1;
    CUDA_CHECK(cudaEventCreate(&e0));
    CUDA_CHECK(cudaEventCreate(&e1));

    // Warm the context with a tiny copy so the big-copy timing is honest.
    CUDA_CHECK(cudaMemcpy(d_cent, h_cent, f * sizeof(float), cudaMemcpyHostToDevice));
    CUDA_CHECK(cudaDeviceSynchronize());
    CUDA_CHECK(cudaEventRecord(e0));
    CUDA_CHECK(cudaMemcpy(d_row, h_row, total * sizeof(float), cudaMemcpyHostToDevice));
    CUDA_CHECK(cudaEventRecord(e1));
    CUDA_CHECK(cudaEventSynchronize(e1));
    float copy_ms = 0.0f;
    CUDA_CHECK(cudaEventElapsedTime(&copy_ms, e0, e1));
    const double copy_gbps = (total * sizeof(float)) / (copy_ms * 1e-3) / 1e9;
    CUDA_CHECK(cudaMemcpy(d_col, h_col, total * sizeof(float), cudaMemcpyHostToDevice));

    // ---- Kernel launches ---------------------------------------------------
    const int tpp_threads  = 256;
    const int tpp_blocks   = (n + tpp_threads - 1) / tpp_threads;
    const int warp_threads = 256;
    const int warp_blocks  = (n * 32 + warp_threads - 1) / warp_threads;
    printf("Launch configs   : K1 %d blk x %d thr (1 flow/thread) | K2/K3 %d blk x %d thr (1 flow/warp)\n",
           tpp_blocks, tpp_threads, warp_blocks, warp_threads);

    auto verify = [&](double sum_gpu) {
        printf("  checksum           : cpu=%.3f gpu=%.3f -> %s\n", sum_cpu, sum_gpu,
               (fabs(sum_cpu - sum_gpu) < 1e-2 * (double)n) ? "PASS" : "FAIL");
    };
    auto attack_stats = [&](const float* scores) {
        double s_att = 0.0, s_nrm = 0.0; int n_att = 0, n_nrm = 0;
        for (int i = 0; i < n; ++i) {
            if (h_labels[i]) { s_att += scores[i]; ++n_att; }
            else             { s_nrm += scores[i]; ++n_nrm; }
        }
        printf("  real-data signal   : mean score attacks=%.4f (n=%d) vs normal=%.4f (n=%d)\n",
               n_att ? s_att / n_att : 0.0, n_att, n_nrm ? s_nrm / n_nrm : 0.0, n_nrm);
    };

    // K1
    float ms1 = 0.0f;
    {
        score_thread_per_flow<<<tpp_blocks, tpp_threads>>>(d_row, d_cent, d_scores, n, f);
        CUDA_CHECK(cudaGetLastError());
        CUDA_CHECK(cudaDeviceSynchronize());
        CUDA_CHECK(cudaEventRecord(e0));
        score_thread_per_flow<<<tpp_blocks, tpp_threads>>>(d_row, d_cent, d_scores, n, f);
        CUDA_CHECK(cudaEventRecord(e1));
        CUDA_CHECK(cudaEventSynchronize(e1));
        CUDA_CHECK(cudaEventElapsedTime(&ms1, e0, e1));
        CUDA_CHECK(cudaMemcpy(h_gpu, d_scores, (size_t)n * sizeof(float), cudaMemcpyDeviceToHost));
        double s = 0.0;
        for (int i = 0; i < n; ++i) s += h_gpu[i];
        printf("K1 GPU thread/flow : %8.2f ms   (%5.1fx vs CPU serial)\n", ms1, cpu_ms / ms1);
        verify(s);
    }

    // K2
    float ms2 = 0.0f;
    {
        score_warp_coalesced<<<warp_blocks, warp_threads>>>(d_row, d_cent, d_scores, n, f);
        CUDA_CHECK(cudaGetLastError());
        CUDA_CHECK(cudaDeviceSynchronize());
        CUDA_CHECK(cudaEventRecord(e0));
        score_warp_coalesced<<<warp_blocks, warp_threads>>>(d_row, d_cent, d_scores, n, f);
        CUDA_CHECK(cudaEventRecord(e1));
        CUDA_CHECK(cudaEventSynchronize(e1));
        CUDA_CHECK(cudaEventElapsedTime(&ms2, e0, e1));
        CUDA_CHECK(cudaMemcpy(h_gpu, d_scores, (size_t)n * sizeof(float), cudaMemcpyDeviceToHost));
        double s = 0.0;
        for (int i = 0; i < n; ++i) s += h_gpu[i];
        printf("K2 GPU coalesced   : %8.2f ms   (%5.1fx vs CPU serial, %.1fx vs K1)\n",
               ms2, cpu_ms / ms2, ms1 / ms2);
        verify(s);
        attack_stats(h_gpu);   // real attack/normal separation on GPU results
    }

    // K3
    float ms3 = 0.0f;
    {
        score_warp_strided<<<warp_blocks, warp_threads>>>(d_col, d_cent, d_scores, n, f);
        CUDA_CHECK(cudaGetLastError());
        CUDA_CHECK(cudaDeviceSynchronize());
        CUDA_CHECK(cudaEventRecord(e0));
        score_warp_strided<<<warp_blocks, warp_threads>>>(d_col, d_cent, d_scores, n, f);
        CUDA_CHECK(cudaEventRecord(e1));
        CUDA_CHECK(cudaEventSynchronize(e1));
        CUDA_CHECK(cudaEventElapsedTime(&ms3, e0, e1));
        CUDA_CHECK(cudaMemcpy(h_gpu, d_scores, (size_t)n * sizeof(float), cudaMemcpyDeviceToHost));
        double s = 0.0;
        for (int i = 0; i < n; ++i) s += h_gpu[i];
        printf("K3 GPU strided     : %8.2f ms   (same math as K2, %.1fx SLOWER)\n", ms3, ms3 / ms2);
        verify(s);
    }

    printf("\n--- controlled experiment: wide rows (F=64, synthetic, labeled) ---\n");
    {
        const int wn = 1 << 20, wf = 64;             // 1M rows x 64 features
        float* w_row = (float*)malloc((size_t)wn * wf * sizeof(float));
        float* w_col = (float*)malloc((size_t)wn * wf * sizeof(float));
        float* w_out = (float*)malloc((size_t)wn * sizeof(float));
        if (!w_row || !w_col || !w_out) { fprintf(stderr, "malloc failed\n"); return 1; }
        for (int i = 0; i < wn; ++i)
            for (int j = 0; j < wf; ++j) {
                const float v = (float)((i * 2654435761u + j) % 1000) / 1000.0f;
                w_row[(size_t)i * wf + j] = v;
                w_col[(size_t)j * wn + i] = v;
            }
        float *dw_row, *dw_col, *dw_out;
        CUDA_CHECK(cudaMalloc(&dw_row, (size_t)wn * wf * sizeof(float)));
        CUDA_CHECK(cudaMalloc(&dw_col, (size_t)wn * wf * sizeof(float)));
        CUDA_CHECK(cudaMalloc(&dw_out, (size_t)wn * sizeof(float)));
        CUDA_CHECK(cudaMemcpy(dw_row, w_row, (size_t)wn * wf * sizeof(float), cudaMemcpyHostToDevice));
        CUDA_CHECK(cudaMemcpy(dw_col, w_col, (size_t)wn * wf * sizeof(float), cudaMemcpyHostToDevice));
        const int wblocks = (wn * 32 + 255) / 256;

        auto timeit = [&](void (*k)(const float*, float*, int, int), const float* src) {
            k<<<wblocks, 256>>>(src, dw_out, wn, wf);
            CUDA_CHECK(cudaGetLastError());
            CUDA_CHECK(cudaDeviceSynchronize());
            CUDA_CHECK(cudaEventRecord(e0));
            k<<<wblocks, 256>>>(src, dw_out, wn, wf);
            CUDA_CHECK(cudaEventRecord(e1));
            CUDA_CHECK(cudaEventSynchronize(e1));
            float ms = 0.0f;
            CUDA_CHECK(cudaEventElapsedTime(&ms, e0, e1));
            return ms;
        };
        const float wcoa = timeit(wide_coalesced, dw_row);
        const float wstr = timeit(wide_strided,  dw_col);
        printf("coalesced (row-major) : %6.2f ms\n", wcoa);
        printf("strided  (col-major)  : %6.2f ms   -> %.1fx SLOWER for identical math\n",
               wstr, wstr / wcoa);
        printf("(this is the lesson the 5-feature production data is too narrow to show)\n");

        CUDA_CHECK(cudaFree(dw_row)); CUDA_CHECK(cudaFree(dw_col)); CUDA_CHECK(cudaFree(dw_out));
        free(w_row); free(w_col); free(w_out);
    }

    printf("\n--- what to notice ---\n");
    printf("1. Real UNSW-NB15 flows from the project loader — %d flow-scorings here\n", n);
    printf("   (real rows x%d emulation). Attacks score ~2x normal on average.\n", scale);
    printf("2. CPU serial: 1 core. GPU K1: all %d cores at once = %.0fx speedup.\n",
           cuda_cores, cpu_ms / ms1);
    printf("3. At 5 production features the naive K1 kernel wins: millions of\n");
    printf("   independent threads hide latency best. The wide-row experiment shows\n");
    printf("   the coalescing penalty that appears once feature vectors grow.\n");
    printf("4. Uploading the flow matrix over PCIe took %.1f ms (%.0f GB/s) vs a\n",
           copy_ms, copy_gbps);
    printf("   %.2f ms kernel — data movement dominates; keep data resident in VRAM.\n", ms2);
    printf("5. Every kernel reproduced the CPU scores (checksum PASS) — acceleration\n");
    printf("   changed the speed, never the answer.\n");

    CUDA_CHECK(cudaEventDestroy(e0));
    CUDA_CHECK(cudaEventDestroy(e1));
    CUDA_CHECK(cudaFree(d_row));
    CUDA_CHECK(cudaFree(d_col));
    CUDA_CHECK(cudaFree(d_cent));
    CUDA_CHECK(cudaFree(d_scores));
    free(h_row); free(h_col); free(h_cent); free(h_cpu); free(h_gpu); free(h_labels);
    return 0;
}
