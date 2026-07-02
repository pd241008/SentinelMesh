"use client"

import { useEffect, useState } from "react"

import BandwidthChart from "@/components/BandwidthChart"
import SweepChart from "@/components/SweepChart"
import { loadSweepResults } from "@/lib/loadSweepResults"
import type { SweepResult } from "@/lib/types"

export default function SweepPage() {
  const [data, setData] = useState<SweepResult[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadSweepResults().then((results) => {
      setData(results)
      setLoading(false)
    })
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center py-24 text-gray-500">
        Loading sweep results...
      </div>
    )
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold">Sweep Overview</h1>
        <p className="mt-1 text-sm text-gray-400">
          Detection recall and bandwidth across mesh size (N), fanout (f), quorum (q), and fragmentation (k)
        </p>
      </div>

      <section>
        <h2 className="mb-3 text-lg font-semibold">Detection Recall</h2>
        <SweepChart data={data} />
      </section>

      <section>
        <h2 className="mb-3 text-lg font-semibold">Bandwidth Overhead</h2>
        <BandwidthChart data={data} />
      </section>

      {data.length > 0 && (
        <section>
          <h2 className="mb-3 text-lg font-semibold">Raw Results</h2>
          <div className="overflow-x-auto rounded-lg border border-gray-800">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-gray-800 bg-gray-900">
                <tr>
                  {Object.keys(data[0]).map((key) => (
                    <th key={key} className="px-3 py-2 font-medium text-gray-400">{key}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {data.map((row, i) => (
                  <tr key={i} className="border-b border-gray-800/50 even:bg-gray-900/30">
                    {Object.values(row).map((val, j) => (
                      <td key={j} className="px-3 py-1.5 text-gray-300">{val}</td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </div>
  )
}
