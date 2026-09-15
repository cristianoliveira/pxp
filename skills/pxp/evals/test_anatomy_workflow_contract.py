"""Validate the anatomy workflow eval package without running a model."""

import hashlib
import json
from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parents[3]
EVAL_ROOT = ROOT / "tests/evals/pxp/anatomy-workflow"
MANIFEST = EVAL_ROOT / "manifest.json"


class AnatomyWorkflowContractTests(unittest.TestCase):
    def setUp(self):
        self.manifest = json.loads(MANIFEST.read_text())

    def test_cases_are_bounded_and_prompts_do_not_include_expected_geometry(self):
        cases = self.manifest["cases"]
        self.assertEqual(len(cases), 6)
        self.assertEqual(len({case["id"] for case in cases}), len(cases))
        for case in cases:
            with self.subTest(case=case["id"]):
                self.assertGreater(case["budget"], 0)
                prompt = (EVAL_ROOT / self.manifest["prompt_root"] / case["prompt"]).read_text().lower()
                self.assertNotIn("expected", prompt)
                self.assertNotIn("bounds:", prompt)
                image = EVAL_ROOT / case["image"]
                self.assertTrue(image.is_file())
                self.assertEqual(case["sha256"], hashlib.sha256(image.read_bytes()).hexdigest())
                self.assertTrue((EVAL_ROOT / self.manifest["expected_root"] / case["expected"]).is_file())

    def test_controls_cover_required_failure_and_layout_cases(self):
        ids = {case["id"] for case in self.manifest["cases"]}
        self.assertTrue({
            "simple-control", "grouped-control", "empty-control",
            "truncated-control", "malformed-control", "ambiguous-layout-holdout",
        } <= ids)

    def test_workflow_controls_are_declared_and_review_is_noninteractive(self):
        self.assertEqual({"anatomy", "compare", "probe", "scan", "review"}, set(self.manifest["workflows"]))
        self.assertIn("interactive", self.manifest["workflows"]["review"]["note"])
        self.assertIn("same image twice", self.manifest["workflows"]["compare"]["note"])


if __name__ == "__main__":
    unittest.main()
