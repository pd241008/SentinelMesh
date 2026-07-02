"use client"

import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts"

import type { SweepResult } from "@/lib/types"

const COLORS = ["#3b82f6", "#22c55e", "#ef4444"]
const MODES = [
  { key: "gossip_recall", label: "Gossip" },
  { key: "indep_recall", label: "Independent" },
  { key: "cent_recall", label: "Centralized" },
] as const

export default function SweepChart({ data }: { data: SweepResult[] }) {
  if (data.length === 0) {
    return (
      <div className="flex items-center justify-center rounded-lg border border-dashed border-gray-700 p-12 text-gray-500">
        No sweep results loaded
      </div>
    )
  }

  const chartData = data.map((r) => ({
    label: `N=${r.N} f=${r.f} q=${r.q} k=${r.k}`,
    gossip_recall: r.gossip_recall,
    indep_recall: r.indep_recall,
    cent_recall: r.cent_recall,
  }))

  return (
    <div className="overflow-x-auto">
      <BarChart width={chartData.length * 60 + 80} height={350} data={chartData}>
        <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
        <XAxis dataKey="label" tick={{ fontSize: 10, fill: "#9ca3af" }} interval={0} angle={-20} textAnchor="end" height={60} />
        <YAxis domain={[0, 1]} tick={{ fill: "#9ca3af" }} />
        <Tooltip
          contentStyle={{ background: "#1f2937", border: "1px solid #374151", borderRadius: 8 }}
          labelStyle={{ color: "#e5e7eb" }}
        />
        <Legend />
        {MODES.map((m, i) => (
          <Bar key={m.key} dataKey={m.key} name={m.label} fill={COLORS[i]} />
        ))}
      </BarChart>
    </div>
  )
}
