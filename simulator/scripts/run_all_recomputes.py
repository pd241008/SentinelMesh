import os
import subprocess
import pandas as pd

def run_cmd(cmd):
    print(f"Running: {cmd}")
    subprocess.run(cmd, shell=True, check=True, cwd="simulator")

# 1. Canonical Table I (N=32/f=3/k=8, all q)
# We can use test_stagger.yaml (which is exactly N=32, f=3, k=8, W=5)
run_cmd("go run cmd/simulate/main.go -config configs/test_stagger.yaml -data ../data/raw/UNSW_NB15_testing-set.csv -output ../results/canonical_table -seed 42")
print("\n--- Canonical Table I (N=32, f=3, k=8) ---")
df1 = pd.read_csv("results/canonical_table/sweep_results.csv")
print(df1[['q', 'gossip_corrected_flow_recon_recall', 'gossip_corrected_flow_dos_recall', 'gossip_latency']].to_string(index=False))

# 2. Task 2 Crossover Grid (N=64, all k x f)
# Create a config
with open("simulator/configs/task2.yaml", "w") as f:
    f.write("""sweep:
  N: [64]
  f: [2, 3, 4]
  k: [4, 8, 16]
  W: 5
  thresholds:
    threshold_recon: 1.66
    threshold_dos: 0.82
    threshold_ewma: 0.89
""")
run_cmd("go run cmd/simulate/main.go -config configs/task2.yaml -data ../data/raw/UNSW_NB15_testing-set.csv -output ../results/task2 -seed 42")
print("\n--- Task 2 Crossover Grid (N=64) ---")
df2 = pd.read_csv("results/task2/sweep_results.csv")
df2_k8 = df2[df2['k'] == 8]
print(df2_k8[['k', 'f', 'q', 'gossip_corrected_flow_dos_recall']].to_string(index=False))

# 3. Task 1 std-dev table (seeds 42, 43, 44 for N=32, f=3, k=8)
for seed in [42, 43, 44]:
    run_cmd(f"go run cmd/simulate/main.go -config configs/test_stagger.yaml -data ../data/raw/UNSW_NB15_testing-set.csv -output ../results/task1_{seed} -seed {seed}")
df3_42 = pd.read_csv("results/task1_42/sweep_results.csv")
df3_43 = pd.read_csv("results/task1_43/sweep_results.csv")
df3_44 = pd.read_csv("results/task1_44/sweep_results.csv")
print("\n--- Task 1 Std-Dev Table ---")
# compute std-dev across the 3 seeds for Recon and DoS
for q_idx in range(len(df3_42)):
    q = df3_42['q'].iloc[q_idx]
    recon_vals = [df3_42['gossip_corrected_flow_recon_recall'].iloc[q_idx], df3_43['gossip_corrected_flow_recon_recall'].iloc[q_idx], df3_44['gossip_corrected_flow_recon_recall'].iloc[q_idx]]
    dos_vals = [df3_42['gossip_corrected_flow_dos_recall'].iloc[q_idx], df3_43['gossip_corrected_flow_dos_recall'].iloc[q_idx], df3_44['gossip_corrected_flow_dos_recall'].iloc[q_idx]]
    import statistics
    print(f"q={q}: Recon = {statistics.mean(recon_vals):.4f} +/- {statistics.stdev(recon_vals):.4f} | DoS = {statistics.mean(dos_vals):.4f} +/- {statistics.stdev(dos_vals):.4f}")

# 4. Task 4 Analysis category (N=32, k=16, f=4, q=8)
# We can use sanity_k16f4.yaml
run_cmd("go run cmd/simulate/main.go -config configs/sanity_k16f4.yaml -data ../data/raw/UNSW_NB15_testing-set.csv -output ../results/task4 -seed 42 > ../results/task4.log 2>&1")
print("\n--- Task 4 Analysis/EWMA ---")
run_cmd("grep -A 3 'k=16 → q ∈' ../results/task4.log")

# 5. Task 5 Clustered Evasion (N=32, k=8)
run_cmd("go run cmd/simulate/main.go -config configs/test_stagger.yaml -data ../data/raw/UNSW_NB15_testing-set.csv -output ../results/task5_clustered -seed 42 -clustered > ../results/task5.log 2>&1")
print("\n--- Task 5 Clustered ---")
run_cmd("grep -A 3 'k=8 → q ∈' ../results/task5.log")

