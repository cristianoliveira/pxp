"""Runner policy checks; browser execution is covered by verify.py's real smoke."""

from contextlib import nullcontext
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import verify


class VerificationPolicyTests(unittest.TestCase):
    def test_expected_behavior_failure_counts_as_detected_not_as_working_ui(self):
        self.assertTrue(
            verify.matches_expected(
                {"content": True, "retry": False, "visual_difference": False},
                {"content": True, "retry": False, "visual_difference": False},
            )
        )
        self.assertFalse(
            verify.matches_expected(
                {"content": True, "retry": True, "visual_difference": False},
                {"content": True, "retry": False, "visual_difference": False},
            )
        )

    def test_missing_or_nonboolean_results_are_infrastructure_errors(self):
        for result in ({}, {"retry": None}, {"retry": 0}, {"retry": "false"}):
            with self.subTest(result=result), self.assertRaises(ValueError):
                verify.matches_expected(result, {"retry": False})

    def run_mocked(
        self,
        workspace,
        *,
        capture_width=436,
        reproduction=0,
        hide_color_difference=False,
        browser_error=False,
        include_alternatives=False,
    ):
        def prepare(name, app):
            app.mkdir(parents=True)

        def command(args, cwd, log, timeout=60):
            if "run-code" not in args:
                return ""
            if browser_error:
                raise RuntimeError("browser crashed")
            name = cwd.name
            return json.dumps(
                {
                    "content": name != "missing-row",
                    "retry": name != "broken-retry",
                    "capture": {"width": capture_width, "height": 406, "dpr": 1},
                }
            )

        def compare(reference, actual, directory, name, threshold):
            delta = directory.name in (
                "row-spacing",
                "missing-row",
                "status-colors",
                "retry-icon",
            )
            if hide_color_difference and directory.name == "status-colors":
                delta = False
            return {
                "changedPixels": reproduction if name == "reproduction" else int(delta),
                "changedRatio": 0.1,
                "perceptualRmse": 0.1,
            }

        with (
            patch.object(verify, "prepare", side_effect=prepare),
            patch.object(verify, "command", side_effect=command) as commands,
            patch.object(verify, "server", return_value=nullcontext()),
            patch.object(verify, "compare", side_effect=compare),
            patch("builtins.print"),
        ):
            outcome = verify.verify(
                workspace, 5191, include_alternatives=include_alternatives
            )
        return outcome, commands

    def test_live_listener_is_not_reused_or_stopped(self):
        with (
            verify.socket.socket() as listener,
            tempfile.TemporaryDirectory() as directory,
        ):
            listener.bind(("127.0.0.1", 0))
            listener.listen()
            port = listener.getsockname()[1]
            with patch.object(verify.subprocess, "Popen") as popen:
                with self.assertRaises(OSError):
                    with verify.server(
                        Path(directory), port, Path(directory) / "server.log"
                    ):
                        self.fail("Must not enter a server using somebody else's port")
                popen.assert_not_called()

    def test_server_preflight_reuses_timewait_addresses_and_stops_owned_process(self):
        with (
            tempfile.TemporaryDirectory() as directory,
            patch.object(verify.socket, "socket") as socket_factory,
            patch.object(verify.subprocess, "Popen") as popen,
            patch.object(verify, "urlopen") as request,
            patch.object(verify.os, "killpg") as killpg,
        ):
            process = popen.return_value
            process.pid = 12345
            process.poll.return_value = None
            request.return_value.__enter__.return_value.status = 200
            with verify.server(Path(directory), 5191, Path(directory) / "server.log"):
                pass

            probe = socket_factory.return_value.__enter__.return_value
            probe.setsockopt.assert_called_once_with(
                verify.socket.SOL_SOCKET, verify.socket.SO_REUSEADDR, 1
            )
            probe.bind.assert_called_once_with(("127.0.0.1", 5191))
            killpg.assert_called_once_with(12345, verify.signal.SIGTERM)
            process.wait.assert_called_once_with(timeout=5)

    def test_runner_saves_all_control_results_with_mocked_browser(self):
        with tempfile.TemporaryDirectory() as directory:
            workspace = Path(directory) / "run"
            passed, commands = self.run_mocked(workspace)
            results = json.loads((workspace / "results.json").read_text())
        self.assertTrue(passed)
        self.assertEqual(len(results), 5)
        self.assertFalse(results["broken-retry"]["observed"]["retry"])
        self.assertTrue(results["broken-retry"]["detected_as_expected"])
        npm_installs = [
            c for c in commands.call_args_list if c.args[0][:2] == ["npm", "ci"]
        ]
        self.assertEqual(len(npm_installs), 1)
        self.assertIn("--ignore-scripts", npm_installs[0].args[0])

    def test_alternatives_allow_visual_difference_and_preserve_declared_review(self):
        with tempfile.TemporaryDirectory() as directory:
            workspace = Path(directory) / "run"
            passed, commands = self.run_mocked(workspace, include_alternatives=True)
            results = json.loads((workspace / "results.json").read_text())
        self.assertTrue(passed)
        self.assertEqual(len(results), 7)
        self.assertTrue(results["retry-icon"]["observed"]["visual_difference"])
        for name in ("grid-rows", "retry-icon"):
            self.assertTrue(results[name]["detected_as_expected"])
            self.assertEqual(
                results[name]["review_status"],
                "accepted" if name == "retry-icon" else "pending_human_review",
            )
        tests = [
            call
            for call in commands.call_args_list
            if call.args[0] == ["npm", "run", "test:coverage"]
        ]
        self.assertEqual(len(tests), 3)

    def test_runner_fails_when_visual_check_misses_color_defect(self):
        with tempfile.TemporaryDirectory() as directory:
            workspace = Path(directory) / "run"
            passed, _ = self.run_mocked(workspace, hide_color_difference=True)
            results = json.loads((workspace / "results.json").read_text())
        self.assertFalse(passed)
        self.assertFalse(results["status-colors"]["detected_as_expected"])

    def test_capture_drift_blocks_grading(self):
        for options in ({"capture_width": 900}, {"reproduction": 1}):
            with (
                self.subTest(options=options),
                tempfile.TemporaryDirectory() as directory,
            ):
                workspace = Path(directory) / "run"
                with self.assertRaises(RuntimeError):
                    self.run_mocked(workspace, **options)
                self.assertFalse((workspace / "results.json").exists())

    def test_browser_crash_is_not_a_successful_negative(self):
        with tempfile.TemporaryDirectory() as directory:
            workspace = Path(directory) / "run"
            with self.assertRaisesRegex(RuntimeError, "browser crashed"):
                self.run_mocked(workspace, browser_error=True)
            self.assertFalse((workspace / "results.json").exists())


if __name__ == "__main__":
    unittest.main()
