import importlib.util
import hashlib
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch


sys.dont_write_bytecode = True
HERE = Path(__file__).parent


def driver():
    spec = importlib.util.spec_from_file_location("measure", HERE / "measure.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class SourceInventoryTests(unittest.TestCase):
    def inventory(self, module, checkout, names, links_path=None, gitlinks=None):
        with self.listing(module, names, gitlinks):
            return module.source_inventory(checkout, links_path) if links_path else module.source_inventory(checkout)

    def regular_entries(self, module, files):
        for name, expected in files.items():
            path = Path(name)
            self.assertTrue(stat.S_ISREG(path.lstat().st_mode), "regular-only manifest refused " + name)
            self.assertEqual(path, path.resolve(strict=True))
            self.assertEqual(expected, module.digest(path))

    def test_actual_tracked_symlinks_produce_regular_manifest_entries(self):
        module = driver()
        checkout = Path("/Users/stephan/Workspace/skunkworks/gomad/temporal")
        names = ["config/development.yaml", "config/development-sqlite.yaml",
                 "schema/elasticsearch/visibility/index_template_v7.json",
                 "schema/elasticsearch/visibility/versioned/v14/index_template_v7.json"]
        inventory = self.inventory(module, checkout, names)
        self.regular_entries(module, inventory["files"])
        self.assertEqual(len(inventory["files"]), 2)
        self.assertEqual([entry["path"] for entry in inventory["links"]], [names[0], names[2]])

    def fixture(self, directory):
        checkout = Path(directory) / "checkout"
        checkout.mkdir()
        (checkout / "target.txt").write_bytes(b"alpha\n")
        (checkout / "second.txt").write_bytes(b"alpha\n")
        (checkout / "ordinary file.txt").write_bytes(b"ordinary\n")
        (checkout / "alias.txt").symlink_to("target.txt")
        names = ["alias.txt", "target.txt", "second.txt", "ordinary file.txt"]
        evidence = Path(directory) / "evidence"
        evidence.mkdir()
        return checkout, evidence, names

    def listing(self, module, names, gitlinks=None):
        def output(argv, **kwargs):
            if "--stage" in argv:
                return b"".join(b"160000 " + oid.encode() + b" 0\t" + os.fsencode(name) + b"\0" for name, oid in (gitlinks or {}).items())
            return b"\0".join(os.fsencode(name) for name in names) + b"\0"
        return patch.object(module.subprocess, "check_output", side_effect=output)

    def test_actual_empty_tracked_gitlink_is_bound_without_regular_file_claim(self):
        module = driver()
        checkout = Path("/Users/stephan/Workspace/skunkworks/gomad/temporal")
        path = "develop/docker-compose/grafana/provisioning/temporalio-dashboards"
        oid = "590cbf37af8cef99387b2a1f88c163728b003ea4"
        inventory = self.inventory(module, checkout, [path], gitlinks={path: oid})
        self.assertEqual(inventory["files"], {})
        self.assertEqual(inventory["gitlinks"], [{"path": path, "mode": "160000", "oid": oid, "state": "empty-uninitialized"}])

    def test_gitlink_identity_and_empty_state_are_bound_and_rechecked_before_child(self):
        module = driver()
        for change in ("oid", "populated", "missing", "symlink", "unindexed"):
            with self.subTest(change=change), tempfile.TemporaryDirectory() as directory:
                checkout, evidence, names = self.fixture(directory)
                submodule = checkout / "submodule"
                submodule.mkdir()
                names.append("submodule")
                gitlinks = {"submodule": "1" * 40}
                with self.listing(module, names, gitlinks):
                    inventory, manifest = module.write_source_manifest(checkout, evidence)
                    source_sha = "sha256:" + module.digest(manifest)
                    self.assertNotIn(str(submodule), inventory["files"])
                    self.assertEqual(json.loads((evidence / "source-links.json").read_text())["gitlinks"],
                                     [{"path": "submodule", "mode": "160000", "oid": "1" * 40, "state": "empty-uninitialized"}])
                    self.regular_entries(module, inventory["files"])
                    if change == "oid":
                        gitlinks["submodule"] = "2" * 40
                    elif change == "populated":
                        (submodule / "added").write_text("changed\n")
                    elif change == "missing":
                        submodule.rmdir()
                    elif change == "symlink":
                        submodule.rmdir()
                        (checkout / "other").mkdir()
                        submodule.symlink_to("other")
                    else:
                        gitlinks.clear()
                    marker = evidence / "child-started"
                    with self.assertRaises(RuntimeError):
                        module.verify_source_inventory(checkout, inventory, manifest, source_sha)
                        subprocess.run([sys.executable, "-I", "-c", "from pathlib import Path;import sys;Path(sys.argv[1]).touch()", str(marker)], check=True)
                    self.assertFalse(marker.exists())

    def test_invalid_or_unmerged_index_metadata_refuses(self):
        module = driver()
        with tempfile.TemporaryDirectory() as directory:
            checkout, evidence, names = self.fixture(directory)
            for entry in (b"malformed\0", b"160000 invalid 0\tsubmodule\0",
                          b"160000 " + b"1" * 40 + b" 1\tsubmodule\0",
                          b"040000 " + b"1" * 40 + b" 0\tsubmodule\0"):
                with self.subTest(entry=entry):
                    def output(argv, **kwargs):
                        return entry if "--stage" in argv else b"\0".join(os.fsencode(name) for name in names) + b"\0"
                    with patch.object(module.subprocess, "check_output", side_effect=output), self.assertRaises(RuntimeError):
                        module.source_inventory(checkout)

    def test_manifest_binds_logical_map_and_only_canonical_regular_bytes(self):
        module = driver()
        with tempfile.TemporaryDirectory() as directory:
            checkout, evidence, names = self.fixture(directory)
            with self.listing(module, names):
                inventory, manifest = module.write_source_manifest(checkout, evidence)
                module.verify_source_inventory(checkout, inventory, manifest, "sha256:" + module.digest(manifest))
            links_path = evidence / "source-links.json"
            self.assertEqual(json.loads(links_path.read_text()), {
                "schema": "gomad.measurement-source-links/v1",
                "links": [{"path": "alias.txt", "target": "target.txt", "resolved": "target.txt",
                           "sha256": hashlib.sha256(b"alpha\n").hexdigest()}],
                "gitlinks": [],
            })
            entries = dict((path, sha) for sha, path in (line.split("  ", 1) for line in manifest.read_text().splitlines()))
            self.assertEqual(set(entries), {str(checkout / name) for name in names if name != "alias.txt"} | {str(links_path)})
            self.assertEqual(entries[str(links_path)], hashlib.sha256(links_path.read_bytes()).hexdigest())
            self.regular_entries(module, entries)

    def test_retarget_with_equal_bytes_and_raw_target_changes_are_detected(self):
        module = driver()
        with tempfile.TemporaryDirectory() as directory:
            checkout, evidence, names = self.fixture(directory)
            before = self.inventory(module, checkout, names)
            for target in ("second.txt", "./target.txt"):
                (checkout / "alias.txt").unlink()
                (checkout / "alias.txt").symlink_to(target)
                after = self.inventory(module, checkout, names)
                self.assertEqual(before["files"], after["files"])
                self.assertNotEqual(before["links"], after["links"])

    def test_contained_chain_binds_every_link_even_unlisted_intermediate(self):
        module = driver()
        with tempfile.TemporaryDirectory() as directory:
            checkout, evidence, names = self.fixture(directory)
            (checkout / "alias.txt").unlink()
            (checkout / "alias.txt").symlink_to("intermediate.txt")
            (checkout / "intermediate.txt").symlink_to("target.txt")
            before = self.inventory(module, checkout, names)
            self.assertEqual([entry["path"] for entry in before["links"]], ["alias.txt", "intermediate.txt"])
            (checkout / "intermediate.txt").unlink()
            (checkout / "intermediate.txt").symlink_to("./target.txt")
            after = self.inventory(module, checkout, names)
            self.assertEqual(before["files"], after["files"])
            self.assertNotEqual(before["links"], after["links"])

    def test_external_broken_loop_directory_special_and_missing_paths_refuse(self):
        module = driver()
        for kind in ("external", "broken", "loop", "directory", "special", "missing"):
            with self.subTest(kind=kind), tempfile.TemporaryDirectory() as directory:
                checkout, evidence, names = self.fixture(directory)
                path = checkout / "alias.txt"
                path.unlink()
                if kind == "external":
                    (Path(directory) / "outside").write_bytes(b"alpha\n")
                    path.symlink_to("../outside")
                elif kind == "broken":
                    path.symlink_to("absent")
                elif kind == "loop":
                    path.symlink_to("alias.txt")
                elif kind == "directory":
                    path.symlink_to(".")
                elif kind == "special":
                    os.mkfifo(path)
                with self.assertRaises(RuntimeError):
                    self.inventory(module, checkout, names)

    def test_changes_abort_before_dummy_child_dispatch(self):
        module = driver()
        for kind in ("retarget", "content", "missing source", "sidecar", "missing sidecar", "manifest"):
            with self.subTest(kind=kind), tempfile.TemporaryDirectory() as directory:
                checkout, evidence, names = self.fixture(directory)
                with self.listing(module, names):
                    inventory, manifest = module.write_source_manifest(checkout, evidence)
                    source_sha = "sha256:" + module.digest(manifest)
                    if kind == "retarget":
                        (checkout / "alias.txt").unlink()
                        (checkout / "alias.txt").symlink_to("second.txt")
                    elif kind == "content":
                        (checkout / "target.txt").write_bytes(b"changed\n")
                    elif kind == "missing source":
                        (checkout / "ordinary file.txt").unlink()
                    elif kind == "sidecar":
                        (evidence / "source-links.json").write_text("{}\n")
                    elif kind == "missing sidecar":
                        (evidence / "source-links.json").unlink()
                    else:
                        manifest.write_text("changed\n")
                    marker = evidence / "child-started"
                    with self.assertRaises((RuntimeError, OSError)):
                        module.verify_source_inventory(checkout, inventory, manifest, source_sha)
                        subprocess.run([sys.executable, "-I", "-c", "from pathlib import Path;import sys;Path(sys.argv[1]).touch()", str(marker)], check=True)
                    self.assertFalse(marker.exists())

    def test_existing_entry_and_byte_caps_stay_fail_closed(self):
        module = driver()
        with tempfile.TemporaryDirectory() as directory:
            checkout, evidence, names = self.fixture(directory)
            for constant, value in (("MAX_SOURCE_ENTRIES", 2), ("MAX_SOURCE_FILE_BYTES", 5), ("MAX_SOURCE_BYTES", 10)):
                with self.subTest(constant=constant), patch.object(module, constant, value):
                    with self.assertRaises(RuntimeError):
                        self.inventory(module, checkout, names)
            with patch.object(module, "MAX_SOURCE_MANIFEST_BYTES", 20), self.listing(module, names):
                with self.assertRaisesRegex(RuntimeError, "manifest byte limit"):
                    module.write_source_manifest(checkout, evidence)

    def test_empty_inventory_cannot_be_replaced_by_sidecar_only_manifest(self):
        module = driver()
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(RuntimeError):
                self.inventory(module, Path(directory), [])

    def test_sidecar_counts_against_existing_entry_and_byte_caps(self):
        module = driver()
        for constant, value in (("MAX_SOURCE_ENTRIES", 3), ("MAX_SOURCE_BYTES", 25)):
            with self.subTest(constant=constant), tempfile.TemporaryDirectory() as directory:
                checkout, evidence, names = self.fixture(directory)
                names.remove("alias.txt")
                with patch.object(module, constant, value), self.listing(module, names):
                    with self.assertRaises(RuntimeError):
                        module.write_source_manifest(checkout, evidence)


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
