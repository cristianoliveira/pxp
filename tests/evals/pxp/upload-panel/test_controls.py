"""Offline tests for golden provenance and one-defect source controls."""

import hashlib
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest

import controls

ROOT = Path(__file__).resolve().parent


class ControlTests(unittest.TestCase):
    def test_accepted_artifacts_have_verified_hashes_and_human_approval(self):
        provenance = controls.verify_accepted()
        self.assertEqual(provenance["decision"], "accept")
        self.assertEqual(provenance["feedback"], "Looks good")
        self.assertEqual(provenance["gates"]["visual_quality"], "pass")
        self.assertGreaterEqual(len(provenance["artifact_sha256"]), 5)

    def test_reference_is_the_original_task_png(self):
        original = (
            ROOT.parents[3]
            / "skills/pxp/fixtures/upload-modal-multifiles.png"
        )
        self.assertEqual(
            (ROOT / "accepted/reference.png").read_bytes(), original.read_bytes()
        )

    def test_solution_is_not_inside_the_agent_skill_or_starter(self):
        skill = ROOT.parents[3] / "skills/pxp"
        self.assertFalse(ROOT.is_relative_to(skill))
        with tarfile.open(skill / "fixtures/todoapp.tar.gz") as archive:
            self.assertFalse(any("UploadStatus" in name for name in archive.getnames()))
        self.assertFalse(any("accepted" in p.parts for p in skill.rglob("*")))

    def test_mutations_change_exactly_one_source_file_each(self):
        original = controls.load_source(ROOT / "accepted/app.tar.gz")
        for name, specification in controls.variants().items():
            with self.subTest(variant=name):
                modified = controls.mutate(original, specification)
                self.assertEqual(set(modified), set(original))
                changed = [p for p in original if original[p] != modified[p]]
                self.assertEqual(changed, [specification["file"]])
                self.assertEqual(
                    modified[changed[0]].decode(),
                    original[changed[0]]
                    .decode()
                    .replace(specification["old"], specification["new"]),
                )
        self.assertEqual(original, controls.load_source(ROOT / "accepted/app.tar.gz"))

    def test_alternatives_keep_source_controls_separate_and_require_human_review(self):
        original = controls.load_source(ROOT / "accepted/app.tar.gz")
        alternatives = controls.alternatives()
        self.assertEqual(set(alternatives), {"grid-rows", "retry-icon"})
        self.assertFalse(set(alternatives) & set(controls.variants()))
        for name, specification in alternatives.items():
            with self.subTest(alternative=name):
                modified = controls.mutate(original, specification)
                self.assertEqual(
                    [path for path in original if original[path] != modified[path]],
                    [specification["file"]],
                )
                self.assertEqual(
                    specification["expected"], {"content": True, "retry": True}
                )
                self.assertEqual(
                    specification["review_status"],
                    "accepted" if name == "retry-icon" else "pending_human_review",
                )
                with tempfile.TemporaryDirectory() as temporary:
                    destination = Path(temporary) / name
                    controls.prepare(name, destination)
                    self.assertEqual(
                        (destination / specification["file"]).read_bytes(),
                        modified[specification["file"]],
                    )
        self.assertEqual(original, controls.load_source(ROOT / "accepted/app.tar.gz"))

    def test_icon_acceptance_is_bound_to_reviewed_image_and_source(self):
        alternative = controls.alternatives()["retry-icon"]
        review = json.loads((ROOT / alternative["review"]).read_text())
        self.assertEqual(review["decision"], "accept")
        self.assertEqual(review["feedback"], "Yes")
        self.assertEqual(review["question"], "Would you accept this thinner icon too?")
        image = (ROOT / review["screenshot"]).read_bytes()
        self.assertEqual(hashlib.sha256(image).hexdigest(), review["screenshot_sha256"])
        measured = json.loads((ROOT / "alternative-observations.json").read_text())
        self.assertEqual(
            review["screenshot_sha256"], measured["screenshot_sha256"]["retry-icon"]
        )
        source = controls.load_source(ROOT / "accepted/app.tar.gz")
        changed = controls.mutate(source, alternative)[alternative["file"]]
        self.assertEqual(
            hashlib.sha256(changed).hexdigest(), review["modified_source_sha256"]
        )
        self.assertEqual(
            hashlib.sha256((ROOT / "accepted/app.tar.gz").read_bytes()).hexdigest(),
            review["base_archive_sha256"],
        )

    def test_ambiguous_or_missing_mutation_fails(self):
        for source in (b"before before", b"no match"):
            with self.subTest(source=source):
                with self.assertRaisesRegex(ValueError, "exactly once"):
                    controls.mutate(
                        {"src/a.ts": source},
                        {"file": "src/a.ts", "old": "before", "new": "after"},
                    )

    def test_mutation_cannot_create_an_unexpected_file(self):
        with self.assertRaisesRegex(ValueError, "source file"):
            controls.mutate({}, {"file": "../outside", "old": "a", "new": "b"})

    def test_prepare_writes_real_source_and_refuses_existing_destination(self):
        with tempfile.TemporaryDirectory() as temporary:
            destination = Path(temporary) / "accepted"
            controls.prepare("accepted", destination)
            self.assertTrue((destination / "src/UploadStatus.tsx").is_file())
            self.assertTrue((destination / "package-lock.json").is_file())
            self.assertFalse((destination / "node_modules").exists())
            with self.assertRaises(FileExistsError):
                controls.prepare("accepted", destination)

    def test_unknown_variant_fails_before_creating_destination(self):
        with tempfile.TemporaryDirectory() as temporary:
            destination = Path(temporary) / "not-created"
            with self.assertRaisesRegex(ValueError, "Unknown variant"):
                controls.prepare("not-a-variant", destination)
            self.assertFalse(destination.exists())

    def test_corrupted_archive_fails_hash_validation(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "app.tar.gz").write_bytes(b"tampered")
            (root / "provenance.json").write_text(
                json.dumps(
                    {
                        "decision": "accept",
                        "artifact_sha256": {
                            "app.tar.gz": hashlib.sha256(b"original").hexdigest()
                        },
                    }
                )
            )
            with self.assertRaisesRegex(ValueError, "Checksum mismatch"):
                controls.verify_accepted(root)

    def test_unsafe_archive_members_fail_without_extraction(self):
        for name, kind in (
            ("app/../../outside", tarfile.REGTYPE),
            ("/absolute", tarfile.REGTYPE),
            ("app/link", tarfile.SYMTYPE),
            ("app/link", tarfile.LNKTYPE),
            ("other/a", tarfile.REGTYPE),
        ):
            with (
                self.subTest(name=name, kind=kind),
                tempfile.TemporaryDirectory() as temporary,
            ):
                archive = Path(temporary) / "source.tar.gz"
                with tarfile.open(archive, "w:gz") as tar:
                    member = tarfile.TarInfo(name)
                    member.type = kind
                    member.linkname = "/outside" if kind != tarfile.REGTYPE else ""
                    tar.addfile(member, io.BytesIO(b""))
                with self.assertRaises(ValueError):
                    controls.load_source(archive)

    def test_archive_size_count_duplicate_and_required_file_limits(self):
        cases = (
            ([("app/large", 1_000_001)], "Unsupported"),
            ([(f"app/{i}", 900_000) for i in range(5)], "Oversized"),
            ([(f"app/{i}", 0) for i in range(129)], "Unsafe"),
            ([("app/duplicate", 0), ("app/duplicate", 0)], "duplicate"),
            ([("app/incomplete", 0)], "missing"),
        )
        for entries, error in cases:
            with self.subTest(error=error), tempfile.TemporaryDirectory() as temporary:
                archive = Path(temporary) / "source.tar.gz"
                with tarfile.open(archive, "w:gz") as tar:
                    for name, size in entries:
                        member = tarfile.TarInfo(name)
                        member.size = size
                        tar.addfile(member, io.BytesIO(bytes(size)))
                with self.assertRaisesRegex(ValueError, error):
                    controls.load_source(archive)

    def test_unapproved_or_unsafe_provenance_is_rejected(self):
        cases = (
            ({"decision": "reject"}, "not been accepted"),
            (
                {"decision": "accept", "artifact_sha256": {"../outside": "invalid"}},
                "Unsafe artifact",
            ),
        )
        for provenance, message in cases:
            with (
                self.subTest(message=message),
                tempfile.TemporaryDirectory() as temporary,
            ):
                root = Path(temporary)
                (root / "provenance.json").write_text(json.dumps(provenance))
                with self.assertRaisesRegex(ValueError, message):
                    controls.verify_accepted(root)

    def test_expectations_require_content_behavior_and_visual_evidence(self):
        variants = controls.variants()
        self.assertEqual(
            set(variants),
            {"row-spacing", "missing-row", "status-colors", "broken-retry"},
        )
        self.assertEqual(
            variants["broken-retry"]["expected"],
            {"content": True, "retry": False, "visual_difference": False},
        )
        self.assertFalse(variants["missing-row"]["expected"]["content"])
        for name in ("row-spacing", "status-colors"):
            self.assertEqual(
                variants[name]["expected"],
                {"content": True, "retry": True, "visual_difference": True},
            )


if __name__ == "__main__":
    unittest.main()
