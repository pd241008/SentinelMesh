export interface SweepResult {
  N: number
  f: number
  q: number
  k: number
  W: number
  gossip_recall: number
  gossip_bandwidth: number
  gossip_latency: number
  indep_recall: number
  indep_bandwidth: number
  indep_latency: number
  cent_recall: number
  cent_bandwidth: number
  cent_latency: number
}

export interface CrosscheckCategory {
  category: string
  total: number
  detected: number
  precision: number
  recall: number
  f1: number
  avg_score: number
}

export interface CrosscheckResult {
  model: string
  overall: {
    accuracy: number
    precision: number
    recall: number
    f1: number
    avg_score_attack: number
    avg_score_normal: number
  }
  per_category: CrosscheckCategory[]
}
