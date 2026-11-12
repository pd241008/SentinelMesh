import pandas as pd
import matplotlib.pyplot as plt
import numpy as np

# Set publication style
plt.rcParams.update({
    'font.size': 9,
    'font.family': 'serif',
    'axes.labelsize': 10,
    'axes.titlesize': 10,
    'xtick.labelsize': 8,
    'ytick.labelsize': 8,
    'legend.fontsize': 8,
    'figure.titlesize': 11,
    'lines.linewidth': 1.5,
    'lines.markersize': 6,
})

df = pd.read_csv('/root/workspace/workspace/03-Code/Projects/Legacy/SentinalMesh/results/full_grid/master_grid.csv')
sub = df[(df['N'] == 32) & (df['k'] == 8)]

fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(7.0, 2.8), sharey=True)

# Colors and markers
colors = {'f=2': '#1f77b4', 'f=3': '#2ca02c', 'f=4': '#ff7f0e'}
markers = {'f=2': 'o', 'f=3': 's', 'f=4': '^'}

# 1. Independent Point
indep_row = sub[sub['f'] == 3].sort_values('q').iloc[0]
r_indep_fpr = indep_row['indep_recon_fpr_mean'] * 100
r_indep_rec = indep_row['indep_flow_recon_recall_mean'] * 100
d_indep_fpr = indep_row['indep_dos_fpr_mean'] * 100
d_indep_rec = indep_row['indep_flow_dos_recall_mean'] * 100

ax1.scatter(r_indep_fpr, r_indep_rec, color='red', marker='X', s=80, label='Independent ($q=1$)', zorder=5)
ax2.scatter(d_indep_fpr, d_indep_rec, color='red', marker='X', s=80, label='Independent ($q=1$)', zorder=5)

# 2. Centralized Curve
cent_df = sub[sub['f'] == 3].sort_values('q')
ax1.plot(cent_df['cent_recon_fpr_mean'] * 100, cent_df['cent_flow_recon_recall_mean'] * 100,
         'k--', marker='d', label='Centralized ($q \\in [2,8]$)', zorder=4)
ax2.plot(cent_df['cent_dos_fpr_mean'] * 100, cent_df['cent_flow_dos_recall_mean'] * 100,
         'k--', marker='d', label='Centralized ($q \\in [2,8]$)', zorder=4)

for _, row in cent_df.iterrows():
    q = int(row['q'])
    ax1.annotate(f'$q{q}$', (row['cent_recon_fpr_mean']*100, row['cent_flow_recon_recall_mean']*100),
                 textcoords="offset points", xytext=(-10, 4), fontsize=7, color='black')
    ax2.annotate(f'$q{q}$', (row['cent_dos_fpr_mean']*100, row['cent_flow_dos_recall_mean']*100),
                 textcoords="offset points", xytext=(-10, 4), fontsize=7, color='black')

# 3. Gossip Curves (f=2, 3, 4)
for f in [2, 3, 4]:
    fdf = sub[sub['f'] == f].sort_values('q')
    label = f'Gossip ($f={f}$)'
    ax1.plot(fdf['gossip_recon_fpr_mean'] * 100, fdf['gossip_flow_recon_recall_mean'] * 100,
             color=colors[f'f={f}'], marker=markers[f'f={f}'], label=label, zorder=3)
    ax2.plot(fdf['gossip_dos_fpr_mean'] * 100, fdf['gossip_flow_dos_recall_mean'] * 100,
             color=colors[f'f={f}'], marker=markers[f'f={f}'], label=label, zorder=3)

    for _, row in fdf.iterrows():
        q = int(row['q'])
        if f == 3: # annotate f=3
            ax1.annotate(f'$q{q}$', (row['gossip_recon_fpr_mean']*100, row['gossip_flow_recon_recall_mean']*100),
                         textcoords="offset points", xytext=(4, -8), fontsize=7, color=colors['f=3'])
            ax2.annotate(f'$q{q}$', (row['gossip_dos_fpr_mean']*100, row['gossip_flow_dos_recall_mean']*100),
                         textcoords="offset points", xytext=(4, -8), fontsize=7, color=colors['f=3'])

ax1.set_title('(a) Reconnaissance Fragmentation ($k=8$)')
ax1.set_xlabel('False-Positive Rate (%)')
ax1.set_ylabel('Uncorrected Recall (%)')
ax1.grid(True, linestyle=':', alpha=0.6)
ax1.set_xlim(-2, 85)
ax1.set_ylim(15, 105)
ax1.legend(loc='lower right', framealpha=0.9)

ax2.set_title('(b) Low-Rate DoS Fragmentation ($k=8$)')
ax2.set_xlabel('False-Positive Rate (%)')
ax2.grid(True, linestyle=':', alpha=0.6)
ax2.set_xlim(-2, 95)
ax2.legend(loc='lower right', framealpha=0.9)

plt.tight_layout()
plt.savefig('/root/workspace/workspace/03-Code/Projects/Legacy/SentinalMesh/paper/figures/fig2_roc.pdf', bbox_inches='tight')
plt.savefig('/root/workspace/workspace/03-Code/Projects/Legacy/SentinalMesh/paper/figures/fig2_roc.png', dpi=300, bbox_inches='tight')
print("Successfully generated fig2_roc.pdf and fig2_roc.png")

