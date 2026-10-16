import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch


HERE = Path(__file__).parent


def driver():
    spec = importlib.util.spec_from_file_location("measure", HERE / "measure.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class EnvironmentTests(unittest.TestCase):
    def inherited(self):
        return {
            "PATH": "/usr/bin:/bin", "HOME": "/ordinary/home", "LANG": "C.UTF-8",
            "GOCACHE": "/ordinary/cache", "GOMODCACHE": "/ordinary/modcache",
            "GOPATH": "/ordinary/go", "XDG_CACHE_HOME": "/ordinary/xdg",
            "GOPROXY": "https://proxy.golang.org,direct", "GOSUMDB": "sum.golang.org",
            "HTTPS_PROXY": "http://proxy.example:8080", "NO_PROXY": "localhost,.example",
            "UNRELATED_CREDENTIAL": "synthetic-private-token",
            "DATABASE_PASSWORD": "synthetic-private-password", "AWS_SESSION_TOKEN": "synthetic-session",
            "GOMAD_OTHER_OVERRIDE": "uncontrolled", "GOMAD_WASM_TEST_HELPER": "/uncontrolled/helper",
            "GOFLAGS": "-modfile=/uncontrolled", "GOENV": "/uncontrolled/env",
            "GOROOT": "/uncontrolled/root", "GOEXPERIMENT": "uncontrolled",
            "GOMAXPROCS": "99", "GOMEMLIMIT": "100KiB", "GOTOOLCHAIN": "auto",
            "GOWORK": "/uncontrolled/work", "GOOS": "uncontrolled", "GOARCH": "uncontrolled",
            "BASH_ENV": "/uncontrolled/shell", "LD_PRELOAD": "/uncontrolled/library",
            "PYTHONPATH": "/uncontrolled/python", "CGO_ENABLED": "1",
        }

    def test_credentials_and_overrides_reach_neither_child_nor_evidence(self):
        module = driver()
        environment = module.gate_environment(self.inherited())
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            argv = [sys.executable, "-I", "-c", "import json,os;print(json.dumps(dict(os.environ)))"]
            module.write_json(root / "command.json", {"argv": argv, "environment": environment})
            child = json.loads(subprocess.check_output(argv, env=environment, text=True))
            evidence = json.loads((root / "command.json").read_text())
        for actual in (environment, child, evidence["environment"]):
            for key in ("UNRELATED_CREDENTIAL", "DATABASE_PASSWORD", "AWS_SESSION_TOKEN",
                        "GOMAD_OTHER_OVERRIDE", "GOMAD_WASM_TEST_HELPER", "GOROOT", "GOEXPERIMENT",
                        "GOMEMLIMIT", "GOOS", "GOARCH", "BASH_ENV", "LD_PRELOAD",
                        "PYTHONPATH", "CGO_ENABLED"):
                self.assertNotIn(key, actual)
            for key in ("PATH", "HOME", "GOCACHE", "GOMODCACHE", "GOPATH", "XDG_CACHE_HOME",
                        "GOPROXY", "GOSUMDB", "HTTPS_PROXY", "NO_PROXY"):
                self.assertEqual(actual[key], self.inherited()[key])
            for key, value in {"GOENV": "off", "GOTOOLCHAIN": "local", "GOWORK": "off", "GOFLAGS": "", "GOMAXPROCS": "2"}.items():
                self.assertEqual(actual[key], value)
            self.assertNotIn("synthetic-private", json.dumps(actual))

    def test_auxiliary_commands_do_not_inherit_credentials(self):
        module = driver()
        with patch.dict(os.environ, self.inherited(), clear=True):
            output = module.command([sys.executable, "-I", "-c", "import json,os;print(json.dumps(dict(os.environ)))"], None)
        child = json.loads(output)
        self.assertNotIn("UNRELATED_CREDENTIAL", child)
        self.assertNotIn("GOMAD_OTHER_OVERRIDE", child)
        self.assertEqual(child["GOENV"], "off")
        self.assertEqual(child["GOMAXPROCS"], "2")

    def test_auth_bearing_proxy_settings_fail_without_echoing_values(self):
        module = driver()
        for key, value in (
            ("GOPROXY", "https://user:synthetic-private-token@proxy.example,direct"),
            ("GOSUMDB", "sum.example https://user:synthetic-private-token@sum.example"),
            ("HTTP_PROXY", "user:synthetic-private-token@proxy.example:8080"),
            ("HTTPS_PROXY", "https://proxy.example?token=synthetic-private-token"),
            ("ALL_PROXY", "https://proxy.example#synthetic-private-token"),
            ("https_proxy", "https://user%3Asynthetic-private-token@proxy.example"),
            ("NO_PROXY", "https://user:synthetic-private-token@proxy.example"),
        ):
            with self.subTest(key=key):
                inherited = {"PATH": "/usr/bin:/bin", key: value}
                with self.assertRaises(ValueError) as raised:
                    module.gate_environment(inherited)
                self.assertIn(key, str(raised.exception))
                self.assertNotIn(value, str(raised.exception))
                self.assertNotIn("synthetic-private-token", str(raised.exception))

    def test_missing_path_and_malformed_proxy_constraints_fail_closed(self):
        module = driver()
        for change in ({"PATH": ""}, {"PATH": "relative/bin"},
                       {"HTTPS_PROXY": "https://"}, {"HTTP_PROXY": "https://[broken"},
                       {"GOPROXY": "https://proxy.example,,direct"},
                       {"GOPROXY": "https://proxy.example:invalid"},
                       {"GOPROXY": "socks5://proxy.example:1080"},
                       {"GOPROXY": "file:///ordinary/proxy with spaces"},
                       {"GOSUMDB": " "},
                       {"GOSUMDB": "sum.example https://sum.example extra"}):
            with self.subTest(change=change):
                inherited = {"PATH": "/usr/bin:/bin", **change}
                with self.assertRaises(ValueError):
                    module.gate_environment(inherited)
        with self.assertRaisesRegex(ValueError, "PATH"):
            module.gate_environment({})

    def test_proxy_lists_sumdb_and_ordinary_system_settings_survive(self):
        module = driver()
        inherited = {
            "PATH": "/usr/bin:/bin", "GOPROXY": "https://one.example|https://two.example,direct",
            "GOSUMDB": "sum.example https://sum.example", "ALL_PROXY": "socks5://proxy.example:1080",
            "no_proxy": "localhost,127.0.0.1", "SSL_CERT_FILE": "/etc/ssl/cert.pem",
            "SSL_CERT_DIR": "/etc/ssl/certs", "USER": "ordinary", "LOGNAME": "ordinary",
            "TZ": "UTC", "LC_ALL": "C.UTF-8", "TMP": "/tmp", "TEMP": "/tmp",
            "GOPRIVATE": "private.example/*", "GONOPROXY": "private.example/*", "GONOSUMDB": "private.example/*",
        }
        environment = module.gate_environment(inherited)
        for key, value in inherited.items():
            self.assertEqual(environment[key], value)
        for setting in ("direct", "off", "file:///ordinary/proxy"):
            self.assertEqual(module.gate_environment({"PATH": "/usr/bin", "GOPROXY": setting})["GOPROXY"], setting)

    def test_refused_proxy_prevents_even_an_auxiliary_child_launch(self):
        module = driver()
        with tempfile.TemporaryDirectory() as directory:
            marker = Path(directory) / "launched"
            argv = [sys.executable, "-I", "-c", "from pathlib import Path;import sys;Path(sys.argv[1]).touch()", str(marker)]
            with patch.dict(os.environ, {"PATH": "/usr/bin", "HTTPS_PROXY": "https://u:synthetic-private-token@proxy.example"}, clear=True):
                with self.assertRaisesRegex(ValueError, "HTTPS_PROXY"):
                    module.command(argv, None)
            self.assertFalse(marker.exists())

    def test_default_schedule_preserves_all_sixteen_cells_and_abbaab_order(self):
        module = driver()
        self.assertEqual(module.schedule(3), [
            ("base-onebox-rep1", "base", "onebox", "success"),
            ("cand-onebox-rep1", "cand", "onebox", "success"),
            ("cand-onebox-rep2", "cand", "onebox", "success"),
            ("base-onebox-rep2", "base", "onebox", "success"),
            ("base-onebox-rep3", "base", "onebox", "success"),
            ("cand-onebox-rep3", "cand", "onebox", "success"),
            ("base-multi-instance-rep1", "base", "multi-instance", "success"),
            ("cand-multi-instance-rep1", "cand", "multi-instance", "success"),
            ("cand-multi-instance-rep2", "cand", "multi-instance", "success"),
            ("base-multi-instance-rep2", "base", "multi-instance", "success"),
            ("base-multi-instance-rep3", "base", "multi-instance", "success"),
            ("cand-multi-instance-rep3", "cand", "multi-instance", "success"),
            ("base-onebox-canary", "base", "onebox", "canary"),
            ("cand-onebox-canary", "cand", "onebox", "canary"),
            ("base-multi-instance-canary", "base", "multi-instance", "canary"),
            ("cand-multi-instance-canary", "cand", "multi-instance", "canary"),
        ])


if __name__ == "__main__":
    unittest.main()
