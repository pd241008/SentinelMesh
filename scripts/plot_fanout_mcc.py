import matplotlib.pyplot as plt
import numpy as np

plt.rcParams.update({
    'font.size': 8.5,
    'font.family': 'serif',
    'axes.labelsize': 9.5,
    'axes.titlesize': 9.5,
    'xtick.labelsize': 8.5,
    'ytick.labelsize': 8.5,
    'legend.fontsize': 7.5,
    'figure.titlesize': 10.5,
})

# Ground-truth data from Table 4 (N=64, k=8, q=4)
fanouts = ['f = 2', 'f = 3', 'f = 4']
x = np.arange(len(fanouts))
width = 0.19

# Data: DoS Corr is strictly higher than Recon Corr across all fanouts!
recon_unc = [61.96, 78.76, 85.53]
recon_corr = [39.03, 47.58, 46.62]
dos_unc = [61.88, 73.11, 78.45]
dos_corr = [45.46, 53.37, 56.87]

fig, ax = plt.subplots(figsize=(5.4, 3.0))

rects1 = ax.bar(x - 1.5*width, recon_unc, width, label='Recon (Uncorrected)', color='#1f77b4', alpha=0.5, edgecolor='black', hatch='//')
rects2 = ax.bar(x - 0.5*width, recon_corr, width, label='Recon (MCC-Corrected)', color='#1f77b4', edgecolor='black')
rects3 = ax.bar(x + 0.5*width, dos_unc, width, label='DoS (Uncorrected)', color='#ff7f0e', alpha=0.5, edgecolor='black', hatch='\\\\')
rects4 = ax.bar(x + 1.5*width, dos_corr, width, label='DoS (MCC-Corrected)', color='#ff7f0e', edgecolor='black')

# Add text labels on top of bars
def autolabel(rects, is_bold=False):
    for rect in rects:
        h = rect.get_height()
        weight = 'bold' if is_bold else 'normal'
        ax.annotate(f'{h:.1f}%',
                    xy=(rect.get_x() + rect.get_width() / 2, h),
                    xytext=(0, 2),  # 2 points vertical offset
                    textcoords="offset points",
                    ha='center', va='bottom', fontsize=6.5, weight=weight)

autolabel(rects1)
autolabel(rects2, is_bold=True)
autolabel(rects3)
autolabel(rects4, is_bold=True)

ax.set_ylabel('Flow-Level Recall (%)')
ax.set_title('Fanout Sensitivity & MCC Impact ($N=64, k=8, q=4$)')
ax.set_xticks(x)
ax.set_xticklabels(fanouts)
ax.set_ylim(25, 98)
ax.grid(True, linestyle=':', alpha=0.5, axis='y')
ax.legend(loc='upper left', ncol=2, framealpha=0.95, fontsize=7.2)

plt.tight_layout()
plt.savefig('/root/workspace/workspace/03-Code/Projects/Legacy/SentinalMesh/paper/figures/fig4_fanout_mcc.pdf', bbox_inches='tight')
plt.savefig('/root/workspace/workspace/03-Code/Projects/Legacy/SentinalMesh/paper/figures/fig4_fanout_mcc.png', dpi=300, bbox_inches='tight')
print("Successfully regenerated fig4_fanout_mcc.pdf and png with exact data labels matching Table 4.")
