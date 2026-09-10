"""Validate committed eval inputs without browser, network, or model calls."""

from collections import Counter
import hashlib
import json
from pathlib import Path
import struct
import tarfile
import unittest

from pack_fixture import ROOT_FILES

SKILL = Path(__file__).resolve().parents[1]


class EvalContractTests(unittest.TestCase):
    def test_cases_have_unique_ids_complete_expectations_and_existing_inputs(self):
        data = json.loads((SKILL / "evals/evals.json").read_text())
        self.assertEqual(data["skill_name"], "pxp")
        cases = data["evals"]
        self.assertEqual(len(cases), 3)
        self.assertEqual(len({case["id"] for case in cases}), len(cases))
        for case in cases:
            with self.subTest(case=case["id"]):
                self.assertIsInstance(case["id"], int)
                self.assertTrue(case["prompt"].strip())
                self.assertTrue(case["expected_output"].strip())
                self.assertGreaterEqual(len(case["expectations"]), 5)
                self.assertEqual(len(case["expectations"]), len(set(case["expectations"])))
                self.assertEqual(len(case["files"]), len({Path(p).name for p in case["files"]}))
                for name in case["files"]:
                    self.assertTrue((SKILL / name).is_file(), name)

    def test_reference_is_unchanged_and_matches_crop_case_dimensions(self):
        image = (SKILL / "fixtures/upload-modal-multifiles.png").read_bytes()
        self.assertEqual(image[:8], b"\x89PNG\r\n\x1a\n")
        self.assertEqual(struct.unpack(">II", image[16:24]), (436, 406))
        checksum = (SKILL / "fixtures/reference.sha256").read_text().split()[0]
        self.assertEqual(hashlib.sha256(image).hexdigest(), checksum)
        cases = json.loads((SKILL / "evals/evals.json").read_text())["evals"]
        self.assertIn("32,24,436,406", " ".join(cases[2]["expectations"]))

    def test_state_has_fixed_distinct_ids_and_reference_status_counts(self):
        state = json.loads((SKILL / "fixtures/upload-state.json").read_text())
        self.assertEqual(state["title"], "Uploading 7 items")
        items = state["items"]
        self.assertEqual(len(items), 7)
        self.assertEqual(len({item["id"] for item in items}), 7)
        self.assertEqual(Counter(item["status"] for item in items), {
            "uploading": 2, "failed": 1, "completed": 4,
        })

    def test_archive_is_small_safe_source_with_original_tests_and_lockfile(self):
        archive_path = SKILL / "fixtures/todoapp.tar.gz"
        self.assertLess(archive_path.stat().st_size, 100_000)
        with tarfile.open(archive_path) as archive:
            names = set(archive.getnames())
            for member in archive.getmembers():
                parts = Path(member.name).parts
                self.assertEqual(parts[0], "todoapp")
                self.assertNotIn("..", parts)
                self.assertTrue(member.isfile())
                self.assertFalse(set(parts) & {
                    "node_modules", "dist", "coverage", ".pi", ".tmp", ".playwright-cli",
                })
            for name in (*ROOT_FILES, "src/main.tsx", "src/App.test.tsx", "src/model.test.ts", "src/storage.test.ts"):
                self.assertIn(f"todoapp/{name}", names)
            package = json.load(archive.extractfile("todoapp/package.json"))
            lock = json.load(archive.extractfile("todoapp/package-lock.json"))
            self.assertEqual(package["dependencies"], lock["packages"][""]["dependencies"])
            self.assertEqual(package["devDependencies"], lock["packages"][""]["devDependencies"])


if __name__ == "__main__":
    unittest.main()
