from dataclasses import dataclass, field
from pathlib import Path
from typing import Sequence

import numpy as np
import pandas as pd

from dataset import CATEGORY_COLUMN, FEATURE_COLUMNS, FlowDataset
from models import AutoencoderScorer, BaseScorer, GoEWMAScorer, IsolationForestScorer

RESULTS_DIR = Path(__file__).resolve().parent.parent.parent / "results" / "crosscheck"


@dataclass
class PerCategoryMetrics:
    category: str
    total: int
    detected: int
    precision: float
    recall: float
    f1_score: float
    avg_score: float


@dataclass
class EvaluationResult:
    model_name: str
    accuracy: float
    precision: float
    recall: float
    f1_score: float
    avg_score_attack: float
    avg_score_normal: float
    scores: np.ndarray = field(repr=False)
    labels: np.ndarray = field(repr=False)
    categories: np.ndarray = field(repr=False)
    per_category: list[PerCategoryMetrics] = field(default_factory=list)


def _compute_binary_metrics(
    preds: np.ndarray, labels: np.ndarray,
) -> tuple[float, float, float, float]:
    tp = (preds & labels).sum()
    fp = (preds & ~labels).sum()
    fn = (~preds & labels).sum()
    tn = (~preds & ~labels).sum()

    total = tp + tn + fp + fn
    accuracy = float(tp / total) if total > 0 else 0.0
    precision = float(tp / (tp + fp)) if (tp + fp) > 0 else 0.0
    recall = float(tp / (tp + fn)) if (tp + fn) > 0 else 0.0
    f1 = (
        2 * precision * recall / (precision + recall)
        if (precision + recall) > 0
        else 0.0
    )
    return accuracy, precision, recall, f1


def evaluate_scorer(
    scorer: BaseScorer,
    dataset: FlowDataset,
    model_name: str,
    threshold: float = 0.5,
) -> EvaluationResult:
    scorer.train(dataset)
    scores = scorer.score(dataset)
    labels = dataset.get_labels()
    categories = dataset.get_categories()
    preds = scores >= threshold

    accuracy, precision, recall, f1 = _compute_binary_metrics(preds, labels)

    attack_mask = labels
    normal_mask = ~labels
    avg_score_attack = float(scores[attack_mask].mean()) if attack_mask.any() else 0.0
    avg_score_normal = float(scores[normal_mask].mean()) if normal_mask.any() else 0.0

    per_category: list[PerCategoryMetrics] = []
    for cat in sorted(set(categories)):
        mask = categories == cat
        cat_total = mask.sum()
        if cat_total == 0:
            continue
        cat_labels = labels[mask]
        cat_preds = preds[mask]
        cat_scores = scores[mask]
        detected = cat_preds.sum()
        _, cat_prec, cat_rec, cat_f1 = _compute_binary_metrics(cat_preds, cat_labels)
        per_category.append(PerCategoryMetrics(
            category=str(cat),
            total=int(cat_total),
            detected=int(detected),
            precision=cat_prec,
            recall=cat_rec,
            f1_score=cat_f1,
            avg_score=float(cat_scores.mean()),
        ))

    return EvaluationResult(
        model_name=model_name,
        accuracy=accuracy,
        precision=precision,
        recall=recall,
        f1_score=f1,
        avg_score_attack=avg_score_attack,
        avg_score_normal=avg_score_normal,
        per_category=per_category,
        scores=scores,
        labels=labels,
        categories=categories,
    )


def run_validation(
    dataset_path: str | Path,
    num_nodes: int = 1,
    threshold: float = 0.5,
    include_go_ewma: bool = True,
) -> list[EvaluationResult]:
    dataset = FlowDataset.load_csv(dataset_path)
    results: list[EvaluationResult] = []

    if num_nodes > 1 and "id" not in dataset.flows.columns:
        dataset.flows["id"] = range(len(dataset))

    partitions = dataset.partition_by_node(num_nodes)

    for node_id, partition in enumerate(partitions):
        scorers: list[tuple[BaseScorer, str]] = [
            (IsolationForestScorer(random_state=42), f"if_node{node_id}"),
            (AutoencoderScorer(epochs=50), f"ae_node{node_id}"),
        ]
        if include_go_ewma:
            scorers.insert(0, (GoEWMAScorer(), f"go_ewma_node{node_id}"))

        for scorer, name in scorers:
            result = evaluate_scorer(scorer, partition, name, threshold)
            results.append(result)

    return results


def generate_crosscheck_report(
    dataset_path: str | Path,
    num_nodes: int = 1,
    threshold: float = 0.5,
    output_dir: str | Path | None = None,
) -> list[EvaluationResult]:
    results = run_validation(dataset_path, num_nodes, threshold)

    if output_dir is None:
        output_dir = RESULTS_DIR
    output_dir = Path(output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    _write_summary(results, output_dir)
    _write_per_category(results, output_dir)
    _write_per_flow_scores(results, output_dir, dataset_path)
    _write_json_report(results, output_dir)

    return results


def _write_summary(results: Sequence[EvaluationResult], output_dir: Path) -> None:
    rows = []
    for r in results:
        rows.append({
            "model": r.model_name,
            "accuracy": round(r.accuracy, 4),
            "precision": round(r.precision, 4),
            "recall": round(r.recall, 4),
            "f1": round(r.f1_score, 4),
            "avg_score_attack": round(r.avg_score_attack, 4),
            "avg_score_normal": round(r.avg_score_normal, 4),
        })
    df = pd.DataFrame(rows)
    df.to_csv(output_dir / "summary.csv", index=False)


def _write_per_category(results: Sequence[EvaluationResult], output_dir: Path) -> None:
    rows = []
    for r in results:
        for pc in r.per_category:
            rows.append({
                "model": r.model_name,
                "category": pc.category,
                "total": pc.total,
                "detected": pc.detected,
                "precision": round(pc.precision, 4),
                "recall": round(pc.recall, 4),
                "f1": round(pc.f1_score, 4),
                "avg_score": round(pc.avg_score, 4),
            })
    df = pd.DataFrame(rows)
    df.to_csv(output_dir / "per_category.csv", index=False)


def _write_per_flow_scores(
    results: Sequence[EvaluationResult],
    output_dir: Path,
    dataset_path: str | Path,
) -> None:
    dataset = FlowDataset.load_csv(dataset_path)
    if "id" in dataset.flows.columns:
        flow_ids = dataset.flows["id"].values
    else:
        flow_ids = np.arange(len(dataset))
    categories = dataset.get_categories()
    labels = dataset.get_labels()

    data = {
        "flow_id": flow_ids,
        "category": categories,
        "label": labels,
    }
    for r in results:
        data[r.model_name] = r.scores

    df = pd.DataFrame(data)
    df.to_csv(output_dir / "per_flow_scores.csv", index=False)


def _write_json_report(results: Sequence[EvaluationResult], output_dir: Path) -> None:
    import json

    report: list[dict] = []
    for r in results:
        report.append({
            "model": r.model_name,
            "overall": {
                "accuracy": round(r.accuracy, 4),
                "precision": round(r.precision, 4),
                "recall": round(r.recall, 4),
                "f1": round(r.f1_score, 4),
                "avg_score_attack": round(r.avg_score_attack, 4),
                "avg_score_normal": round(r.avg_score_normal, 4),
            },
            "per_category": [
                {
                    "category": pc.category,
                    "total": pc.total,
                    "detected": pc.detected,
                    "precision": round(pc.precision, 4),
                    "recall": round(pc.recall, 4),
                    "f1": round(pc.f1_score, 4),
                    "avg_score": round(pc.avg_score, 4),
                }
                for pc in r.per_category
            ],
        })

    with open(output_dir / "report.json", "w") as f:
        json.dump(report, f, indent=2)


def print_results(results: Sequence[EvaluationResult]) -> None:
    rows = []
    for r in results:
        rows.append({
            "model": r.model_name,
            "accuracy": f"{r.accuracy:.3f}",
            "precision": f"{r.precision:.3f}",
            "recall": f"{r.recall:.3f}",
            "f1": f"{r.f1_score:.3f}",
            "avg(attack)": f"{r.avg_score_attack:.3f}",
            "avg(normal)": f"{r.avg_score_normal:.3f}",
        })
    print(pd.DataFrame(rows).to_string(index=False))
