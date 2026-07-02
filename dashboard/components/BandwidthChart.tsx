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

export default function BandwidthChart({ data }: { data: SweepResult[] }) {
  if (data.length === 0) {
    return (
      <div className="flex items-center justify-center rounded-lg border border-dashed border-gray-700 p-12 text-gray-500">
        No sweep results loaded
      </div>
    )
  }

  const chartData = data.map((r) => ({
    label: `N=${r.N} f=${r.f}`,
    gossip: r.gossip_bandwidth,
    centralized: r.cent_bandwidth,
    independent: r.indep_bandwidth,
  }))

  return (
    <div className="overflow-x-auto">
      <BarChart width={chartData.length * 60 + 80} height={300} data={chartData}>
        <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
        <XAxis dataKey="label" tick={{ fontSize: 10, fill: "#9ca3af" }} interval={0} angle={-20} textAnchor="end" height={60} />
        <YAxis tick={{ fill: "#9ca3af" }} />
        <Tooltip
          contentStyle={{ background: "#1f2937", border: "1px solid #374151", borderRadius: 8 }}
          labelStyle={{ color: "#e5e7eb" }}
        />
        <Legend />
        <Bar dataKey="gossip" name="Gossip" fill="#3b82f6" />
        <Bar dataKey="centralized" name="Centralized" fill="#ef4444" />
        <Bar dataKey="independent" name="Independent" fill="#22c55e" />
      </BarChart>
    </div>
  )
}
