"""Exercise controls and optional alternatives. No model calls or quality cutoff."""

import argparse
from contextlib import contextmanager
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import tarfile
import time
from urllib.error import URLError
from urllib.request import urlopen

from controls import ROOT, alternatives, prepare, variants


def command(args, cwd, log, timeout=60):
    with log.open("w") as output:
        process = subprocess.Popen(
            args,
            cwd=cwd,
            stdout=output,
            stderr=subprocess.STDOUT,
            start_new_session=True,
        )
        try:
            code = process.wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            os.killpg(process.pid, signal.SIGKILL)
            process.wait()
            raise RuntimeError(f"Command timed out; see {log}") from None
    if code:
        raise RuntimeError(f"Command failed ({code}); see {log}")
    return log.read_text()


@contextmanager
def server(app, port, log):
    # Refuse occupied ports before spawning, and never stop another process.
    with socket.socket() as probe:
        # A prior control can leave TIME_WAIT sockets, without a live listener.
        probe.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        probe.bind(("127.0.0.1", port))
    with log.open("w") as output:
        process = subprocess.Popen(
            ["npm", "run", "dev", "--", "--port", str(port), "--strictPort"],
            cwd=app,
            stdout=output,
            stderr=subprocess.STDOUT,
            start_new_session=True,
        )
        try:
            deadline = time.monotonic() + 20
            while True:
                if process.poll() is not None:
                    raise RuntimeError(f"Server exited; see {log}")
                try:
                    with urlopen(
                        f"http://127.0.0.1:{port}/upload-preview", timeout=1
                    ) as response:
                        if response.status == 200:
                            break
                except (URLError, TimeoutError):
                    pass
                if time.monotonic() >= deadline:
                    raise RuntimeError(f"Server did not become ready; see {log}")
                time.sleep(0.1)
            yield
        finally:
            if process.poll() is None:
                os.killpg(process.pid, signal.SIGTERM)
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    os.killpg(process.pid, signal.SIGKILL)
                    process.wait()


def compare(reference, actual, directory, name, threshold):
    text = command(
        [
            "pxp",
            str(reference),
            str(actual),
            "--threshold",
            str(threshold),
            "--json",
            "--output",
            str(directory / f"{name}-mask.png"),
            "--overlay",
            str(directory / f"{name}-overlay.png"),
            "--report",
            str(directory / f"{name}-report.html"),
        ],
        directory,
        directory / f"{name}-metrics.json",
    )
    return json.loads(text)


def matches_expected(observed, expected):
    if any(type(observed.get(key)) is not bool for key in expected):
        raise ValueError(
            "Missing or non-boolean check result; not a valid control result"
        )
    return all(observed[key] == value for key, value in expected.items())


def verify(workspace, port, *, include_alternatives=False):
    workspace.mkdir(parents=True, exist_ok=False)
    candidates = alternatives() if include_alternatives else {}
    definitions = {
        "accepted": {
            "expected": {"content": True, "retry": True, "visual_difference": False}
        },
        **variants(),
        **candidates,
    }
    session = f"ppc{os.getpid()}"
    results = {}
    for name, definition in definitions.items():
        directory = workspace / name
        app = directory / "app"
        prepare(name, app)
        if name == "accepted":
            command(
                ["npm", "ci", "--ignore-scripts", "--no-fund", "--no-audit"],
                app,
                directory / "install.log",
                timeout=120,
            )
        else:
            # Every source copy has the same lockfile; share one dependency install.
            (app / "node_modules").symlink_to(
                workspace / "accepted/app/node_modules", target_is_directory=True
            )
        if name == "accepted" or name in candidates:
            command(
                ["npm", "run", "test:coverage"],
                app,
                directory / "tests.log",
                timeout=120,
            )
        command(["npm", "run", "build"], app, directory / "build.log", timeout=120)
        options = {
            "url": f"http://127.0.0.1:{port}/upload-preview",
            "screenshot": str(directory / "actual.png"),
        }
        script = directory / "browser-check.js"
        script.write_text(
            "async page => {\n"
            + (ROOT / "browser_check.js").read_text()
            + "\nreturn await checkPanel(page, "
            + json.dumps(options)
            + ");\n}\n"
        )
        with server(app, port, directory / "server.log"):
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
                    directory,
                    directory / "browser-open.log",
                )
                text = command(
                    [
                        "playwright-cli",
                        f"-s={session}",
                        "run-code",
                        "--filename",
                        str(script),
                        "--raw",
                    ],
                    directory,
                    directory / "browser-result.json",
                )
                observed = json.loads(text)
            finally:
                command(
                    ["playwright-cli", f"-s={session}", "close"],
                    directory,
                    directory / "browser-close.log",
                )
        capture = observed["capture"]
        if (capture["width"], capture["height"], capture["dpr"]) != (436, 406, 1):
            raise RuntimeError("Capture settings drifted; not a valid control result")
        if name == "accepted":
            reproduction = compare(
                ROOT / "accepted/actual.png",
                directory / "actual.png",
                directory,
                "reproduction",
                0,
            )
            if reproduction["changedPixels"] != 0:
                raise RuntimeError(
                    "Accepted capture did not reproduce exactly. Review browser/OS/font drift; do not grade controls."
                )
        reference_metrics = compare(
            ROOT / "accepted/reference.png",
            directory / "actual.png",
            directory,
            "reference",
            8,
        )
        delta = compare(
            workspace / "accepted/actual.png",
            directory / "actual.png",
            directory,
            "accepted-delta",
            8,
        )
        observed["visual_difference"] = delta["changedPixels"] > 0
        results[name] = {
            "observed": observed,
            "expected": definition["expected"],
            "detected_as_expected": matches_expected(observed, definition["expected"]),
            "reference_changed_ratio": reference_metrics["changedRatio"],
            "reference_perceptual_rmse": reference_metrics["perceptualRmse"],
            "accepted_delta_changed_pixels": delta["changedPixels"],
        }
        if name in candidates:
            results[name]["review_status"] = definition["review_status"]
        (workspace / "results.json").write_text(json.dumps(results, indent=2) + "\n")
        print(
            name,
            json.dumps({key: observed[key] for key in definition["expected"]}),
            flush=True,
        )
    return all(result["detected_as_expected"] for result in results.values())


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "workspace", type=Path, help="New disposable directory outside skill inputs"
    )
    parser.add_argument("--port", type=int, default=5191)
    parser.add_argument(
        "--include-alternatives",
        action="store_true",
        help="Also test Grid rows and thinner Retry icon; human review remains separate",
    )
    args = parser.parse_args()
    try:
        passed = verify(
            args.workspace.resolve(),
            args.port,
            include_alternatives=args.include_alternatives,
        )
    except (OSError, ValueError, RuntimeError, KeyError, tarfile.TarError) as error:
        parser.exit(2, f"Calibration infrastructure error: {error}\n")
    if not passed:
        parser.exit(1, "A control did not behave as expected; inspect results.json\n")
    print(
        "Checks matched expectations. Alternatives still require human review; no universal visual quality threshold is inferred."
    )


if __name__ == "__main__":
    main()
