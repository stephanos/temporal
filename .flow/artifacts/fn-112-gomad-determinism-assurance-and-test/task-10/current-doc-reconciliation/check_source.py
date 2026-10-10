import argparse
import datetime
import hashlib
import json
import pathlib
import re
import shutil
import subprocess
import sys
import time

TASK = "fn-112-gomad-determinism-assurance-and-test.10"
OLD = "eb82ea59a4db14a19ce07c9ced0ee3f8e4bcddd6"
PACKET = ".flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10"
GUIDES = ["tools/gomad3/" + name + ".md" for name in
          ["README", "SPEC", "ARCHITECTURE", "CLI", "TUTORIAL"]]
GUIDES += ["tools/gomad3integration/README.md", "AGENTS.md", "MILESTONES.md"]
PREFIXES = ["tools/gomad3/cmd/gomad/internal/cli/", "tools/gomad3/cmd/gomadtool/"]


def digest(data):
    return hashlib.sha256(data).hexdigest()


def git(*arguments):
    return subprocess.check_output(["git", *arguments])


def tokens(body):
    pattern = r'//[^\n]*|/\*[\s\S]*?\*/|"(?:\\.|[^"\\])*"|`[^`]*`|\x27(?:\\.|[^\x27\\])*\x27|[A-Za-z_][\w]*|\d+|[^\s]'
    return [match.group() for match in re.finditer(pattern, body)
            if not match.group().startswith(("//", "/*"))]


def flag_projection(body):
    source = tokens(body)
    calls = []
    for index in range(len(source) - 3):
        if (source[index] not in ("flag", "flags") or source[index + 1] != "."
                or source[index + 3] != "("):
            continue
        depth = 1
        end = index + 4
        while depth and end < len(source):
            depth += (source[end] == "(") - (source[end] == ")")
            end += 1
        if depth:
            raise ValueError("unclosed flag call")
        calls.append(source[index:end])
    return calls


def anchors(body):
    result = set(re.findall(r'<a\s+(?:id|name)=["\x27]([^"\x27]+)', body))
    counts = {}
    for heading in re.findall(r"^#{1,6}\s+(.+?)\s*#*$", body, re.M):
        slug = re.sub(r"[^\w\- ]", "", heading.lower()).replace(" ", "-")
        count = counts.get(slug, 0)
        counts[slug] = count + 1
        result.add(slug + ("-" + str(count) if count else ""))
    return result


parser = argparse.ArgumentParser()
parser.add_argument("--root", required=True)
parser.add_argument("--expected-head", required=True)
parser.add_argument("--output", required=True)
options = parser.parse_args()
started = datetime.datetime.now(datetime.timezone.utc).isoformat()
clock = time.monotonic()
root = pathlib.Path(options.root).resolve()
if pathlib.Path.cwd().resolve() != root:
    raise SystemExit("physical cwd differs from the assigned workspace")
if git("rev-parse", "--show-toplevel").decode().strip() != str(root):
    raise SystemExit("Git root differs from the assigned workspace")
head = git("rev-parse", "HEAD").decode().strip()
if head != options.expected_head:
    raise SystemExit("current HEAD differs from the explicit expected binding")
output = pathlib.Path(options.output)
if not output.resolve().is_relative_to(root / PACKET / "current-doc-reconciliation"):
    raise SystemExit("output is outside the assigned evidence directory")
if output.exists():
    raise SystemExit("refusing to overwrite an existing observation")
errors = []
bindings = {}
input_locations = {}
provenance_gaps = []


def read(path):
    if (root / path).is_file():
        data = (root / path).read_bytes()
        input_locations[path] = "physical worktree"
    else:
        data = git("show", head + ":" + path)
        input_locations[path] = "committed HEAD blob; absent from sparse worktree"
    bindings[path] = digest(data)
    return data


historical_path = PACKET + "/source-acceptance-20261009/sealed-source-audit.json"
historical = json.loads(read(historical_path))
continuation = json.loads(read(PACKET + "/continuation-20261009/evidence.json"))
if historical["errors"] or continuation["current_guides"]["errors"]:
    errors.append("the referenced historical guide inventory is not clean")
documents = {path: read(path).decode() for path in GUIDES}
prior_bindings = []
for path, expected in historical["inputs_sha256"].items():
    if pathlib.Path(path).is_absolute():
        prior_bindings.append({"path": path, "historical_binary_only": True,
                               "historical_sha256": expected, "executed": False})
        continue
    old = git("show", OLD + ":" + path)
    if digest(old) != expected:
        errors.append("historical source identity mismatch: " + path)
    current = read(path)
    prior_bindings.append({"path": path, "historical_sha256": expected,
                           "current_sha256": digest(current), "changed": old != current})

tracked = git("ls-files", "-z", *PREFIXES).decode().split("\0")
source_paths = sorted(path for path in tracked if path.endswith(".go") and not path.endswith("_test.go"))
source_comparisons = []
for path in source_paths:
    current = read(path)
    old = git("show", OLD + ":" + path)
    old_flags = flag_projection(old.decode())
    current_flags = flag_projection(current.decode())
    source_comparisons.append({"path": path, "historical_sha256": digest(old),
                               "current_sha256": digest(current), "changed": old != current,
                               "flag_calls": len(current_flags),
                               "flag_calls_including_defaults_and_help_equal": old_flags == current_flags})
    if old_flags != current_flags:
        errors.append("flag/help/default registration changed: " + path)
for path in ["tools/gomad3/cmd/gomad/main.go", "tools/gomad3/go.mod", "tools/gomad3/go.sum"]:
    read(path)

index = documents["tools/gomad3/CLI.md"].split("## Command index", 1)[1]
routes = []
pack = read("tools/gomad3/cmd/gomadtool/compatibility_pack.go").decode()
actions = re.findall(r'case "([a-z-]+)":', pack.split("func runCompatibilityPackDiscover", 1)[0])
for tool, dispatcher in [("gomad", "tools/gomad3/cmd/gomad/internal/cli/cli.go"),
                         ("gomadtool", "tools/gomad3/cmd/gomadtool/main.go")]:
    current = read(dispatcher).decode()
    old = git("show", OLD + ":" + dispatcher).decode()
    old_usage = re.search(r'const usage = ("(?:\\.|[^"\\])*"|`[^`]*`)', old).group(1)
    current_usage = re.search(r'const usage = ("(?:\\.|[^"\\])*"|`[^`]*`)', current).group(1)
    old_cases = re.findall(r'case "([a-z-]+)":', old)
    current_cases = re.findall(r'case "([a-z-]+)":', current)
    if old_usage != current_usage or old_cases != current_cases:
        errors.append("usage or dispatcher grammar changed: " + tool)
    section = index.split("### `" + tool + "`", 1)[1].split("\n### ", 1)[0]
    for name in re.findall(r'^\| `([^`]+)` \|', section, re.M):
        parts = name.split()
        if parts[0] not in current_cases:
            errors.append("undispatched indexed command: " + tool + " " + name)
        variants = [["compatibility-pack", action] for action in actions] if name == "compatibility-pack" else [parts]
        routes.extend([[tool, *variant, "-h"] for variant in variants])
historical_routes = [entry["argv"] for entry in historical["command_help_inventory"]]
if sorted(routes) != sorted(historical_routes):
    errors.append("indexed help routes differ from retained observations")
observations = []
for entry in historical["command_help_inventory"]:
    name = "-".join(entry["argv"][:-1])
    if entry["argv"] == ["gomadtool", "compatibility-pack", "refresh", "-h"]:
        name = "gomadtool-compatibility-pack refresh"
    path = PACKET + "/source-acceptance-20261009/final-guides/" + name + ".help.txt"
    raw = read(path)
    raw_matches = digest(raw) == entry["help_sha256"]
    known_refresh_gap = (
        entry["argv"] == ["gomadtool", "compatibility-pack", "refresh", "-h"]
        and entry["help_sha256"] == "42c826d83aea0bcd282e9cab26f5026bdd076f2c551cba950280536e9a2e5443"
        and digest(raw) == "b403700ca838778dc8680b891cdeae2b2f4f16ebe51824aeb11e01b7fcc91165")
    if not raw_matches and not known_refresh_gap:
        errors.append("retained help capture mismatch: " + path)
    if known_refresh_gap:
        provenance_gaps.append({"argv": entry["argv"], "audit_help_sha256": entry["help_sha256"],
                                "physical_capture_sha256": digest(raw),
                                "reason": "Historical inline refresh help observation has no matching raw capture. The retained fn111 capture contains top-level usage after its stale action split."})
    observations.append({"argv": entry["argv"], "retained_exit": entry["status"],
                         "capture": path, "capture_sha256": digest(raw),
                         "audit_help_sha256": entry["help_sha256"], "raw_capture_matches": raw_matches,
                         "fresh_execution": False})
all_help_flags = {name for entry in historical["command_help_inventory"] for name in entry["flags"]}
target_parser = read("tools/gomad3/cmd/gomad/internal/cli/cli.go").decode()
positional = set(re.findall(r'arguments\[1\] != "--([a-z-]+)"', target_parser))
documented_flags = set(re.findall(r"--([a-z][a-z0-9-]*)", "\n".join(documents[path] for path in GUIDES[:5])))
if documented_flags - all_help_flags - positional:
    errors.append({"documented_flags_without_retained_help_or_positional_parser":
                   sorted(documented_flags - all_help_flags - positional)})

links = []
for path, body in documents.items():
    for destination in re.findall(r"\[[^\]]*\]\(([^)]+)\)", body):
        if "://" in destination or destination.startswith("mailto:"):
            continue
        name, _, fragment = destination.partition("#")
        target = (root / path).parent / name if name else root / path
        target = target.resolve()
        present = target.is_file()
        if not target.is_relative_to(root):
            errors.append("local link escapes workspace: " + path + " " + destination)
            continue
        relative = str(target.relative_to(root))
        try:
            target_bytes = read(relative)
        except subprocess.CalledProcessError:
            target_bytes = None
        resolves = target_bytes is not None and (not fragment or fragment in anchors(target_bytes.decode()))
        links.append({"source": path, "destination": destination, "committed_or_worktree_resolves": resolves,
                      "physical_worktree_present": present})
        if not resolves:
            errors.append(links[-1])

for path in ["Makefile", ".github/workflows/gomad3.yml", ".github/workflows/gomad3-smoke.yml",
             "tools/gomad3integration/qualification/soak.json",
             "tools/gomad3integration/qualification/smoke.json",
             "tools/gomad3integration/qualification/tests.json"]:
    if read(path) != git("show", OLD + ":" + path):
        errors.append("retained scheduling/selection input changed: " + path)
guide_links = [link for link in links if link["source"] in GUIDES[:6]]
proof_root = ".flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/"
proofs = {}
for number in [52, 53, 54, 55]:
    names = (["reconciled-source-proof.json", "reconciled-evidence.json", "handover.md"]
             if number == 55 else ["source-proof.json", "evidence.json", "handover.md"])
    for name in names:
        path = proof_root + "task-" + str(number) + "/" + name
        contents = read(path)
        if name.endswith("json"):
            proofs[(number, name)] = json.loads(contents)

proof52 = proofs[(52, "source-proof.json")]
evidence52 = proofs[(52, "evidence.json")]
production_bindings = []
for item in proof52["files"]:
    path = item["file"]
    expected = evidence52["bounded_source_sha256"][pathlib.Path(path).name]
    current = digest(read(path))
    complete = item["complete_original_bytes_restored_after_removing_only_admitted_checks"]
    if current != expected or not complete or not item["all_fmt_operands_and_order_identical"]:
        errors.append("task52 source proof no longer binds: " + path)
    production_bindings.append({"path": path, "proof_owner": "fn109.52", "candidate_sha256": expected,
                                "current_sha256": current, "admitted_stderr_checks": item["admitted_writes"],
                                "proof_claims_original_byte_reconstruction": complete})
proof53 = proofs[(53, "source-proof.json")]
for item in proof53["production_files"]:
    path = item["path"]
    candidate = (git("show", "8177ddec76f30c1db027f282358a213176855ef8:" + path)
                 if path.endswith("/cli.go") else read(path))
    if digest(candidate) != item["candidate_sha256"] or not item["complete_original_bytes_reconstructed"]:
        errors.append("task53 source proof no longer binds: " + path)
    if digest(git("show", proof53["base_commit"] + ":" + path)) != item["base_sha256"]:
        errors.append("task53 base source identity mismatch: " + path)
    production_bindings.append({"path": path, "proof_owner": "fn109.53", "candidate_sha256": item["candidate_sha256"],
                                "current_sha256": digest(read(path)), "admitted_stderr_checks": item["admitted_checks"],
                                "intermediate_candidate": path.endswith("/cli.go"),
                                "runtime_gap_count": sum(bool(site["runtime_gap"]) for site in item["sites"])})
cli_path = "tools/gomad3/cmd/gomad/internal/cli/cli.go"
proof54 = proofs[(54, "source-proof.json")]
proof55 = proofs[(55, "reconciled-source-proof.json")]
if (proof54["base_sha256"] != next(item["candidate_sha256"] for item in proof53["production_files"] if item["path"] == cli_path)
        or proof54["candidate_sha256"] != proof55["base_sha256"]
        or proof55["candidate_sha256"] != digest(read(cli_path))
        or not proof54["complete_original_bytes_reconstructed"]
        or not proof55["complete_original_cli_bytes_reconstructed"]):
    errors.append("doctor/replay proof chain does not bind current CLI")
if digest(git("show", "a2b020a178bd8cae92016d2c8cd4cdca81023d23:" + cli_path)) != proof54["candidate_sha256"]:
    errors.append("task54 committed candidate differs from its proof")
control_tests = [
    ("tools/gomad3/cmd/gomadtool/terminal_diagnostics_test.go", evidence52["bounded_source_sha256"]["terminal_diagnostics_test.go"]),
    ("tools/gomad3/cmd/gomad/internal/cli/terminal_diagnostics_test.go",
     digest(git("show", "8177ddec76f30c1db027f282358a213176855ef8:tools/gomad3/cmd/gomad/internal/cli/terminal_diagnostics_test.go"))),
    ("tools/gomad3/cmd/gomad/internal/cli/doctor_output_test.go",
     digest(git("show", "a2b020a178bd8cae92016d2c8cd4cdca81023d23:tools/gomad3/cmd/gomad/internal/cli/doctor_output_test.go"))),
    ("tools/gomad3/cmd/gomad/internal/cli/replay_output_test.go", proof55["new_test_sha256"]),
    (proof55["characterization_path"], proof55["characterization_candidate_sha256"]),
]
for path, expected in control_tests:
    if digest(read(path)) != expected:
        errors.append("retained control source no longer matches current bytes: " + path)
control_bindings = []
for number, name, expected_exit in [
    (52, "preservation-before-final", 0), (52, "preservation-after", 0),
    (53, "preservation-before-final", 0), (53, "preservation-after", 0),
    (54, "focused-before-final", 1), (54, "focused-after", 0),
    (55, "focused-red-final", 1), (55, "reconciled-focused", 0),
]:
    prefix = proof_root + "task-" + str(number) + "/" + name
    receipt = json.loads(read(prefix + ".json"))
    for stream in ["stdout", "stderr"]:
        if digest(read(prefix + "." + stream)) != receipt[stream + "_sha256"]:
            errors.append("retained adapter control raw hash mismatch: " + prefix + "." + stream)
    if receipt["exit_code"] != expected_exit or not receipt["source_unchanged"] or not receipt["terminal"]:
        errors.append("retained adapter control status/source/terminal mismatch: " + prefix)
    counts = {"pass": 0, "fail": 0, "skip": 0}
    for line in read(prefix + ".stdout").decode().splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if event.get("Test") and event.get("Action") in counts:
            counts[event["Action"]] += 1
    if not sum(counts.values()) or (expected_exit == 0 and (counts["fail"] or counts["skip"])):
        errors.append("retained adapter control collection/outcome mismatch: " + prefix)
    control_bindings.append({"receipt": prefix + ".json", "command": receipt["command"],
                             "retained_exit": receipt["exit_code"], "test_observations": counts,
                             "elapsed_seconds": receipt["elapsed_seconds"],
                             "raw_stream_hashes_verified": True, "fresh_execution": False})
for path, expected in list(bindings.items()):
    current = ((root / path).read_bytes() if input_locations[path] == "physical worktree"
               else git("show", head + ":" + path))
    if digest(current) != expected:
        errors.append("input changed during source inventory: " + path)
result = {"task": TASK, "status": "in_progress", "head": head, "historical_source": OLD,
          "cwd": str(root), "started_at": started,
          "ended_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
          "elapsed_seconds": round(time.monotonic() - clock, 3), "errors": errors,
          "script_sha256": digest(pathlib.Path(__file__).read_bytes()),
          "python": sys.executable, "python_sha256": digest(pathlib.Path(sys.executable).read_bytes()),
          "git": shutil.which("git"), "git_sha256": digest(pathlib.Path(shutil.which("git")).read_bytes()),
          "inputs_sha256": bindings, "input_locations": input_locations,
          "prior_audit_input_bindings": prior_bindings, "provenance_gaps": provenance_gaps,
          "command_source_projection": source_comparisons, "retained_help_observations": observations,
          "raw_bound_help_observations": sum(item["raw_capture_matches"] for item in observations),
          "hash_only_help_observations": sum(not item["raw_capture_matches"] for item in observations),
          "adapter_proof_bindings": production_bindings,
          "retained_control_bindings": control_bindings,
          "stdout_failure_changes": {"doctor_checks": proof54["three_checks"],
                                     "verify_only_replay_failed_write_status": proof55["single_check_failure_status"]},
          "help_observation_count": len(observations),
          "unique_help_routes": len({tuple(route) for route in routes}),
          "positional_target_flags": sorted(positional), "documented_flags": sorted(documented_flags),
          "guide_links_count": len(guide_links), "expanded_links_count": len(links), "links": links,
          "historical_fn111_exit": 1, "historical_fn111_diagnostics": 54,
          "fresh_binary_execution": False, "go_commands_executed": False,
          "native_bound": None, "native_qualification_claim": False,
          "coverage_limit": "Source inventory and retained healthy help projection only. Adapter status/output controls are reconciled separately; no binary or aggregate source-gate pass is inferred."}
with output.open("x") as stream:
    json.dump(result, stream, indent=2)
    stream.write("\n")
print(json.dumps({"errors": errors, "head": head, "source_files": len(source_paths),
                  "retained_help_observations": len(observations), "unique_help_routes": result["unique_help_routes"],
                  "guide_links": len(guide_links), "expanded_links": len(links),
                  "elapsed_seconds": result["elapsed_seconds"]}))
sys.exit(bool(errors))
