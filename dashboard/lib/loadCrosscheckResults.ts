import type { CrosscheckResult } from "./types"

export async function loadCrosscheckResults(): Promise<CrosscheckResult[]> {
  const res = await fetch("/data/report.json")
  if (!res.ok) return []
  return res.json()
}
