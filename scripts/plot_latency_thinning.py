import pandas as pd
import matplotlib.pyplot as plt

plt.rcParams.update({
    'font.size': 9,
    'font.family': 'serif',
    'axes.labelsize': 10,
    'axes.titlesize': 10,
    'xtick.labelsize': 8,
    'ytick.labelsize': 8,
    'legend.fontsize': 8,
    'lines.linewidth': 1.5,
    'lines.markersize': 6,
})

df = pd.read_csv('/root/workspace/workspace/03-Code/Projects/Legacy/SentinalMesh/results/latency_thinning.csv')

fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(7.2, 2.9), sharey=True)

# Colors for different lambdas
lambdas = [0.25, 0.5, 1.0, 5.0, 20.0]
colors = {0.25: '#d62728', 0.5: '#ff7f0e', 1.0: '#2ca02c', 5.0: '#1f77b4', 20.0: '#9467bd'}
markers = {0.25: 'o', 0.5: 's', 1.0: '^', 5.0: 'd', 20.0: 'v'}

# Panel (a): f = 2
df_f2 = df[df['f'] == 2]
for lam in lambdas:
    sub = df_f2[df_f2['lambda'] == lam].sort_values('N')
    lbl = f'$\\lambda = {lam}$ fl/rnd'
    ax1.plot(sub['N'], sub['latency'], marker=markers[lam], color=colors[lam], label=lbl)

ax1.set_title('(a) Binary Overlay ($f=2$)')
ax1.set_xlabel('Mesh Size ($N$)')
ax1.set_ylabel('Escalation Latency (Rounds)')
ax1.set_xscale('log', base=2)
ax1.set_xticks([8, 16, 32, 64])
ax1.get_xaxis().set_major_formatter(plt.ScalarFormatter())
ax1.grid(True, linestyle=':', alpha=0.6)
ax1.legend(loc='upper left', framealpha=0.9, fontsize=7.5)

# Panel (b): f = 3
df_f3 = df[df['f'] == 3]
for lam in lambdas:
    sub = df_f3[df_f3['lambda'] == lam].sort_values('N')
    lbl = f'$\\lambda = {lam}$ fl/rnd'
    ax2.plot(sub['N'], sub['latency'], marker=markers[lam], color=colors[lam], label=lbl)

ax2.set_title('(b) Mesh Overlay ($f=3$)')
ax2.set_xlabel('Mesh Size ($N$)')
ax2.set_xscale('log', base=2)
ax2.set_xticks([8, 16, 32, 64])
ax2.get_xaxis().set_major_formatter(plt.ScalarFormatter())
ax2.grid(True, linestyle=':', alpha=0.6)
ax2.legend(loc='upper left', framealpha=0.9, fontsize=7.5)

plt.tight_layout()
plt.savefig('/root/workspace/workspace/03-Code/Projects/Legacy/SentinalMesh/paper/figures/fig5_latency_thinning.pdf', bbox_inches='tight')
plt.savefig('/root/workspace/workspace/03-Code/Projects/Legacy/SentinalMesh/paper/figures/fig5_latency_thinning.png', dpi=300, bbox_inches='tight')
print("Successfully generated fig5_latency_thinning.pdf and fig5_latency_thinning.png")

