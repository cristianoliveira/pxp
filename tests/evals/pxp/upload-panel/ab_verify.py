"""Coordinator-owned verification for one A/B upload-panel candidate.

The candidate only supplies source at ``outputs/app``. This program builds it,
starts it on loopback, captures fresh raw browser images, and checks behavior.
It deliberately does not trust an agent's screenshots, metrics, or report.
"""

import argparse
import hashlib
import json
from pathlib import Path
import tarfile

from controls import ROOT
from verify import command, compare, server

EXPECTED_SIZE = (436, 406, 1)


def source_files(app):
    """Hash source inputs so a capture can be tied to exact delivered code."""
    ignored = {"node_modules", "dist", "coverage", ".git"}
    return {
        str(path.relative_to(app)): hashlib.sha256(path.read_bytes()).hexdigest()
        for path in sorted(app.rglob("*"))
        if path.is_file()
        and path.suffix != ".tsbuildinfo"
        and not any(part in ignored for part in path.parts)
    }


def browser_script(options):
    return (
        "async page => {\n"
        + (ROOT / "ab_browser_check.js").read_text()
        + "\nreturn await checkCandidate(page, "
        + json.dumps(options)
        + ");\n}\n"
    )


def verify_candidate(run_directory, workspace, port):
    """Return independent checks for an agent run or fail closed on bad evidence."""
    run_directory = run_directory.resolve()
    workspace.mkdir(parents=True, exist_ok=False)
    app = run_directory / "outputs/app"
    if not app.is_dir() or not (app / "package.json").is_file():
        raise RuntimeError("Candidate must deliver its app directly at outputs/app")
    before = source_files(app)
    command(
        ["npm", "ci", "--ignore-scripts", "--no-audit", "--no-fund"],
        app,
        workspace / "install.log",
        timeout=120,
    )
    command(["npm", "test"], app, workspace / "tests.log", timeout=120)
    command(["npm", "run", "build"], app, workspace / "build.log", timeout=120)
    options = {
        "url": f"http://127.0.0.1:{port}/upload-preview",
        "initial": str(workspace / "initial.png"),
        "final": str(workspace / "final.png"),
    }
    script = workspace / "browser-check.js"
    script.write_text(browser_script(options))
    session = f"abv{workspace.name[-4:]}"
    with server(app, port, workspace / "server.log"):
        try:
            command(
                [
                    "playwright-cli",
                    f"-s={session}",
                    "open",
                    options["url"],
                    "--config",
                    str(ROOT / "accepted/capture.json"),
                ],
                workspace,
                workspace / "browser-open.log",
            )
            observed = json.loads(
                command(
                    [
                        "playwright-cli",
                        f"-s={session}",
                        "run-code",
                        "--filename",
                        str(script),
                        "--raw",
                    ],
                    workspace,
                    workspace / "browser-result.json",
                )
            )
        finally:
            command(
                ["playwright-cli", f"-s={session}", "close"],
                workspace,
                workspace / "browser-close.log",
            )
    if (
        tuple(observed["capture"][key] for key in ("width", "height", "dpr"))
        != EXPECTED_SIZE
    ):
        raise RuntimeError("Coordinator capture settings drifted")
    if source_files(app) != before:
        raise RuntimeError("Candidate source changed while coordinator verified it")
    identical = compare(
        workspace / "initial.png",
        workspace / "final.png",
        workspace,
        "source-reproduction",
        0,
    )
    if identical["changedPixels"]:
        raise RuntimeError(
            "Fresh initial and final source captures differ; evidence is unstable"
        )
    reference = compare(
        ROOT / "accepted/reference.png",
        workspace / "final.png",
        workspace,
        "reference",
        8,
    )
    result = {
        "source_sha256": before,
        "observed": observed,
        "raw_source_reproduction_changed_pixels": identical["changedPixels"],
        "reference_changed_ratio": reference["changedRatio"],
        "reference_perceptual_rmse": reference["perceptualRmse"],
        "evidence": {
            name: hashlib.sha256((workspace / name).read_bytes()).hexdigest()
            for name in (
                "initial.png",
                "final.png",
                "reference-metrics.json",
                "source-reproduction-metrics.json",
            )
        },
    }
    (workspace / "result.json").write_text(json.dumps(result, indent=2) + "\n")
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("run_directory", type=Path)
    parser.add_argument(
        "workspace", type=Path, help="New coordinator-owned evidence directory"
    )
    parser.add_argument("--port", type=int, default=5191)
    args = parser.parse_args()
    try:
        result = verify_candidate(
            args.run_directory, args.workspace.resolve(), args.port
        )
    except (OSError, RuntimeError, ValueError, KeyError, tarfile.TarError) as error:
        parser.exit(2, f"A/B verification infrastructure error: {error}\n")
    print(
        json.dumps(
            {
                "observed": result["observed"],
                "reference_changed_ratio": result["reference_changed_ratio"],
            }
        )
    )


if __name__ == "__main__":
    main()
