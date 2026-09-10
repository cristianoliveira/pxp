"""Offline checks for source-only, reproducible eval fixtures."""

import importlib.util
import io
import os
from pathlib import Path
import tarfile
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location(
    "pack_fixture", Path(__file__).with_name("pack_fixture.py")
)
pack_fixture = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(pack_fixture)


class PackFixtureTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.app = self.root / "app"
        self.app.mkdir()
        for name in pack_fixture.ROOT_FILES:
            (self.app / name).write_text("fixture\n")
        for name in ("src/main.tsx", "public/favicon.svg"):
            path = self.app / name
            path.parent.mkdir(exist_ok=True)
            path.write_text("fixture\n")

    def pack(self):
        return pack_fixture.pack(self.app)

    def test_archive_contains_source_lockfile_and_dotfile_config_not_generated_data(self):
        for name in (
            "node_modules/react/index.js", "dist/index.html", "coverage/index.html",
            ".pi/session.json", ".tmp/output.png", ".playwright-cli/page.yml",
            "src/.cache/old.ts", "src/main.tsbuildinfo", "src/.DS_Store",
        ):
            path = self.app / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text("do not ship")

        with tarfile.open(fileobj=io.BytesIO(self.pack()), mode="r:gz") as archive:
            self.assertEqual(
                sorted(archive.getnames()),
                sorted(f"todoapp/{name}" for name in (
                    *pack_fixture.ROOT_FILES, "src/main.tsx", "public/favicon.svg"
                )),
            )
            self.assertEqual(archive.extractfile("todoapp/src/main.tsx").read(), b"fixture\n")

    def test_archive_bytes_do_not_depend_on_timestamps_or_permissions(self):
        first = self.pack()
        for path in self.app.rglob("*"):
            os.utime(path, (123456789, 123456789))
            path.chmod(0o755)

        self.assertEqual(first, self.pack())

    def test_missing_required_file_fails(self):
        (self.app / "package-lock.json").unlink()

        with self.assertRaisesRegex(ValueError, "package-lock.json"):
            self.pack()

    def test_source_file_symlink_is_rejected(self):
        (self.app / "src/main.tsx").unlink()
        (self.app / "src/main.tsx").symlink_to(self.app / "README.md")

        with self.assertRaisesRegex(ValueError, "symlink"):
            self.pack()

    def test_source_directory_symlink_is_rejected(self):
        (self.app / "src/main.tsx").unlink()
        (self.app / "src").rmdir()
        (self.app / "src").symlink_to(self.app / "public", target_is_directory=True)

        with self.assertRaisesRegex(ValueError, "symlink"):
            self.pack()

    def test_app_root_symlink_is_rejected(self):
        link = self.root / "linked-app"
        link.symlink_to(self.app, target_is_directory=True)

        with self.assertRaisesRegex(ValueError, "symlink"):
            pack_fixture.pack(link)

    def test_required_config_symlink_is_rejected(self):
        (self.app / "package.json").unlink()
        (self.app / "package.json").symlink_to(self.app / "README.md")

        with self.assertRaisesRegex(ValueError, "symlink"):
            self.pack()

    def test_missing_entrypoint_fails(self):
        (self.app / "src/main.tsx").unlink()

        with self.assertRaisesRegex(ValueError, "src/main.tsx"):
            self.pack()

    def test_cli_writes_reproducible_archive(self):
        output = self.root / "app.tar.gz"

        with patch("sys.argv", ["pack_fixture.py", str(self.app), str(output)]), patch("sys.stdout", new_callable=io.StringIO) as stdout:
            pack_fixture.main()

        self.assertEqual(output.read_bytes(), self.pack())
        self.assertIn(f"{output}: {output.stat().st_size} bytes", stdout.getvalue())

    def test_cli_reports_missing_input_without_writing_output(self):
        output = self.root / "app.tar.gz"
        missing = self.root / "missing"

        with patch("sys.argv", ["pack_fixture.py", str(missing), str(output)]), patch("sys.stderr", new_callable=io.StringIO) as stderr:
            with self.assertRaises(SystemExit) as raised:
                pack_fixture.main()

        self.assertEqual(raised.exception.code, 1)
        self.assertIn("Cannot pack fixture: Missing required file", stderr.getvalue())
        self.assertFalse(output.exists())

    def test_missing_source_tree_fails(self):
        (self.app / "src/main.tsx").unlink()
        (self.app / "src").rmdir()

        with self.assertRaisesRegex(ValueError, "src"):
            self.pack()


if __name__ == "__main__":
    unittest.main()
