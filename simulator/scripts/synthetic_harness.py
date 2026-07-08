import random
import statistics
import itertools

def sample_correlated(pool, active_pool, size, correlation):
    selected = set()
    while len(selected) < size:
        available = list(set(pool) - selected)
        active_avail = list(set(active_pool) - selected)
        if len(active_avail) > 0 and random.random() < correlation:
            selected.add(random.choice(active_avail))
        else:
            selected.add(random.choice(available))
    return selected

def run_synthetic_harness():
    N = 32
    k = 8
    num_flows = 1000
    seeds = [42, 43, 44]
    
    true_recalls = [0.3, 0.5, 0.7, 0.9]
    noise_levels = [0.0, 0.1, 0.25, 0.5, 0.75]
    q_values = [2, 4, 6, 8]
    correlations = [0.0, 0.3, 0.6]
    
    discard_rules = [
        "Broad-OR",
        "ExactMatches",
        "Subset",
        "Overlap-0.5",
        "Overlap-0.7",
        "Overlap-0.9",
        "Overlap-0.95",
        "Pop-Correction"
    ]
    
    # rule -> correlation -> max_error
    rule_max_error = {r: {c: 0.0 for c in correlations} for r in discard_rules}
    first_failure = {r: {c: None for c in correlations} for r in discard_rules}
    
    for c in correlations:
        for q in q_values:
            for recall in true_recalls:
                for p_noise in noise_levels:
                    cell_results = {r: [] for r in discard_rules}
                    
                    for seed in seeds:
                        random.seed(seed)
                        true_tp_count = int(num_flows * recall)
                        
                        rule_retained = {r: 0 for r in discard_rules}
                        t_detections = 0
                        c_detections = 0
                        
                        for i in range(num_flows):
                            is_strong_attack = (i < true_tp_count)
                            
                            # Background Corrobs (from N nodes)
                            if random.random() < p_noise:
                                m_b = random.randint(q, N) if q <= N else N
                                background_corrobs = set(random.sample(range(N), m_b))
                            else:
                                background_corrobs = set()
                                
                            # Replacement Corrobs (ONLY generated during attacks, used for per-flow comparison)
                            if random.random() < p_noise:
                                m_r = random.randint(q, k) if q <= k else k
                                replacement_corrobs = sample_correlated(range(N), background_corrobs, m_r, c)
                            else:
                                replacement_corrobs = set()
                                
                            # Attack Corrobs (from k targeted nodes)
                            if is_strong_attack:
                                m_a = random.randint(q, k) if q <= k else k
                            else:
                                m_a = random.randint(0, min(q - 1, k))
                            attack_corrobs = sample_correlated(range(k), set(range(k)) & background_corrobs, m_a, c)
                                
                            # Combine
                            tCorrobs = attack_corrobs | background_corrobs
                            cCorrobs = replacement_corrobs | background_corrobs
                            
                            is_t_detected = len(tCorrobs) >= q
                            c_detected = len(cCorrobs) >= q
                            
                            if is_t_detected:
                                t_detections += 1
                                if is_strong_attack:
                                    # Broad-OR
                                    if not c_detected: rule_retained["Broad-OR"] += 1
                                    # ExactMatches
                                    if not (c_detected and tCorrobs == cCorrobs): rule_retained["ExactMatches"] += 1
                                    # Subset
                                    if not (c_detected and tCorrobs.issubset(cCorrobs)): rule_retained["Subset"] += 1
                                    # Overlaps
                                    overlap = len(tCorrobs & cCorrobs) / float(len(tCorrobs)) if len(tCorrobs) > 0 else 0.0
                                    if not (c_detected and overlap >= 0.5): rule_retained["Overlap-0.5"] += 1
                                    if not (c_detected and overlap >= 0.7): rule_retained["Overlap-0.7"] += 1
                                    if not (c_detected and overlap >= 0.9): rule_retained["Overlap-0.9"] += 1
                                    if not (c_detected and overlap >= 0.95): rule_retained["Overlap-0.95"] += 1
                                    
                        # TN Simulation for p_hat (Control FPR)
                        # In isolated-category MCC, Control experiences NO replacement injection during True Negative windows.
                        # Thus, Control FPR (p_hat) is purely the ambient background noise rate crossing threshold q.
                        tn_windows = 1000
                        tn_c_detections = 0
                        for _ in range(tn_windows):
                            if random.random() < p_noise:
                                m_b = random.randint(q, N) if q <= N else N
                                background_corrobs = set(random.sample(range(N), m_b))
                            else:
                                background_corrobs = set()
                            if len(background_corrobs) >= q:
                                tn_c_detections += 1
                                
                        p_hat = tn_c_detections / float(tn_windows)
                        obs_rate = t_detections / float(num_flows)
                        
                        if p_hat < 1.0:
                            pop_recall = (obs_rate - p_hat) / (1.0 - p_hat)
                        else:
                            pop_recall = 0.0
                            
                        # We clip to [0,1] for error calculation
                        clipped_pop_recall = max(0.0, min(1.0, pop_recall))
                        
                        for r in discard_rules:
                            if r == "Pop-Correction":
                                computed_recall = clipped_pop_recall
                            else:
                                computed_recall = rule_retained[r] / float(num_flows)
                            error = abs(computed_recall - recall)
                            cell_results[r].append(error)
                            
                    for r in discard_rules:
                        mean_err = statistics.mean(cell_results[r])
                        if mean_err > rule_max_error[r][c]:
                            rule_max_error[r][c] = mean_err
                            if mean_err > 0.03 and first_failure[r][c] is None:
                                first_failure[r][c] = f"q={q}, R={recall}, N={p_noise}, Err={mean_err:.3f}"

    print("=== SYNTHETIC HARNESS RESULTS ===")
    print(f"Evaluated 80 cells across 3 seeds per correlation level.\n")
    print(f"{'Discard Rule':<15} | {'Corr':<4} | {'Max Error':<10} | {'Status':<6} | {'First Failure Context'}")
    print("-" * 75)
    for r in discard_rules:
        for c in correlations:
            status = "PASS" if rule_max_error[r][c] <= 0.03 else "FAIL"
            fail_ctx = first_failure[r][c] if first_failure[r][c] else "-"
            print(f"{r:<15} | {c:<4.1f} | {rule_max_error[r][c]:<10.4f} | {status:<6} | {fail_ctx}")

if __name__ == "__main__":
    run_synthetic_harness()
