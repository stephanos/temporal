import contextlib
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tarfile
import tempfile
from unittest.mock import patch

BASE = Path(__file__).resolve().parent
TAR_CWD = BASE / "tar-cwd"
TAR_CWD.mkdir(exist_ok=True)
RUN_ROOT = Path(tempfile.mkdtemp(prefix="old-runs-", dir=BASE))
MEMBER = "apalache-0.62.4/lib/apalache.jar"
JAR = b"local installer test jar"
ORIGINAL_RUN = subprocess.run
source = ORIGINAL_RUN(["git", "show", "94e47d7878:model/scripts/fetch-apalache.py"],
                      capture_output=True, check=True).stdout
namespace = {"__name__": "archive_research", "__file__": str(BASE / "old-installer.py")}
exec(compile(source, "old-installer-at-94e47d7878.py", "exec"), namespace)


def entry(name=MEMBER, data=JAR, kind=tarfile.REGTYPE, link="", pax=None):
    member = tarfile.TarInfo(name)
    member.type = kind
    member.linkname = link
    member.size = len(data) if kind in (tarfile.REGTYPE, tarfile.CONTTYPE) else 0
    member.uid = member.gid = 0
    member.uname = "spoof\n-rw-r--r-- 0/0 1 1970-01-01 00:00 " + MEMBER
    member.gname = "spoof\towner"
    member.pax_headers = pax or {}
    return member, data


def raw_archive(entries, fmt=tarfile.PAX_FORMAT):
    target = io.BytesIO()
    with tarfile.open(fileobj=target, mode="w:", format=fmt) as writer:
        for member, data in entries:
            writer.addfile(member, io.BytesIO(data) if member.size else None)
    return target.getvalue()


fixtures = {
    "regular": gzip.compress(raw_archive([entry()])),
    "symlink": gzip.compress(raw_archive([entry(kind=tarfile.SYMTYPE, link="outside")])),
    "hardlink": gzip.compress(raw_archive([entry(name="payload"), entry(kind=tarfile.LNKTYPE, link="payload")])),
    "directory": gzip.compress(raw_archive([entry(kind=tarfile.DIRTYPE)])),
    "fifo": gzip.compress(raw_archive([entry(kind=tarfile.FIFOTYPE)])),
    "contiguous_regular": gzip.compress(raw_archive([entry(kind=tarfile.CONTTYPE)])),
    "missing": gzip.compress(raw_archive([entry(name="unrelated")])),
    "duplicate_last_good": gzip.compress(raw_archive([entry(data=b"wrong"), entry()])),
    "duplicate_last_wrong": gzip.compress(raw_archive([entry(), entry(data=b"wrong")])),
    "duplicate_link_then_regular": gzip.compress(raw_archive([entry(kind=tarfile.SYMTYPE, link="outside"), entry()])),
    "duplicate_regular_then_link": gzip.compress(raw_archive([entry(), entry(kind=tarfile.SYMTYPE, link="outside")])),
    "pax_path_exact": gzip.compress(raw_archive([entry(name="renamed", pax={"path": MEMBER})])),
    "pax_path_other": gzip.compress(raw_archive([entry(pax={"path": "unrelated"})])),
    "pax_control_other_plus_regular": gzip.compress(raw_archive([entry(name="other\n" + MEMBER), entry()])),
    "pax_link_newline_spoof": gzip.compress(raw_archive([entry(kind=tarfile.SYMTYPE, link="outside\n-rw-r--r-- 0/0 23 1970-01-01 00:00 " + MEMBER)])),
    "regular_with_slash": gzip.compress(raw_archive([entry(name=MEMBER + "/")])),
    "leading_dot": gzip.compress(raw_archive([entry(name="./" + MEMBER)])),
    "leading_slash": gzip.compress(raw_archive([entry(name="/" + MEMBER)])),
    "child_only": gzip.compress(raw_archive([entry(name=MEMBER + "/child")])),
    "gnu_regular": gzip.compress(raw_archive([entry()], tarfile.GNU_FORMAT)),
    "unknown_type": gzip.compress(raw_archive([entry(kind=b"Z")])),
    "wrong_jar": gzip.compress(raw_archive([entry(data=b"wrong")])),
    "unrelated_traversal_plus_regular": gzip.compress(raw_archive([entry(name="../outside-sentinel", data=b"tampered"), entry()])),
    "selected_link_absolute": gzip.compress(raw_archive([entry(kind=tarfile.SYMTYPE, link="/outside-sentinel")])),
}
raw = bytearray(raw_archive([entry()]))
raw[0] ^= 1
fixtures["bad_tar_checksum"] = gzip.compress(raw)
raw = bytearray(fixtures["regular"])
raw[-8] ^= 1
fixtures["bad_gzip_crc"] = bytes(raw)
fixtures["truncated_gzip"] = fixtures["regular"][:-12]
fixtures["trailing_gzip_garbage"] = fixtures["regular"] + b"garbage"


def invoke(argv):
    result = ORIGINAL_RUN(argv, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                          env=dict(os.environ, LC_ALL="C", TZ="UTC", TAR_OPTIONS=""),
                          cwd=TAR_CWD, timeout=5, check=False)
    return {"argv": argv, "exit": result.returncode,
            "stdout_hex": result.stdout.hex(), "stderr_hex": result.stderr.hex(),
            "stdout_repr": repr(result.stdout), "stderr_repr": repr(result.stderr)}


def old_installer(name, payload, mock_pin):
    root = RUN_ROOT / (name + ("-mock" if mock_pin else "-production"))
    root.mkdir(parents=True, exist_ok=True)
    def fetch(argv, **kwargs):
        Path(argv[-1]).write_bytes(payload)
    changes = {"JAR_SHA256": hashlib.sha256(JAR).hexdigest()}
    if mock_pin:
        changes["ARCHIVE_SHA256"] = hashlib.sha256(payload).hexdigest()
    try:
        with patch.dict(namespace, changes), patch.object(subprocess, "run", fetch):
            output = namespace["install"](root)
        return {"accepted": True, "contents_hex": output.read_bytes().hex()}
    except Exception as error:
        return {"accepted": False, "error_type": type(error).__name__, "error": str(error)}


results = {"platform": sys.platform, "python": sys.version,
           "tar_version": invoke(["tar", "--version"]),
           "gzip_version": invoke(["gzip", "--version"]),
           "bsdtar_path": shutil.which("bsdtar"), "cases": {}}
(BASE / "outside-sentinel").write_bytes(b"protected")
for name, payload in fixtures.items():
    archive = BASE / "archives" / (name + ".tgz")
    archive.parent.mkdir(parents=True, exist_ok=True)
    archive.write_bytes(payload)
    common = ["--numeric-owner", "--no-recursion", "--no-wildcards", "--anchored"]
    commands = {
        "list_exact": ["tar", "--list", "--verbose", *common, "--file", str(archive), "--", MEMBER],
        "stdout_exact": ["tar", "--extract", "--to-stdout", *common, "--file", str(archive), "--", MEMBER],
    }
    for occurrence in (1, 2, 3):
        commands["list_occurrence_" + str(occurrence)] = [
            "tar", "--list", "--verbose", *common, "--occurrence=" + str(occurrence),
            "--file", str(archive), "--", MEMBER]
        commands["stdout_occurrence_" + str(occurrence)] = [
            "tar", "--extract", "--to-stdout", *common, "--occurrence=" + str(occurrence),
            "--file", str(archive), "--", MEMBER]
    results["cases"][name] = {
        "sha256": hashlib.sha256(payload).hexdigest(),
        "old_production_pin": old_installer(name, payload, False),
        "old_mock_pin": old_installer(name, payload, True),
        "commands": {key: invoke(argv) for key, argv in commands.items()},
    }
results["extraction_safety"] = {
    "stdout_only_cwd_contents": sorted(path.name for path in TAR_CWD.iterdir()),
    "outside_sentinel_contents_hex": (BASE / "outside-sentinel").read_bytes().hex(),
}
(BASE / "results.json").write_text(json.dumps(results, indent=2) + "\n")
for name, result in results["cases"].items():
    old = result["old_mock_pin"]
    listing = result["commands"]["list_exact"]
    extraction = result["commands"]["stdout_exact"]
    print(name, json.dumps({"old": old, "list_exit": listing["exit"],
          "list": listing["stdout_repr"], "extract_exit": extraction["exit"],
          "extract": extraction["stdout_repr"]}))
