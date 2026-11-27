"""Check the packaged Alfred Configure layouts and exercise the shipped launcher."""
import itertools
import json
import os
from pathlib import Path
import plistlib
import subprocess
import tempfile
import time
import unittest
import zipfile


ARCHIVE = Path(__file__).resolve().parents[1] / "NaverSearchPlus.alfredworkflow"


class UnifiedPackageTests(unittest.TestCase):
    def test_configured_layouts(self):
        with zipfile.ZipFile(ARCHIVE) as archive:
            plist = plistlib.loads(archive.read("info.plist"))
            template, = (obj["config"]["path"] for obj in plist["objects"]
                         if obj["type"] == "alfred.workflow.userinterface.alfredui")
            config = {item["variable"]: item["config"] for item in plist["userconfigurationconfig"]
                      if "variable" in item}
            keys = ("nn_width", "nn_font", "nn_rows")
            choices = [[value for _, value in config[key]["pairs"]] for key in keys]
            expected = set()
            for values in itertools.product(*choices):
                layout = template
                for key, value in zip(keys, values):
                    layout = layout.replace("{var:" + key + "}", value)
                expected.add(layout + "/ui.json")
                ui = json.loads(archive.read(layout + "/ui.json"))
                self.assertNotIn("{file:", json.dumps(ui))
                variables = ui["variables"]
                self.assertEqual(variables["nn_list_height"], int(values[2]) * variables["nn_row_height"])
                self.assertTrue(archive.getinfo(layout + "/run").external_attr >> 16 & 0o111)
            actual = {name for name in archive.namelist() if name.endswith(".alfredui/ui.json")}
            self.assertEqual(expected, actual)

    def test_empty_launch_runs_daily_cleanup(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            subprocess.run(["unzip", "-q", str(ARCHIVE)], cwd=root, check=True)
            cache = root / "cache"
            cache.mkdir()
            stale = cache / "old-query.json"
            stale.touch()
            os.utime(stale, (0, 0))
            # Keep the packaged worker offline while checking its daily cleanup.
            (root / "update-settings.json").write_text('{"automatic_updates":false}')
            env = dict(os.environ, alfred_workflow_cache=str(cache), alfred_workflow_data=tmp,
                       nn_resume="0")
            launcher = root / "nui-current-default-6.alfredui" / "run"
            args = [launcher, "nui", "start", ""]
            self.assertEqual(json.loads(subprocess.check_output(args, env=env)), {})
            deadline = time.monotonic() + 5
            while stale.exists() and time.monotonic() < deadline:
                time.sleep(0.02)
            self.assertFalse(stale.exists())
            stamp = cache / "__worker_run.json"
            first = stamp.stat().st_mtime_ns
            subprocess.run(args, env=env, check=True, stdout=subprocess.PIPE)
            self.assertEqual(stamp.stat().st_mtime_ns, first)

            (root / "update-settings.json").write_text('{"automatic_updates":true}')
            (cache / "__update_info.json").write_text('{"version":"v3.3.6"}')
            env["alfred_workflow_version"] = "3.3.5"
            started = json.loads(subprocess.check_output([launcher, "nui", "start", "서울"], env=env))
            views = started["update"]["ids"]
            self.assertIn("v3.3.6", views["service_name"]["text"])
            row, = views["results"]["data"]["src"]
            self.assertEqual(row["completion"], "서울")
            self.assertEqual(views["results"]["select"], row["uid"])


if __name__ == "__main__":
    unittest.main()
