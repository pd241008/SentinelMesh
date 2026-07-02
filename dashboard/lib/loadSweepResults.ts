import type { SweepResult } from "./types"

export async function loadSweepResults(): Promise<SweepResult[]> {
  const res = await fetch("/data/sweep_results.csv")
  if (!res.ok) return []
  const text = await res.text()
  return parseSweepCSV(text)
}

export function parseSweepCSV(text: string): SweepResult[] {
  const lines = text.trim().split("\n")
  if (lines.length < 2) return []

  const headers = lines[0].split(",").map((h) => h.trim())
  return lines.slice(1).map((line) => {
    const vals = line.split(",").map((v) => v.trim())
    const row: Record<string, number> = {}
    headers.forEach((h, i) => {
      row[h] = Number(vals[i])
    })
    return row as unknown as SweepResult
  })
}
