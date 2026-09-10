from contextlib import nullcontext
import json
from pathlib import Path
import re
import tempfile
import unittest
from unittest.mock import patch

import ab_verify


class ABVerificationTests(unittest.TestCase):
    def test_browser_script_keeps_visual_capture_separate_from_interactions(self):
        script = ab_verify.browser_script(
            {
                "url": "http://127.0.0.1:5191/upload-preview",
                "initial": "/tmp/initial.png",
                "final": "/tmp/final.png",
            }
        )
        self.assertIn("omitBackground: true", script)
        self.assertIn('animations: "disabled"', script)
        self.assertIn("await open();\n  await capture(options.final);", script)
        self.assertIn('await page.keyboard.press("Enter")', script)
        self.assertIn("await afterCancel.count() === 5", script)

    def test_source_files_ignores_install_and_build_products(self):
        with tempfile.TemporaryDirectory() as directory:
            app = Path(directory)
            (app / "src").mkdir()
            (app / "src/component.tsx").write_text("export {};\n")
            (app / "node_modules").mkdir()
            (app / "node_modules/ignored.js").write_text("ignored")
            (app / "dist").mkdir()
            (app / "dist/ignored.js").write_text("ignored")
            (app / "tsconfig.tsbuildinfo").write_text("ignored")
            self.assertEqual(set(ab_verify.source_files(app)), {"src/component.tsx"})

    def test_candidate_requires_direct_app_and_fails_before_install(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            run = root / "run"
            (run / "outputs/app/todoapp").mkdir(parents=True)
            evidence = root / "evidence"
            with patch.object(ab_verify, "command") as command:
                with self.assertRaisesRegex(RuntimeError, "directly at outputs/app"):
                    ab_verify.verify_candidate(run, evidence, 5191)
            command.assert_not_called()

    def test_candidate_result_binds_raw_screenshots_and_unchanged_source(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            run = root / "run"
            app = run / "outputs/app"
            app.mkdir(parents=True)
            (app / "package.json").write_text("{}\n")
            (app / "src").mkdir()
            (app / "src/component.tsx").write_text("export {};\n")
            evidence = root / "evidence"

            def command(args, cwd, log, timeout=60):
                if "run-code" in args:
                    source = Path(args[args.index("--filename") + 1]).read_text()
                    options = json.loads(
                        re.search(r"checkCandidate\(page, (.+)\);", source).group(1)
                    )
                    Path(options["initial"]).write_bytes(b"raw")
                    Path(options["final"]).write_bytes(b"raw")
                    return json.dumps(
                        {
                            "content": True,
                            "interactions": True,
                            "keyboard": True,
                            "initialRows": 7,
                            "finalRows": 7,
                            "capture": {"width": 436, "height": 406, "dpr": 1},
                        }
                    )
                return ""

            def compare(reference, actual, directory, name, threshold):
                (directory / f"{name}-metrics.json").write_text("{}\n")
                return {"changedPixels": 0, "changedRatio": 0.1, "perceptualRmse": 0.2}

            with (
                patch.object(ab_verify, "command", side_effect=command),
                patch.object(ab_verify, "server", return_value=nullcontext()),
                patch.object(ab_verify, "compare", side_effect=compare),
            ):
                result = ab_verify.verify_candidate(run, evidence, 5191)
            self.assertTrue(result["observed"]["content"])
            self.assertEqual(result["raw_source_reproduction_changed_pixels"], 0)
            self.assertEqual(
                set(result["evidence"]),
                {
                    "initial.png",
                    "final.png",
                    "reference-metrics.json",
                    "source-reproduction-metrics.json",
                },
            )


if __name__ == "__main__":
    unittest.main()
