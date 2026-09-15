#!/usr/bin/env python3
"""Run bounded, offline anatomy workflow controls and write a JSON report.

Expected geometry is loaded from tests/evals, never from prompts. The report keeps
alternative workflows comparable on command count and bytes while marking their
geometry as not comparable when they do not emit element bounds.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import subprocess
import sys
import tempfile
import time
from pathlib import Path
from typing import Any


DEFAULT_ROOT = Path(__file__).resolve().parents[3]
DEFAULT_MANIFEST = DEFAULT_ROOT / "tests/evals/pxp/anatomy-workflow/manifest.json"


def sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def run_command(command: list[str], cwd: Path) -> dict[str, Any]:
    started = time.monotonic()
    process = subprocess.run(command, cwd=cwd, capture_output=True, text=True, check=False)
    duration_ms = round((time.monotonic() - started) * 1000, 3)
    parsed = None
    try:
        parsed = json.loads(process.stdout)
    except json.JSONDecodeError:
        pass
    return {
        "command": command,
        "exit_code": process.returncode,
        "stdout_bytes": len(process.stdout.encode()),
        "stderr_bytes": len(process.stderr.encode()),
        "duration_ms": duration_ms,
        "truncated": bool(parsed and parsed.get("truncated")),
        "json": parsed,
        "stdout": process.stdout,
        "stderr": process.stderr,
    }


def command_for(
    workflow: str, pxp: Path, image: Path, args: list[str], artifact_dir: Path,
) -> list[str]:
    if workflow == "anatomy":
        return [str(pxp), "anatomy", str(image), *args]
    if workflow == "compare":
        return [
            str(pxp), str(image), str(image), "--json", "--full",
            "--output", str(artifact_dir / "comparison-mask.png"),
        ]
    if workflow == "probe":
        return [str(pxp), "probe", str(image), str(image), "--at", "0,0", "--format", "json"]
    if workflow == "scan":
        return [str(pxp), "scan", str(image), str(image), "--row", "0", "--format", "json"]
    if workflow == "review":
        return [str(pxp), "review", "--help"]
    raise ValueError(f"unknown workflow: {workflow}")


def check_anatomy(result: dict[str, Any], expected: dict[str, Any]) -> dict[str, Any]:
    output = result["json"] or {}
    checks: dict[str, Any] = {}
    if "exit_code" in expected:
        checks["exit_code"] = result["exit_code"] == expected["exit_code"]
    if "stdout_contains" in expected:
        checks["stdout_contains"] = all(
            text in result.get("stdout", "") for text in expected["stdout_contains"]
        )
    for key in ("background", "width", "height", "group", "total", "returned", "truncated", "query"):
        if key in expected:
            checks[key] = output.get(key) == expected[key]
    if "elements" in expected:
        actual = output.get("elements", [])
        checks["elements"] = [
            {
                "bounds": item.get("bounds"),
                "pixels": item.get("pixels"),
                "dominantColor": item.get("dominantColor"),
            }
            for item in actual
        ] == expected["elements"]
    if "message_contains" in expected:
        checks["message_contains"] = expected["message_contains"] in output.get("message", "")
    if "hint_contains" in expected:
        checks["hint_contains"] = expected["hint_contains"] in output.get("hint", "")
    if "min_elements_with_group_8_min_pixels_8" in expected:
        checks["minimum_elements"] = len(output.get("elements", [])) >= expected["min_elements_with_group_8_min_pixels_8"]
    if "anchor_bounds" in expected:
        actual_bounds = {tuple(item.get("bounds", {}).values()) for item in output.get("elements", [])}
        checks["anchor_bounds"] = all(
            tuple(anchor.values()) in actual_bounds for anchor in expected["anchor_bounds"]
        )
    return {"checks": checks, "passed": all(checks.values()) if checks else None}


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--pxp", type=Path, required=True, help="pxp executable")
    parser.add_argument("--manifest", type=Path, default=DEFAULT_MANIFEST)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    manifest = json.loads(args.manifest.read_text())
    root = args.manifest.parent
    report: dict[str, Any] = {
        "schema": 1,
        "manifest": str(args.manifest),
        "pxp": str(args.pxp),
        "cases": [],
        "limitations": [
            "Alternative workflows are setup-cost controls; compare/probe/scan do not emit anatomy bounds.",
            "Review is checked for command availability only because it requires an interactive browser decision.",
            "This harness measures deterministic CLI evidence, not model quality or visual semantic accuracy.",
        ],
    }
    with tempfile.TemporaryDirectory(prefix="pxp-anatomy-eval-") as temporary:
        artifact_dir = Path(temporary)
        for case in manifest["cases"]:
            image = root / case["image"]
            expected = json.loads((root / manifest["expected_root"] / case["expected"]).read_text())
            prompt = (root / manifest["prompt_root"] / case["prompt"]).read_text()
            case_report: dict[str, Any] = {
                "id": case["id"],
                "kind": case["kind"],
                "image": case["image"],
                "sha256": sha256(image),
                "prompt_sha256": hashlib.sha256(prompt.encode()).hexdigest(),
                "budget": case["budget"],
                "workflows": {},
            }
            for workflow in manifest["workflows"]:
                command = command_for(
                    workflow, args.pxp, image,
                    case["anatomy_args"] if workflow == "anatomy" else [],
                    artifact_dir,
                )
                result = run_command(command, DEFAULT_ROOT)
                if workflow == "anatomy":
                    result["geometry"] = check_anatomy(result, expected)
                else:
                    result["geometry"] = {"comparable": False}
                result.pop("stdout", None)
                result.pop("stderr", None)
                case_report["workflows"][workflow] = result
            report["cases"].append(case_report)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n")
    print(args.output)
    return 0


if __name__ == "__main__":
    sys.exit(main())
