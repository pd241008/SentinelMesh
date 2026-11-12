/-
  Theorem 3: Minimax Evasion Threshold and No-Escape Bound under Dual-Threshold GQC
  Mechanized Formalization for SentinelMesh (IEEE EuroS&P 2027)

  Theorem Statement:
  In an N-node network operating under the Dual-Threshold escalation rule:
      Alert(c, t) = (∃ i, v_i ≥ τ_high) ∨ (|{i : v_i ≥ τ_low}| ≥ q)
  any adversary allocating total anomaly volume V = ∑_{i=1}^N v_i that successfully
  evades both unilateral detection (v_i < τ_high for all i) and quorum formation
  (|{i : v_i ≥ τ_low}| ≤ q - 1) is strictly bounded by:
      V < (q - 1) * τ_high + (N - q + 1) * τ_low
-/

import Mathlib.Data.Finset.Basic
import Mathlib.Data.Fin.Basic
import Mathlib.Algebra.BigOperators.Basic
import Mathlib.Data.Real.Basic

namespace SentinelMesh

/-- Parameters for the Dual-Threshold Minimax Evasion Analysis. -/
structure DualThresholdConfig where
  N : Nat        -- Network size (number of overlay nodes)
  q : Nat        -- Quorum threshold
  tau_low : Float  -- Low-confidence local threshold
  tau_high : Float -- High-confidence unilateral escalation threshold

/-- Validity conditions for dual-threshold parameters. -/
def DualThresholdConfig.isValid (c : DualThresholdConfig) : Prop :=
  c.N > 0 ∧
  c.q ≥ 1 ∧ c.q ≤ c.N ∧
  c.tau_low > 0.0 ∧
  c.tau_high > c.tau_low

/-- The maximum evasion budget V_max allowed without triggering detection. -/
def maxUndetectableVolume (c : DualThresholdConfig) : Float :=
  (c.q.toFloat - 1.0) * c.tau_high + (c.N.toFloat - c.q.toFloat + 1.0) * c.tau_low

/--
  Theorem 3 (Minimax No-Escape Bound):
  If the total attack volume V satisfies:
      V ≥ (q - 1) * τ_high + (N - q + 1) * τ_low
  then no spatial distribution (v_1, ..., v_N) can simultaneously evade
  unilateral high-confidence detection and quorum corroboration.
-/
theorem minimax_no_escape_bound
    (c : DualThresholdConfig)
    (h_val : c.isValid)
    (V : Float)
    (h_vol : V ≥ maxUndetectableVolume c) :
    ¬ (∃ (v : Fin c.N → Float),
        (∀ i, v i < c.tau_high) ∧
        (∃ (S : Finset (Fin c.N)), S.card ≤ c.q - 1 ∧ (∀ i ∉ S, v i < c.tau_low)) ∧
        (Finset.univ.sum (fun i => v i) = V)) := by
  intro h
  rcases h with ⟨v, hv_high, ⟨S, hS_card, hv_low⟩, hv_sum⟩
  have h₁ : c.N > 0 := h_val.1
  have h₂ : c.q ≥ 1 := h_val.2.1
  have h₃ : c.q ≤ c.N := h_val.2.2.1
  have h₄ : c.tau_low > 0.0 := h_val.2.2.2.1
  have h₅ : c.tau_high > c.tau_low := h_val.2.2.2.2
  
  have h₆ : S.card ≤ c.q - 1 := hS_card
  have h₇ : ∀ i ∉ S, v i < c.tau_low := hv_low
  have h₈ : ∀ i, v i < c.tau_high := hv_high
  
  -- Split the sum into two parts: elements in S and elements not in S
  have h₉ : Finset.univ.sum (fun i : Fin c.N => v i) = V := hv_sum
  
  have h₁₀ : Finset.univ.sum (fun i : Fin c.N => v i) = S.sum (fun i => v i) + (Finset.univ \ S).sum (fun i => v i) := by
    rw [← Finset.sum_union (Finset.disjoint_left (fun x hx => Finset.not_mem_sdiff x hx))]
    <;> simp [Finset.ext_iff]
    <;> aesop
  
  rw [h₁₀] at h₉
  
  -- Bound the sum over S
  have h₁₁ : S.sum (fun i => v i) < (S.card : Float) * c.tau_high := by
    have h₁₂ : S.sum (fun i => v i) < S.sum (fun i => c.tau_high) := by
      apply Finset.sum_lt_sum
      · intro i hi
        exact by
          have h₁₃ : v i < c.tau_high := h₈ i
          exact by
            norm_num at h₁₃ ⊢
            <;>
            (try linarith) <;>
            (try exact_mod_cast h₁₃)
      · -- Show there exists at least one element where the inequality is strict
        by_cases hS : S.Nonempty
        · obtain ⟨i, hi⟩ := hS
          have h₁₃ : v i < c.tau_high := h₈ i
          exact ⟨i, hi, by
            norm_num at h₁₃ ⊢
            <;>
            (try linarith) <;>
            (try exact_mod_cast h₁₃)⟩
        · -- If S is empty, the sum is 0 and we need to handle this case
          have h₁₃ : S = ∅ := by
            apply Finset.not_nonempty_iff_eq_empty.mp
            exact hS
          simp_all [Finset.sum_const, Finset.card_empty]
          <;> norm_num at h₂ ⊢ <;>
          (try linarith) <;>
          (try
            {
              have h₁₄ : (c.q : ℕ) ≥ 1 := by exact_mod_cast h₂
              have h₁₅ : (c.q : ℕ) - 1 ≥ 0 := by omega
              simp_all [Finset.sum_const, Finset.card_empty]
              <;> norm_num <;> linarith
            })
    have h₁₃ : S.sum (fun i => c.tau_high) = (S.card : Float) * c.tau_high := by
      simp [Finset.sum_const, Finset.card_univ]
      <;> ring_nf
      <;> field_simp
      <;> ring_nf
    rw [h₁₃] at h₁₂
    exact h₁₂
  
  -- Bound the sum over the complement of S
  have h₁₂ : (Finset.univ \ S).sum (fun i => v i) < ((Finset.univ \ S).card : Float) * c.tau_low := by
    have h₁₃ : (Finset.univ \ S).sum (fun i => v i) < (Finset.univ \ S).sum (fun i => c.tau_low) := by
      apply Finset.sum_lt_sum
      · intro i hi
        have h₁₄ : i ∉ S := by
          simp only [Finset.mem_sdiff, Finset.mem_univ, true_and_iff] at hi
          exact hi
        have h₁₅ : v i < c.tau_low := h₇ i h₁₄
        exact by
          norm_num at h₁₅ ⊢
          <;>
          (try linarith) <;>
          (try exact_mod_cast h₁₅)
      · -- Show there exists at least one element where the inequality is strict
        by_cases hT : (Finset.univ \ S).Nonempty
        · obtain ⟨i, hi⟩ := hT
          have h₁₄ : i ∉ S := by
            simp only [Finset.mem_sdiff, Finset.mem_univ, true_and_iff] at hi
            exact hi
          have h₁₅ : v i < c.tau_low := h₇ i h₁₄
          exact ⟨i, hi, by
            norm_num at h₁₅ ⊢
            <;>
            (try linarith) <;>
            (try exact_mod_cast h₁₅)⟩
        · -- If the complement is empty, then S = univ, which means S.card = N
          have h₁₄ : Finset.univ \ S = ∅ := by
            apply Finset.not_nonempty_iff_eq_empty.mp
            exact hT
          have h₁₅ : S = Finset.univ := by
            apply Finset.ext_iff.mpr
            intro x
            simp only [h₁₄, Finset.mem_empty_iff_false, Finset.mem_sdiff, Finset.mem_univ, true_and_iff]
            <;> tauto
          have h₁₆ : S.card = c.N := by
            rw [h₁₅]
            simp [Finset.card_univ]
            <;> norm_cast
          have h₁₇ : S.card ≤ c.q - 1 := hS_card
          have h₁₈ : (c.q : ℕ) ≤ c.N := h₃
          have h₁₉ : (c.N : ℕ) ≤ c.q - 1 := by
            omega
          have h₂₀ : (c.q : ℕ) - 1 < c.q := by
            have h₂₁ : (c.q : ℕ) ≥ 1 := by exact_mod_cast h₂
            omega
          have h₂₁ : (c.N : ℕ) < c.q := by
            omega
          have h₂₂ : (c.q : ℕ) ≤ c.N := h₃
          omega
    have h₁₄ : (Finset.univ \ S).sum (fun i => c.tau_low) = ((Finset.univ \ S).card : Float) * c.tau_low := by
      simp [Finset.sum_const, Finset.card_univ]
      <;> ring_nf
      <;> field_simp
      <;> ring_nf
    rw [h₁₄] at h₁₃
    exact h₁₃
  
  -- Combine the bounds
  have h₁₃ : (S.card : Float) * c.tau_high + ((Finset.univ \ S).card : Float) * c.tau_low ≤ maxUndetectableVolume c := by
    have h₁₄ : (S.card : ℕ) ≤ c.q - 1 := by exact_mod_cast hS_card
    have h₁₅ : (Finset.univ \ S).card = c.N - S.card := by
      rw [Finset.card_sdiff (Finset.subset_univ S)]
      <;> simp [Finset.card_univ]
      <;> norm_cast
    have h₁₆ : (S.card : Float) * c.tau_high + ((Finset.univ \ S).card : Float) * c.tau_low = (S.card : Float) * c.tau_high + (c.N - S.card : Float) * c.tau_low := by
      rw [h₁₅]
      <;> norm_cast
      <;> field_simp
      <;> ring_nf
    rw [h₁₆]
    have h₁₇ : (S.card : Float) * c.tau_high + (c.N - S.card : Float) * c.tau_low ≤ (c.q - 1 : Float) * c.tau_high + (c.N - (c.q - 1) : Float) * c.tau_low := by
      have h₁₈ : (S.card : Float) ≤ (c.q - 1 : Float) := by
        have h₁₉ : (S.card : ℕ) ≤ c.q - 1 := by exact_mod_cast hS_card
        exact_mod_cast h₁₉
      have h₁₉ : c.tau_high > c.tau_low := h₅
      have h₂₀ : (c.N : Float) ≥ (S.card : Float) := by
        have h₂₁ : S.card ≤ c.N := by
          calc
            S.card ≤ Finset.card (Finset.univ : Finset (Fin c.N)) := Finset.card_le_card (Finset.subset_univ S)
            _ = c.N := by simp [Finset.card_univ]
        exact_mod_cast h₂₁
      have h₂₁ : (S.card : Float) * c.tau_high + (c.N - S.card : Float) * c.tau_low ≤ (c.q - 1 : Float) * c.tau_high + (c.N - (c.q - 1) : Float) * c.tau_low := by
        -- Since τ_high > τ_low, increasing the coefficient of τ_high (and decreasing the coefficient of τ_low) increases the sum
        have h₂₂ : (c.q - 1 : Float) ≥ (S.card : Float) := by
          have h₂₃ : (S.card : ℕ) ≤ c.q - 1 := by exact_mod_cast hS_card
          exact_mod_cast h₂₃
        have h₂₃ : (c.N : Float) - (c.q - 1 : Float) ≤ (c.N : Float) - (S.card : Float) := by
          linarith
        have h₂₄ : c.tau_high > c.tau_low := h₅
        nlinarith [h₂₄]
      exact h₂₁
    have h₁₈ : (c.q - 1 : Float) * c.tau_high + (c.N - (c.q - 1) : Float) * c.tau_low = maxUndetectableVolume c := by
      dsimp [maxUndetectableVolume]
      <;> ring_nf
      <;> field_simp
      <;> ring_nf
      <;> norm_cast
      <;> simp_all [Nat.cast_sub, Nat.cast_add, Nat.cast_one]
      <;> field_simp at *
      <;> ring_nf at *
      <;> norm_num at *
      <;> linarith
    linarith
  
  -- Contradiction: V ≥ maxUndetectableVolume but sum < maxUndetectableVolume
  have h₁₄ : (S.card : Float) * c.tau_high + ((Finset.univ \ S).card : Float) * c.tau_low < maxUndetectableVolume c := by
    linarith
  
  have h₁₅ : S.sum (fun i => v i) + (Finset.univ \ S).sum (fun i => v i) < maxUndetectableVolume c := by
    linarith
  
  have h₁₆ : V < maxUndetectableVolume c := by
    linarith
  
  linarith

end SentinelMesh