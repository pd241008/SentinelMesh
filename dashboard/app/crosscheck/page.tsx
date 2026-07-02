"use client"

import { useEffect, useState } from "react"

import { loadCrosscheckResults } from "@/lib/loadCrosscheckResults"
import type { CrosscheckResult } from "@/lib/types"

export default function CrosscheckPage() {
  const [data, setData] = useState<CrosscheckResult[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadCrosscheckResults().then((results) => {
      setData(results)
      setLoading(false)
    })
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center py-24 text-gray-500">
        Loading crosscheck results...
      </div>
    )
  }

  if (data.length === 0) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">ML Crosscheck</h1>
        <p className="text-sm text-gray-400">Comparison of Go EWMA scorer vs ML models</p>
        <div className="flex items-center justify-center rounded-lg border border-dashed border-gray-700 p-12 text-gray-500">
          No crosscheck results loaded. Run the validation pipeline first.
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold">ML Crosscheck</h1>
        <p className="mt-1 text-sm text-gray-400">Comparison of Go EWMA scorer vs Isolation Forest and Autoencoder</p>
      </div>

      <section>
        <h2 className="mb-3 text-lg font-semibold">Overall Metrics</h2>
        <div className="overflow-x-auto rounded-lg border border-gray-800">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-gray-800 bg-gray-900">
              <tr>
                <th className="px-3 py-2 font-medium text-gray-400">Model</th>
                <th className="px-3 py-2 font-medium text-gray-400">Accuracy</th>
                <th className="px-3 py-2 font-medium text-gray-400">Precision</th>
                <th className="px-3 py-2 font-medium text-gray-400">Recall</th>
                <th className="px-3 py-2 font-medium text-gray-400">F1</th>
                <th className="px-3 py-2 font-medium text-gray-400">Avg Attack Score</th>
                <th className="px-3 py-2 font-medium text-gray-400">Avg Normal Score</th>
              </tr>
            </thead>
            <tbody>
              {data.map((r, i) => (
                <tr key={i} className="border-b border-gray-800/50 even:bg-gray-900/30">
                  <td className="px-3 py-1.5 font-medium text-white">{r.model}</td>
                  <td className="px-3 py-1.5 text-gray-300">{r.overall.accuracy.toFixed(4)}</td>
                  <td className="px-3 py-1.5 text-gray-300">{r.overall.precision.toFixed(4)}</td>
                  <td className="px-3 py-1.5 text-gray-300">{r.overall.recall.toFixed(4)}</td>
                  <td className="px-3 py-1.5 text-gray-300">{r.overall.f1.toFixed(4)}</td>
                  <td className="px-3 py-1.5 text-gray-300">{r.overall.avg_score_attack.toFixed(4)}</td>
                  <td className="px-3 py-1.5 text-gray-300">{r.overall.avg_score_normal.toFixed(4)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section>
        <h2 className="mb-3 text-lg font-semibold">Per-Category Metrics</h2>
        <div className="overflow-x-auto rounded-lg border border-gray-800">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-gray-800 bg-gray-900">
              <tr>
                <th className="px-3 py-2 font-medium text-gray-400">Model</th>
                <th className="px-3 py-2 font-medium text-gray-400">Category</th>
                <th className="px-3 py-2 font-medium text-gray-400">Total</th>
                <th className="px-3 py-2 font-medium text-gray-400">Detected</th>
                <th className="px-3 py-2 font-medium text-gray-400">Precision</th>
                <th className="px-3 py-2 font-medium text-gray-400">Recall</th>
                <th className="px-3 py-2 font-medium text-gray-400">F1</th>
                <th className="px-3 py-2 font-medium text-gray-400">Avg Score</th>
              </tr>
            </thead>
            <tbody>
              {data.map((r, i) =>
                r.per_category.map((pc, j) => (
                  <tr key={`${i}-${j}`} className="border-b border-gray-800/50 even:bg-gray-900/30">
                    <td className="px-3 py-1.5 font-medium text-white">{i === 0 && j === 0 ? r.model : ""}</td>
                    <td className="px-3 py-1.5 text-gray-300">{pc.category}</td>
                    <td className="px-3 py-1.5 text-gray-300">{pc.total}</td>
                    <td className="px-3 py-1.5 text-gray-300">{pc.detected}</td>
                    <td className="px-3 py-1.5 text-gray-300">{pc.precision.toFixed(4)}</td>
                    <td className="px-3 py-1.5 text-gray-300">{pc.recall.toFixed(4)}</td>
                    <td className="px-3 py-1.5 text-gray-300">{pc.f1.toFixed(4)}</td>
                    <td className="px-3 py-1.5 text-gray-300">{pc.avg_score.toFixed(4)}</td>
                  </tr>
                )),
              )}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  )
}
