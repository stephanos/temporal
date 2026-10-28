#!/usr/bin/env python3
"""Project fn-258's fixed history in an independently created disposable clone.

The caller creates a detached `git clone --no-local --revision=<source>` first.
This command neither clones nor adapts the exported layout. Historical support
remains under .history-support for a subsequent, separately reviewed commit.
"""

import argparse
import functools
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys


SOURCE = "ac60daaf47d2513b41eeff612f0e2758d33e186d"
REDIRECT_ENV = (
    "GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_OBJECT_DIRECTORY",
    "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_INDEX_FILE", "GIT_NAMESPACE",
    "GIT_REPLACE_REF_BASE", "GIT_CONFIG", "GIT_CONFIG_COUNT",
)


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def git(repo, *args, check=True):
    result = subprocess.run(
        ["git", "--no-replace-objects", "-C", str(repo), *args],
        stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False,
    )
    if check and result.returncode:
        raise RuntimeError(
            f"git {args[0]} failed ({result.returncode}): "
            + result.stderr.decode(errors="replace").strip()
        )
    return result.stdout if check else result


def digest(data):
    return hashlib.sha256(data).hexdigest()


def write_json(path, value):
    with path.open("x") as stream:
        json.dump(value, stream, indent=2)
        stream.write("\n")


def worktrees(repo):
    return {
        Path(os.fsdecode(field[len(b"worktree "):])).resolve()
        for field in git(repo, "worktree", "list", "--porcelain", "-z").split(b"\0")
        if field.startswith(b"worktree ")
    }


def source_identity(repo):
    refs = git(repo, "for-each-ref", "--format=%(refname) %(objectname)")
    shallow_path = Path(os.fsdecode(git(
        repo, "rev-parse", "--path-format=absolute", "--git-path", "shallow"
    ).strip()))
    return {
        "head": git(repo, "rev-parse", "HEAD").decode().strip(),
        "refs_sha256": digest(refs),
        "worktrees": sorted(str(path) for path in worktrees(repo)),
        "shallow_file_sha256": digest(shallow_path.read_bytes()) if shallow_path.exists() else None,
    }


def guard_repositories(original, clone):
    for variable in REDIRECT_ENV:
        require(not os.environ.get(variable), f"Unset {variable} before running")
    require(original != clone, "The original repository cannot be filtered")
    require(not original.is_relative_to(clone), "Clone cannot contain the original")
    registered = worktrees(original)
    require(clone not in registered, "Clone is an original registered worktree")
    for repo in (original, clone):
        top = Path(os.fsdecode(git(repo, "rev-parse", "--show-toplevel").strip())).resolve()
        require(top == repo, f"Expected repository root: {repo}")
    original_common = Path(os.fsdecode(git(
        original, "rev-parse", "--path-format=absolute", "--git-common-dir"
    ).strip())).resolve()
    clone_git = clone / ".git"
    require(clone_git.is_dir() and not clone_git.is_symlink(), "Clone needs its own .git directory")
    for name in ("--absolute-git-dir", "--git-common-dir"):
        actual = Path(os.fsdecode(git(
            clone, "rev-parse", "--path-format=absolute", name
        ).strip())).resolve()
        require(actual == clone_git.resolve(), "Clone has a shared or redirected Git directory")
        require(actual != original_common, "Original and clone share their Git directory")
    require(worktrees(clone) == {clone}, "Clone has additional registered worktrees")
    for name in ("alternates", "http-alternates"):
        require(not (clone_git / "objects/info" / name).exists(), "Clone uses object alternates")
    require(not git(clone, "config", "--get-regexp", r"^(extensions\.partialclone|remote\..*\.promisor)$", check=False).stdout,
            "Partial/promisor clone refused")
    require(not git(clone, "symbolic-ref", "-q", "HEAD", check=False).stdout, "Expected detached HEAD")
    require(git(clone, "rev-parse", "HEAD").strip() == SOURCE.encode(), "Clone HEAD differs from fixed source")
    require(not git(clone, "for-each-ref"), "Detached clone must have no additional refs")
    require(git(clone, "remote").strip() == b"origin", "Expected only the original clone remote")
    origin = os.fsdecode(git(clone, "remote", "get-url", "origin").strip())
    require(Path(origin).resolve() == original, "Clone origin must be the explicit local original")
    require(not git(clone, "status", "--porcelain=v1", "--untracked-files=all"), "Clone is dirty")
    require(not (clone_git / "filter-repo").exists(), "Clone was previously filtered")
    require(not (clone_git / "objects").is_symlink(), "Clone object directory is a symlink")
    for path in (clone_git / "objects").rglob("*"):
        require(not path.is_symlink(), "Clone object storage contains a symlink")
        if path.is_file():
            other = original_common / "objects" / path.relative_to(clone_git / "objects")
            require(not other.exists() or not path.samefile(other), "Clone has hard-linked original objects; use --no-local")
    require(not any((clone_git / "objects").rglob("*.promisor")), "Promisor object pack refused")


def ancestry_sets(original, history):
    sets = {"H": set(git(original, "rev-list", SOURCE).splitlines())}
    for name, boundary in history["boundaries"].items():
        revision = boundary["commit"].encode()
        sets["D_" + name] = {revision} | set(git(
            original, "rev-list", "--ancestry-path", boundary["commit"] + ".." + SOURCE
        ).splitlines())
        parents = git(original, "show", "-s", "--format=%P", boundary["commit"]).decode().split()
        require(parents == boundary["parents"], f"Boundary parents differ: {name}")
    sets["R"] = sets["D_canonical_archive_move"] | sets["D_sibling_archive_move"]
    for name, expected in history["ancestry_sets"].items():
        actual = sets[name]
        encoded = b"\n".join(sorted(actual)) + b"\n"
        require(len(actual) == expected["count"], f"Ancestry count differs: {name}")
        require(digest(encoded) == expected["sha256_sorted_lf_terminated"], f"Ancestry hash differs: {name}")
        if "members" in expected:
            require(actual == {item.encode() for item in expected["members"]}, f"Ancestry members differ: {name}")
    return sets


def source_history_boundaries(original, clone, history, sets):
    observations = {}
    boundaries = {}
    for name, repo in (("original", original), ("clone", clone)):
        path = Path(os.fsdecode(git(
            repo, "rev-parse", "--path-format=absolute", "--git-path", "shallow"
        ).strip()))
        raw = path.read_bytes() if path.exists() else b""
        entries = raw.splitlines()
        unique = set(entries)
        require(all(len(entry) == 40 and all(byte in b"0123456789abcdef" for byte in entry)
                    for entry in unique), "Malformed shallow boundary")
        shallow = git(repo, "rev-parse", "--is-shallow-repository").strip() == b"true"
        require(shallow == bool(unique), "Shallow flag and boundary file disagree")
        boundaries[name] = unique
        observations[name] = {
            "shallow": shallow,
            "boundary_entry_count": len(entries),
            "unique_boundaries": sorted(entry.decode() for entry in unique),
            "boundary_file_sha256": digest(raw),
        }
    require(boundaries["original"] == boundaries["clone"],
            "Clone shallow boundaries differ from the original source")
    require(set(git(clone, "rev-list", SOURCE).splitlines()) == sets["H"],
            "Clone available ancestry differs from the recorded source ancestry")
    prefixes = ["model2/framework", "model2/umpire", "model/umpire",
                "model2/hosts/go", "model/hosts/go", "model/dev/xtask"]
    proofs = []
    for cutoff in sorted(boundaries["original"]):
        require(cutoff in sets["H"], "Shallow cutoff is outside available source ancestry")
        for name, anchor in history["boundaries"].items():
            revision = anchor["commit"]
            require(revision.encode() in sets["H"], "Missing Rust history anchor: " + name)
            require(cutoff != revision.encode() and git(
                original, "merge-base", "--is-ancestor", cutoff.decode(), revision, check=False
            ).returncode == 0, "Shallow cutoff is not a strict ancestor of Rust history anchor: " + name)
        present = git(original, "ls-tree", "-r", "-z", "--name-only", cutoff.decode(), "--", *prefixes)
        require(not present, "Selected Rust/framework/host/tool prefix exists at shallow cutoff")
        proofs.append({
            "cutoff": cutoff.decode(),
            "strict_ancestor_of_all_recorded_anchors": True,
            "anchor_count": len(history["boundaries"]),
            "inspected_prefixes": prefixes,
            "selected_prefix_entries_at_cutoff": 0,
        })
    return {
        "repositories": observations,
        "matching_unique_boundary_sets": True,
        "available_ancestry_matches_recorded_H": True,
        "cutoff_proofs": proofs,
        "limitation": (
            "Export preserves only the supplied available source history. History before the "
            "recorded shallow cutoffs, including earlier LICENSE history, is unavailable and "
            "is not claimed as preserved. No fetching, unshallowing or shallow-file changes occur."
            if boundaries["original"] else "No shallow source-history boundary was observed."
        ),
    }


class Projection:
    def __init__(self, original, history, sets, aliases):
        self.original = original
        self.history = history
        self.sets = sets
        self.aliases = aliases
        self.alias_count = 0
        self.projected_commits = 0
        self.merge_resets = 0
        self.parents = {}
        for line in git(original, "rev-list", "--parents", SOURCE).splitlines():
            commit, *parents = line.split()
            self.parents[commit] = parents

    @functools.lru_cache(maxsize=8)
    def snapshot(self, revision):
        if revision is None:
            return {}
        require(revision in self.sets["H"], "Projection escaped the fixed source ancestry")
        rules = [(b"model2/framework/", b""), (b"model2/umpire/", b""),
                 (b"model2/hosts/go/", b"hosts/go/")]
        if revision in self.sets["R"]:
            rules.append((b"model/umpire/", b""))
            if revision not in self.sets["D_host_move_inside_framework"]:
                rules.append((b"model/hosts/go/", b"hosts/go/"))
            if revision not in self.sets["D_native_tool_split"]:
                rules.append((b"model/dev/xtask/", b"dev/xtask/"))
        support = {}
        for entry in self.history["support_provenance_allowlist"]:
            qualification = entry["revision_qualification"]
            admitted = (
                qualification.startswith("commit in H")
                or qualification.startswith("commit in R") and revision in self.sets["R"]
                or qualification.startswith("commit in D_rust_birth") and revision in self.sets["D_rust_birth"]
            )
            require(qualification.startswith(("commit in H", "commit in R", "commit in D_rust_birth")),
                    "Unknown support qualification: " + qualification)
            if admitted:
                support[os.fsencode(entry["path"])] = os.fsencode(entry["destination"])
        selected = [prefix.rstrip(b"/") for prefix, _ in rules] + list(support)
        raw = git(self.original, "ls-tree", "-r", "-z", "--full-tree", revision.decode(), "--",
                  *(os.fsdecode(path) for path in selected))
        result = {}
        owners = {}
        for record in raw.split(b"\0"):
            if not record:
                continue
            metadata, path = record.split(b"\t", 1)
            mode, kind, blob = metadata.split()
            destination = support.get(path)
            for prefix, target in rules:
                if path.startswith(prefix):
                    require(destination is None, "Support and framework rules overlap")
                    destination = target + path[len(prefix):]
                    break
            if destination is None:
                continue
            require(kind == b"blob" and mode in (b"100644", b"100755", b"120000"),
                    "Unexpected selected tree entry: " + os.fsdecode(path))
            require(destination and not destination.startswith(b"/") and
                    all(part not in (b"", b".", b"..", b".git") for part in destination.split(b"/")),
                    "Unsafe projected path")
            value = (mode, blob)
            if destination in result:
                require(result[destination] == value, "Conflicting projection: " + os.fsdecode(destination))
                self.alias_count += 1
                self.aliases.write(json.dumps({
                    "commit": revision.decode(), "destination": os.fsdecode(destination),
                    "sources": [os.fsdecode(owners[destination]), os.fsdecode(path)],
                    "mode": mode.decode(), "blob": blob.decode(),
                }) + "\n")
            else:
                result[destination] = value
                owners[destination] = path
        for path in result:
            parts = path.split(b"/")
            require(not any(b"/".join(parts[:index]) in result for index in range(1, len(parts))),
                    "Projected file/directory collision")
        self.projected_commits += 1
        return result

    def callback(self, module, commit, metadata):
        revision = commit.original_id
        current = self.snapshot(revision)
        parents = self.parents[revision]
        previous = self.snapshot(parents[0] if parents else None)
        if len(parents) > 1:
            if all(self.snapshot(parent) == current for parent in parents):
                changes = []
            else:
                # filter-repo trims merge parents before the callback. A reset
                # stays exact even when its rewritten first parent changes.
                changes = [module.FileChange(b"DELETEALL")]
                changes.extend(module.FileChange(b"M", path, value[1], value[0])
                               for path, value in sorted(current.items()))
                self.merge_resets += 1
        else:
            changes = [module.FileChange(b"D", path) for path in sorted(previous.keys() - current.keys())]
            changes.extend(module.FileChange(b"M", path, value[1], value[0])
                           for path, value in sorted(current.items()) if previous.get(path) != value)
        commit.file_changes = changes


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--original", type=Path, required=True)
    parser.add_argument("--clone", type=Path, required=True)
    parser.add_argument("--history-map", type=Path, required=True)
    parser.add_argument("--source-manifest", type=Path, required=True)
    parser.add_argument("--filter-repo-module", type=Path, required=True)
    parser.add_argument("--report-dir", type=Path, required=True)
    args = parser.parse_args()
    original, clone = args.original.resolve(), args.clone.resolve()
    report = args.report_dir.resolve()
    require(not report.is_relative_to(clone), "Reports must remain outside the filtered clone")
    require(not report.exists(), "Report directory must be new")
    guard_repositories(original, clone)
    before = source_identity(original)
    history_raw = args.history_map.read_bytes()
    history = json.loads(history_raw)
    manifest_raw = args.source_manifest.read_bytes()
    manifest = json.loads(manifest_raw)
    require(history["source_commit"] == manifest["source_commit"] == SOURCE, "Unexpected source checkpoint")
    require(digest(manifest_raw) == history["source_manifest_sha256"], "Manifest binding differs")
    require(history["selected_file_count_including_license"] == len(manifest["records"]) == 1189,
            "Selected source inventory differs")
    sets = ancestry_sets(original, history)
    history_boundary = source_history_boundaries(original, clone, history, sets)
    module_path = args.filter_repo_module.resolve()
    require(not module_path.is_relative_to(clone), "Filter module must remain outside the clone")
    spec = importlib.util.spec_from_file_location("git_filter_repo", module_path)
    require(spec is not None and spec.loader is not None, "Cannot load git-filter-repo")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    sys.dont_write_bytecode = True
    spec.loader.exec_module(module)
    report.mkdir(parents=True)
    with (report / "identical-aliases.jsonl").open("x") as aliases:
        projection = Projection(original, history, sets, aliases)
        active = {path: value for path, value in projection.snapshot(SOURCE.encode()).items()
                  if not path.startswith(b".history-support/")}
        expected = {os.fsencode(record["destination"]): (record["mode"].encode(), record["blob"].encode())
                    for record in manifest["records"]}
        require(active == expected, "Current active projection differs from the source inventory")
        flags = ["--refs", "HEAD", "--prune-empty", "always", "--prune-degenerate", "auto",
                 "--preserve-commit-hashes", "--preserve-commit-encoding", "--replace-refs", "delete-no-add", "--no-gc"]
        write_json(report / "preflight.json", {
            "source_commit": SOURCE, "original": str(original), "clone": str(clone),
            "source_identity": before, "history_map_sha256": digest(history_raw),
            "source_manifest_sha256": digest(manifest_raw), "filter_module_sha256": digest(module_path.read_bytes()),
            "export_script_sha256": digest(Path(__file__).read_bytes()),
            "filter_arguments": flags, "selected_files": len(active),
            "ancestry_counts": {name: len(value) for name, value in sets.items()},
            "source_history_boundary": history_boundary,
            "pruning": "Prune empty projected single-parent commits and degenerate no-op merges; reset meaningful merges to their exact projected tree",
            "partial_semantics": "--refs HEAD implies --partial: no other refs exist, origin remains local, automatic GC/reflog expiration is disabled; original unreachable monorepo objects remain until separately authorized clone-only cleanup after independent proof",
            "support_disposition": "Retained separately under .history-support; no layout adaptation performed",
        })
        previous_cwd = Path.cwd()
        try:
            os.chdir(clone)
            options = module.FilteringOptions.parse_args(flags)
            module.RepoFilter(options, commit_callback=functools.partial(projection.callback, module)).run()
        finally:
            os.chdir(previous_cwd)
        require(source_identity(original) == before, "Original refs/HEAD/worktree registration changed during export")
        commit_map = clone / ".git/filter-repo/commit-map"
        require(commit_map.is_file(), "git-filter-repo did not produce a commit map")
        with (report / "commit-map").open("xb") as output:
            output.write(commit_map.read_bytes())
        mappings = dict(line.split() for line in commit_map.read_bytes().splitlines()[1:] if line.strip())
        mapped_source = mappings.get(SOURCE.encode())
        require(mapped_source is not None, "Source checkpoint has no export map row")
        export_head = git(clone, "rev-parse", "HEAD").strip()
        require(mapped_source == b"0" * 40 or export_head == mapped_source, "Detached HEAD did not follow the exported checkpoint")
        retained_values = {value for value in mappings.values() if value != b"0" * 40}
        export_graph = {}
        for row in git(clone, "rev-list", "--parents", "HEAD").splitlines():
            revision, *parents = row.split()
            export_graph[revision] = parents
        require(set(export_graph) == retained_values, "Export graph differs from nonzero commit-map values")
        ancestors = {}
        def export_ancestors(revision):
            if revision not in ancestors:
                seen, pending = set(), [revision]
                while pending:
                    current = pending.pop()
                    if current not in seen:
                        seen.add(current)
                        pending.extend(export_graph[current])
                ancestors[revision] = seen
            return ancestors[revision]
        frontiers = {}
        for row in git(original, "rev-list", "--reverse", "--topo-order", "--parents", SOURCE).splitlines():
            revision, *parents = row.split()
            mapped = mappings[revision]
            candidates = list(dict.fromkeys(item for parent in parents for item in frontiers[parent]))
            if mapped != b"0" * 40:
                frontiers[revision] = [mapped]
            else:
                frontiers[revision] = [candidate for candidate in candidates if not any(
                    candidate != other and candidate in export_ancestors(other) for other in candidates
                )]
        require(frontiers[SOURCE.encode()] == [export_head], "Export HEAD differs from unique ordered retained source frontier")
        exported_tree = {}
        for record in git(clone, "ls-tree", "-r", "-z", "HEAD").split(b"\0"):
            if record:
                metadata, path = record.split(b"\t", 1)
                mode, kind, blob = metadata.split()
                require(kind == b"blob", "Export contains a non-blob tree entry")
                exported_tree[path] = (mode, blob)
        require(exported_tree == projection.snapshot(SOURCE.encode()), "Exported HEAD tree differs from its complete projection")
        require(not git(clone, "status", "--porcelain=v1", "--untracked-files=all"),
                "Exported HEAD, index and working tree are not synchronized")
        summary = {
            "source_commit": SOURCE, "export_head": export_head.decode(),
            "commit_map_sha256": digest(commit_map.read_bytes()),
            "identical_alias_observations": projection.alias_count,
            "projected_snapshot_computations": projection.projected_commits,
            "merge_tree_resets": projection.merge_resets,
            "snapshot_cache": projection.snapshot.cache_info()._asdict(),
            "source_identity_unchanged": True,
            "source_tip_projection_neutral_pruned": mapped_source == b"0" * 40,
            "export_graph_equals_nonzero_mapped_values": True,
            "ordered_source_retained_frontier": [value.decode() for value in frontiers[SOURCE.encode()]],
            "head_tree_matches_complete_projection": True,
            "head_index_worktree_synchronized": True,
            "source_history_boundary": history_boundary,
            "pending": ["Independent post-filter tree, ancestry and metadata verification",
                        "Clone-only origin removal, reflog expiration and unreachable-object cleanup after retained proof",
                        "Separate removal of historical support during portable-layout adaptation",
                        "All deferred build/test/guest/consumer proofs"],
        }
        write_json(report / "export-result.json", summary)
        print(json.dumps(summary))


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, OSError, ValueError, subprocess.SubprocessError) as error:
        print(f"Export refused or failed: {error}", file=sys.stderr)
        sys.exit(1)
