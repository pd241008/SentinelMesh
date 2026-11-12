import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import numpy as np

plt.rcParams.update({
    'font.size': 8.5,
    'font.family': 'serif',
    'axes.labelsize': 9.5,
    'axes.titlesize': 9.5,
    'xtick.labelsize': 8,
    'ytick.labelsize': 8,
})

df = pd.read_csv('/root/workspace/workspace/03-Code/Projects/Legacy/SentinalMesh/results/full_grid/master_grid.csv')

# Focus on canonical fanout f=3
sub = df[df['f'] == 3].copy()
sub['q_k'] = sub['q'] / sub['k']
# Format q/k as percentage string for clean display
sub['Quorum Ratio'] = sub['q_k'].apply(lambda x: f"{int(x*100)}%")
sub['Config (N, k)'] = sub.apply(lambda r: f"N={int(r['N'])}, k={int(r['k'])}", axis=1)

# Sort configs by N then k
configs_sorted = []
for n in [8, 16, 32, 64]:
    for k in [4, 8, 16]:
        if n >= k: # physically meaningful distributed scenarios
            configs_sorted.append(f"N={n}, k={k}")

sub = sub[sub['Config (N, k)'].isin(configs_sorted)]

pivot_recon = sub.pivot_table(index='Config (N, k)', columns='Quorum Ratio', values='gossip_flow_recon_recall_mean') * 100
pivot_dos = sub.pivot_table(index='Config (N, k)', columns='Quorum Ratio', values='gossip_flow_dos_recall_mean') * 100

# Reorder indices and columns
col_order = ['25%', '50%', '75%', '100%']
# Add cols if present
col_order = [c for c in col_order if c in pivot_recon.columns]

pivot_recon = pivot_recon.reindex(index=configs_sorted, columns=col_order)
pivot_dos = pivot_dos.reindex(index=configs_sorted, columns=col_order)

fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(7.2, 3.8), sharey=True)

sns.heatmap(pivot_recon, annot=True, fmt=".1f", cmap="Blues", cbar=True, ax=ax1,
            vmin=10, vmax=100, cbar_kws={'label': 'Uncorrected Recall (%)', 'shrink': 0.8})
ax1.set_title('(a) Reconnaissance Uncorrected Recall ($f=3$)')
ax1.set_xlabel('Quorum Corroboration Ratio ($q/k$)')
ax1.set_ylabel('Network Topology ($N, k$)')

sns.heatmap(pivot_dos, annot=True, fmt=".1f", cmap="Oranges", cbar=True, ax=ax2,
            vmin=10, vmax=100, cbar_kws={'label': 'Uncorrected Recall (%)', 'shrink': 0.8})
ax2.set_title('(b) DoS Uncorrected Recall ($f=3$)')
ax2.set_xlabel('Quorum Corroboration Ratio ($q/k$)')
ax2.set_ylabel('')

plt.tight_layout()
plt.savefig('/root/workspace/workspace/03-Code/Projects/Legacy/SentinalMesh/paper/figures/fig3_grid_heatmap.pdf', bbox_inches='tight')
plt.savefig('/root/workspace/workspace/03-Code/Projects/Legacy/SentinalMesh/paper/figures/fig3_grid_heatmap.png', dpi=300, bbox_inches='tight')
print("Successfully generated fig3_grid_heatmap.pdf and fig3_grid_heatmap.png")

