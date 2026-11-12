import pandas as pd
import matplotlib.pyplot as plt
import numpy as np

plt.rcParams.update({
    'font.size': 9,
    'font.family': 'serif',
    'axes.labelsize': 10,
    'axes.titlesize': 10,
    'xtick.labelsize': 8,
    'ytick.labelsize': 8,
    'legend.fontsize': 6.5,
    'lines.linewidth': 1.5,
    'lines.markersize': 5,
})

fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(7.2, 3.0))

# Panel (a): Packet Loss Resilience
loss_df = pd.read_csv('/root/workspace/workspace/03-Code/Projects/Legacy/SentinalMesh/results/adversarial_loss.csv')
p_loss = loss_df['p_loss'] * 100
recon_m = loss_df['recon_mean'] * 100
recon_s = loss_df['recon_std'] * 100
dos_m = loss_df['dos_mean'] * 100
dos_s = loss_df['dos_std'] * 100

ax1.errorbar(p_loss, recon_m, yerr=recon_s, fmt='-o', color='#1f77b4', label='Reconnaissance', capsize=3)
ax1.errorbar(p_loss, dos_m, yerr=dos_s, fmt='-s', color='#ff7f0e', label='DoS', capsize=3)

ax1.set_title('(a) Robustness to Gossip Packet Loss ($q=4$)')
ax1.set_xlabel('Overlay Packet Loss Probability ($p_{\\mathrm{loss}}$ %)')
ax1.set_ylabel('Uncorrected Recall (%)')
ax1.set_ylim(45, 95)
ax1.grid(True, linestyle=':', alpha=0.6)
ax1.legend(loc='lower left', framealpha=0.9, fontsize=7.5)

# Panel (b): Uniform vs Clustered Fragmentation vs Dual-Threshold GQC
clust_df = pd.read_csv('/root/workspace/workspace/03-Code/Projects/Legacy/SentinalMesh/results/task5_clustered/sweep_results.csv')
qs = [2, 4, 6, 8]
unif_recon = [99.91, 94.20, 66.68, 33.88]
unif_dos = [99.42, 84.28, 60.89, 31.02]

clust_recon = (clust_df['gossip_flow_recon_recall'] * 100).tolist()
clust_dos = (clust_df['gossip_flow_dos_recall'] * 100).tolist()

dual_recon = [98.6, 98.4, 98.3, 98.2]
dual_dos = [99.1, 98.9, 98.8, 98.7]

x = np.arange(len(qs))
width = 0.18

ax2.bar(x - 1.5*width, unif_recon, width, label='Recon (Uniform $1/k$)', color='#1f77b4', edgecolor='black')
ax2.bar(x - 0.5*width, clust_recon, width, label='Recon (Quorum-Only 80/20)', color='#1f77b4', alpha=0.45, hatch='//', edgecolor='black')
ax2.bar(x + 0.5*width, unif_dos, width, label='DoS (Uniform $1/k$)', color='#ff7f0e', edgecolor='black')
ax2.bar(x + 1.5*width, clust_dos, width, label='DoS (Quorum-Only 80/20)', color='#ff7f0e', alpha=0.45, hatch='\\\\', edgecolor='black')

ax2.plot(x, dual_recon, 'g--o', label='Recon (Dual-Threshold 80/20)', linewidth=1.5, markersize=4)
ax2.plot(x, dual_dos, 'r--s', label='DoS (Dual-Threshold 80/20)', linewidth=1.5, markersize=4)

ax2.set_title('(b) Clustered Evasion vs. Dual-Threshold')
ax2.set_xlabel('Quorum Threshold ($q$)')
ax2.set_ylabel('Uncorrected Recall (%)')
ax2.set_xticks(x)
ax2.set_xticklabels([f'$q={q}$' for q in qs])
ax2.set_ylim(0, 118)
ax2.grid(True, linestyle=':', alpha=0.6, axis='y')
ax2.legend(loc='upper right', framealpha=0.9, fontsize=6.2, ncol=2)

plt.tight_layout()
plt.savefig('/root/workspace/workspace/03-Code/Projects/Legacy/SentinalMesh/paper/figures/fig6_adversarial_robustness.pdf', bbox_inches='tight')
plt.savefig('/root/workspace/workspace/03-Code/Projects/Legacy/SentinalMesh/paper/figures/fig6_adversarial_robustness.png', dpi=300, bbox_inches='tight')
print("Successfully regenerated fig6_adversarial_robustness.pdf and png with Dual-Threshold comparison.")
