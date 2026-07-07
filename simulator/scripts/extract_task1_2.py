import pandas as pd
import sys

def main():
    try:
        df = pd.read_csv('/root/workspace/workspace/03-Code/Projects/Legacy/SentinalMesh/results/full_grid/master_grid.csv')
    except Exception as e:
        print(f"Error reading CSV: {e}")
        return

    print("=== TASK 1: Canonical Recall & FPR Table (N=32, f=3, k=8) ===")
    t1 = df[(df['N']==32) & (df['f']==3) & (df['k']==8)].sort_values('q')
    for _, row in t1.iterrows():
        q = int(row['q'])
        r_mean, r_std = row['gossip_corrected_flow_recon_recall_mean'], row['gossip_corrected_flow_recon_recall_std']
        d_mean, d_std = row['gossip_corrected_flow_dos_recall_mean'], row['gossip_corrected_flow_dos_recall_std']
        rfpr_mean, rfpr_std = row['gossip_recon_fpr_mean'], row['gossip_recon_fpr_std']
        dfpr_mean, dfpr_std = row['gossip_dos_fpr_mean'], row['gossip_dos_fpr_std']
        print(f"q={q}: Recon = {r_mean*100:.2f}% ± {r_std*100:.2f}%, DoS = {d_mean*100:.2f}% ± {d_std*100:.2f}% | "
              f"Recon FPR = {rfpr_mean*100:.2f}% ± {rfpr_std*100:.2f}%, DoS FPR = {dfpr_mean*100:.2f}% ± {dfpr_std*100:.2f}%")

    print("\n=== TASK 2: Fanout-Fragmentation Crossover (N=64, q=4) ===")
    t2 = df[(df['N']==64) & (df['q']==4)].sort_values(['k', 'f'])
    for _, row in t2.iterrows():
        k, f = int(row['k']), int(row['f'])
        r_uncorr = row['gossip_flow_recon_recall_mean']
        d_uncorr = row['gossip_flow_dos_recall_mean']
        r_corr = row['gossip_corrected_flow_recon_recall_mean']
        d_corr = row['gossip_corrected_flow_dos_recall_mean']
        gap_uncorr = r_uncorr - d_uncorr
        gap_corr = r_corr - d_corr
        
        print(f"k={k}, f={f}:")
        print(f"  Uncorrected: Recon={r_uncorr*100:.2f}%, DoS={d_uncorr*100:.2f}%, Gap={gap_uncorr*100:+.2f}%")
        print(f"  Corrected:   Recon={r_corr*100:.2f}%, DoS={d_corr*100:.2f}%, Gap={gap_corr*100:+.2f}%")

if __name__ == '__main__':
    main()
