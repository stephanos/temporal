"""Independently verify an immutable history export using Git objects and task1 evidence."""

import argparse
import datetime
import functools
import hashlib
import json
import os
from pathlib import Path
import stat
import subprocess
import sys


SOURCE = "c88fdc550694020eb1c6a2e09577813d906c9ffe"
PROVENANCE = b".history-support/"


class OutputRefused(ValueError):
    pass


def digest(data):
    return hashlib.sha256(data).hexdigest()


class Repository:
    def __init__(self, root):
        self.root = Path(root).resolve(strict=True)
        self.env = os.environ.copy()
        for key in (
            "GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE",
            "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES",
        ):
            self.env.pop(key, None)
        self.env.update(GIT_OPTIONAL_LOCKS="0", GIT_NO_REPLACE_OBJECTS="1")
        self.batch = None
        self.common = Path(self.git("rev-parse", "--path-format=absolute", "--git-common-dir").decode().strip()).resolve()
        if self.git("rev-parse", "--show-object-format").strip() != b"sha1":
            raise ValueError("this source inventory requires SHA-1 Git object identities")
        self.batch = subprocess.Popen(
            ["git", "--no-optional-locks", "-C", str(self.root), "cat-file", "--batch"],
            env=self.env, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
        )

    def git(self, *args):
        result = subprocess.run(
            ["git", "--no-optional-locks", "-c", "core.fsmonitor=false", "-C", str(self.root), *args],
            env=self.env, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
        )
        if result.returncode:
            raise ValueError(f"git {args[0]} failed: {result.stderr[:1000].decode(errors='replace')}")
        return result.stdout

    def object(self, oid, expected_kind):
        self.batch.stdin.write(oid.encode() + b"\n")
        self.batch.stdin.flush()
        header = self.batch.stdout.readline().split()
        if len(header) != 3 or header[0].decode() != oid or header[1].decode() != expected_kind:
            raise ValueError(f"missing or unexpected {expected_kind} object {oid}: {header!r}")
        size = int(header[2])
        payload = self.batch.stdout.read(size)
        if len(payload) != size or self.batch.stdout.read(1) != b"\n":
            raise ValueError(f"truncated object {oid}")
        if hashlib.sha1(expected_kind.encode() + b" " + str(size).encode() + b"\0" + payload).hexdigest() != oid:
            raise ValueError(f"object payload hash mismatch: {oid}")
        return payload

    @functools.lru_cache(maxsize=16384)
    def children(self, oid):
        data = self.object(oid, "tree")
        entries = {}
        cursor = 0
        while cursor < len(data):
            space = data.index(b" ", cursor)
            nul = data.index(b"\0", space)
            mode = data[cursor:space].decode().zfill(6)
            name = data[space + 1:nul]
            child = data[nul + 1:nul + 21]
            if len(child) != 20 or name in entries:
                raise ValueError(f"malformed or duplicate tree entry in {oid}")
            entries[name] = (mode, child.hex())
            cursor = nul + 21
        return entries

    def lookup(self, tree, path):
        parts = path.split(b"/")
        for index, name in enumerate(parts):
            entry = self.children(tree).get(name)
            if entry is None:
                return None
            mode, oid = entry
            if index == len(parts) - 1:
                return entry
            if mode != "040000":
                return None
            tree = oid

    def leaves(self, tree, prefix=b""):
        for name, (mode, oid) in self.children(tree).items():
            path = prefix + name
            if mode == "040000":
                yield from self.leaves(oid, path + b"/")
            else:
                if mode not in {"100644", "100755", "120000"}:
                    raise ValueError(f"unsupported mode {mode} at {path!r}")
                yield path, (mode, oid)

    def close(self):
        if self.batch:
            self.batch.stdin.close()
            self.batch.stdout.close()
            if self.batch.wait(timeout=10):
                raise ValueError("cat-file reader exited unsuccessfully")


def commit_parts(payload):
    header, message = payload.split(b"\n\n", 1)
    fields = {}
    for line in header.split(b"\n"):
        if not line.startswith(b" "):
            key, value = line.split(b" ", 1)
            fields.setdefault(key, []).append(value)
    tree = fields[b"tree"][0].decode()
    parents = [value.decode() for value in fields.get(b"parent", [])]
    identity = tuple(tuple(fields.get(key, [])) for key in (b"author", b"committer", b"encoding")) + (message,)
    return tree, parents, identity


def tree_digest(entries):
    hasher = hashlib.sha256()
    for path, (mode, oid) in sorted(entries.items()):
        hasher.update(mode.encode() + b" " + oid.encode() + b"\t" + path + b"\0")
    return hasher.hexdigest()


def filesystem_state(root, names):
    hasher = hashlib.sha256()
    for name in sorted(names):
        path = os.path.join(os.fsencode(root), name)
        hasher.update(name + b"\0")
        try:
            info = os.lstat(path)
        except FileNotFoundError:
            hasher.update(b"missing\0")
            continue
        if stat.S_ISLNK(info.st_mode):
            mode, payload = b"120000", os.readlink(path)
        elif stat.S_ISREG(info.st_mode):
            mode = b"100755" if info.st_mode & 0o111 else b"100644"
            with open(path, "rb") as stream:
                payload = stream.read()
        else:
            mode, payload = str(stat.S_IFMT(info.st_mode)).encode(), b"non-file"
        hasher.update(mode + b" " + hashlib.sha256(payload).digest() + b"\0")
    return hasher.hexdigest()


def source_state(repo, report_path):
    index = repo.git("ls-files", "--stage", "-z")
    names = {row.split(b"\t", 1)[1] for row in index.split(b"\0") if row}
    names.update(row for row in repo.git("ls-files", "--others", "--exclude-standard", "-z").split(b"\0") if row)
    try:
        names.discard(os.fsencode(report_path.relative_to(repo.root)))
    except ValueError:
        pass
    return {
        "head": repo.git("rev-parse", "HEAD").decode().strip(),
        "head_tree": repo.git("rev-parse", "HEAD^{tree}").decode().strip(),
        "refs_sha256": digest(repo.git("for-each-ref", "--format=%(refname) %(objectname)")),
        "worktree_registry_sha256": digest(repo.git("worktree", "list", "--porcelain", "-z")),
        "index_sha256": digest(index),
        "tracked_and_nonignored_untracked_paths": len(names),
        "worktree_bytes_modes_symlinks_sha256": filesystem_state(repo.root, names),
    }


def load_map(path):
    mapping = {}
    for line in path.read_bytes().splitlines():
        fields = line.split()
        if not fields or fields == [b"old", b"new"]:
            continue
        if len(fields) != 2 or any(len(v) != 40 or any(c not in b"0123456789abcdef" for c in v) for v in fields):
            raise ValueError("malformed commit-map row")
        old, new = (value.decode() for value in fields)
        if old in mapping:
            raise ValueError(f"duplicate commit-map source: {old}")
        mapping[old] = None if new == "0" * 40 else new
    return mapping


def verify(args, report, failures):
    source = Repository(args.source_repo)
    exported = None
    before = None
    def require(condition, category, detail):
        if not condition:
            failures["count"] += 1
            if len(failures["samples"]) < 50:
                failures["samples"].append({"category": category, "detail": detail})
    try:
        for protected in [source.common]:
            if args.output.is_relative_to(protected):
                raise OutputRefused("output must not be inside source Git metadata")
        before = source_state(source, args.output)
        report["source_state_before"] = before
        if Path(source.git("rev-parse", "--show-toplevel").decode().strip()).resolve() != source.root:
            raise ValueError("source-repo must name the original worktree root")
        if args.root_head_baseline:
            baseline = source.git("rev-parse", args.root_head_baseline + "^{commit}").decode().strip()
            require(before["head"] == baseline, "source-baseline", "source HEAD differs from supplied baseline")
        exported = Repository(args.export_repo)
        if source.common == exported.common or source.root == exported.root:
            raise ValueError("export must have independent source and Git directories")
        if args.output.is_relative_to(exported.common):
            raise OutputRefused("output must not be inside export Git metadata")
        alternates = exported.common / "objects/info/alternates"
        require(not alternates.exists() or not alternates.read_bytes().strip(), "isolation", "export uses an object alternates file")
        manifest_bytes = args.manifest.read_bytes()
        manifest = json.loads(manifest_bytes)
        history = json.loads(args.history_map.read_bytes())
        if manifest["source_commit"] != SOURCE or history["source_commit"] != SOURCE:
            raise ValueError("task1 evidence does not bind the fixed extraction source")
        require(digest(manifest_bytes) == history["source_manifest_sha256"], "manifest", "manifest hash differs from history-map binding")
        records = manifest["records"]
        if len(records) != 1178 or manifest["framework_files"] != 1177:
            raise ValueError("source manifest must contain 1177 framework files and LICENSE")
        original_rows = source.git("rev-list", "--topo-order", "--parents", SOURCE).decode().splitlines()
        graph = {row.split()[0]: row.split()[1:] for row in original_rows}
        order = [row.split()[0] for row in reversed(original_rows)]
        shallow_file = source.common / "shallow"
        shallow_rows = shallow_file.read_text().splitlines() if shallow_file.exists() else []
        shallow_cutoffs = set(shallow_rows)
        if any(len(oid) != 40 or any(char not in "0123456789abcdef" for char in oid) for oid in shallow_cutoffs):
            raise ValueError("malformed original source shallow boundary")
        report["source_history_boundary"] = {
            "is_shallow": source.git("rev-parse", "--is-shallow-repository").strip() == b"true",
            "shallow_file_rows": len(shallow_rows),
            "unique_source_cutoffs": sorted(shallow_cutoffs),
            "cutoffs_in_fixed_ancestry": sorted(shallow_cutoffs & set(graph)),
            "accepted_parent_truncations": [],
        }
        if shallow_cutoffs:
            report["limitations"].append("Source history is shallow. Verification covers exactly the available fixed-source ancestry; unavailable pre-cutoff history, including earlier LICENSE history, is not established.")
        mapping = load_map(args.commit_map)
        require(set(mapping) == set(graph), "commit-map-coverage", {"missing": sorted(set(graph) - set(mapping))[:20], "unexpected": sorted(set(mapping) - set(graph))[:20]})
        if set(mapping) != set(graph):
            raise ValueError("commit map does not cover exactly fixed-source ancestry")
        checkpoint = exported.git("rev-parse", args.checkpoint + "^{commit}").decode().strip()
        require(mapping[SOURCE] == checkpoint, "checkpoint", "supplied checkpoint is not the source commit's mapped export")
        export_rows = exported.git("rev-list", "--parents", checkpoint).decode().splitlines()
        export_graph = {row.split()[0]: row.split()[1:] for row in export_rows}
        flags = {name: set() for name in history["boundaries"]}
        for oid in order:
            for name, boundary in history["boundaries"].items():
                if oid == boundary["commit"] or any(parent in flags[name] for parent in graph[oid]):
                    flags[name].add(oid)
        qualified = flags["canonical_archive_move"] | flags["sibling_archive_move"]
        sets = {"H": set(graph), "R": qualified, **{"D_" + name: members for name, members in flags.items()}}
        for name, expected in history["ancestry_sets"].items():
            actual = sets[name]
            encoded = "".join(oid + "\n" for oid in sorted(actual)).encode()
            require(len(actual) == expected["count"] and digest(encoded) == expected["sha256_sorted_lf_terminated"], "ancestry-qualification", name)
        def allowed(prefix, oid):
            if prefix in {"model2/framework/", "model2/umpire/", "model2/hosts/go/"}:
                return True
            if prefix == "model/umpire/":
                return oid in qualified
            if prefix == "model/hosts/go/":
                return oid in qualified and oid not in flags["host_move_inside_framework"]
            if prefix == "model/dev/xtask/":
                return oid in qualified and oid not in flags["native_tool_split"]
            raise ValueError(f"unsupported qualified projection rule: {prefix}")
        def projection(oid, tree):
            entries = {}
            def admit(path, value):
                if path in entries and entries[path] != value:
                    raise ValueError(f"projection collision at {oid}:{os.fsdecode(path)}")
                entries[path] = value
            for rule in history["projection_rules"]:
                prefix = rule["source_prefix"]
                if not allowed(prefix, oid):
                    continue
                found = source.lookup(tree, prefix.rstrip("/").encode())
                if found:
                    if found[0] != "040000":
                        raise ValueError(f"projection prefix is not a tree: {oid}:{prefix}")
                    for name, value in source.leaves(found[1]):
                        admit(rule["destination_prefix"].encode() + name, value)
            for support in history["support_provenance_allowlist"]:
                qualification = support["revision_qualification"]
                if qualification.startswith("commit in H"):
                    include = True
                elif qualification.startswith("commit in R"):
                    include = oid in qualified
                elif qualification.startswith("commit in D_rust_birth"):
                    include = oid in flags["rust_birth"]
                else:
                    raise ValueError(f"unsupported support qualification: {qualification}")
                if include:
                    found = source.lookup(tree, support["path"].encode())
                    if found:
                        if found[0] not in {"100644", "100755", "120000"}:
                            raise ValueError("support allowlist must select individual blobs")
                        admit(support["destination"].encode(), found)
            return entries
        @functools.lru_cache(maxsize=None)
        def exported_tree_hash(oid):
            tree, _, _ = commit_parts(exported.object(oid, "commit"))
            return tree_digest(dict(exported.leaves(tree)))
        @functools.lru_cache(maxsize=512)
        def export_ancestors(oid):
            seen = set()
            pending = [oid]
            while pending:
                current = pending.pop()
                if current not in seen:
                    seen.add(current)
                    pending.extend(export_graph.get(current, []))
            return seen
        frontiers = {}
        retained = 0
        pruned = 0
        boundary_results = []
        representative_ids = {row.split("\t", 1)[0] for item in history["representative_history"] for row in item["history_records"]}
        required = {value["commit"] for value in history["boundaries"].values()} | representative_ids
        for oid in order:
            tree, raw_parents, identity = commit_parts(source.object(oid, "commit"))
            parents = graph[oid]
            if raw_parents != parents:
                if oid not in shallow_cutoffs or parents or not raw_parents or any(parent in graph for parent in raw_parents):
                    raise ValueError(f"source commit parent mismatch outside an actual shallow cutoff: {oid}")
                report["source_history_boundary"]["accepted_parent_truncations"].append({
                    "commit": oid, "unavailable_raw_parents": raw_parents,
                    "available_graph_parents": parents,
                })
            expected = projection(oid, tree)
            projected_hash = tree_digest(expected)
            mapped = mapping[oid]
            if mapped:
                retained += 1
                require(mapped in export_graph, "reachable-mapped-history", {"old": oid, "new": mapped})
                _, _, new_identity = commit_parts(exported.object(mapped, "commit"))
                require(identity == new_identity, "metadata", {"old": oid, "new": mapped, "fields": "raw author, committer, encoding and complete message"})
                require(projected_hash == exported_tree_hash(mapped), "historical-projection", {"old": oid, "new": mapped})
                frontiers[oid] = {mapped}
                disposition = "retained-and-compared"
            else:
                pruned += 1
                frontier = set().union(*(frontiers[parent] for parent in parents)) if parents else set()
                frontier = {candidate for candidate in frontier if not any(candidate != other and candidate in export_ancestors(other) for other in frontier)}
                expected_hashes = {exported_tree_hash(ancestor) for ancestor in frontier}
                if not frontier:
                    expected_hashes = {tree_digest({})}
                require(expected_hashes == {projected_hash}, "pruned-nonneutral-projection", {"old": oid, "retained_parent_frontier": sorted(frontier)[:10]})
                frontiers[oid] = frontier
                disposition = "zero-pruned; compared with nearest nonredundant retained-parent frontier"
            if oid in required:
                boundary_results.append({"original": oid, "mapped": mapped, "disposition": disposition, "projected_files": len(expected), "projection_sha256": projected_hash})
        report["history"] = {"original_commits": len(graph), "retained_metadata_and_snapshot_comparisons": retained, "zero_pruned_projection_comparisons": pruned, "required_boundaries_and_representatives": boundary_results, "qualification": "Recomputed graph ancestry; every original snapshot independently projected, including pre-R exclusions and host/native-tool cutoffs."}
        report["completed_checks"].append("all mapped historical snapshots, retained metadata, and zero-pruned projection neutrality")
        missing_required = required - set(graph)
        require(not missing_required, "required-history-coverage", sorted(missing_required))
        roles = {item["role"] for item in history["representative_history"]}
        require(roles == {"dsl", "checker", "macro", "fixture", "host", "schema"}, "representative-roles", sorted(roles))
        final_tree, _, _ = commit_parts(exported.object(checkpoint, "commit"))
        final_entries = dict(exported.leaves(final_tree))
        active = {path: value for path, value in final_entries.items() if not path.startswith(PROVENANCE)}
        source_tree, _, _ = commit_parts(source.object(SOURCE, "commit"))
        expected_paths = {record["destination"].encode() for record in records}
        require(len(expected_paths) == len(records), "manifest", "duplicate destination paths")
        require(set(active) == expected_paths, "active-path-inventory", {"missing": [os.fsdecode(p) for p in sorted(expected_paths - set(active))[:20]], "unexpected": [os.fsdecode(p) for p in sorted(set(active) - expected_paths)[:20]]})
        framework = source.lookup(source_tree, b"model/umpire")
        if framework is None or framework[0] != "040000":
            raise ValueError("fixed source framework subtree is missing")
        source_records = dict(source.leaves(framework[1], b"model/umpire/"))
        license_entry = source.lookup(source_tree, b"LICENSE")
        if license_entry is None:
            raise ValueError("fixed source license is missing")
        source_records[b"LICENSE"] = license_entry
        require(set(source_records) == {record["source"].encode() for record in records}, "source-inventory", "manifest source paths differ from fixed source")
        symlinks = 0
        for record in records:
            path = record["destination"].encode()
            original_path = record["source"].encode()
            expected_identity = (record["mode"], record["blob"])
            require(source_records.get(original_path) == expected_identity, "source-record", record["source"])
            payload = source.object(record["blob"], "blob")
            require(len(payload) == record["size"] and digest(payload) == record["sha256"], "source-blob", record["source"])
            if record["mode"] == "120000":
                symlinks += 1
                require(payload.hex() == record.get("symlink_target_hex"), "source-symlink", record["source"])
            if path not in active:
                continue
            require(active[path] == expected_identity, "export-mode-or-blob", record["destination"])
            exported_payload = exported.object(active[path][1], "blob")
            require(exported_payload == payload and len(exported_payload) == record["size"] and digest(exported_payload) == record["sha256"], "export-raw-bytes", record["destination"])
        for key, path in (("framework_tree", "model/umpire"), ("archive_tree", "model/archived")):
            actual = source.git("rev-parse", SOURCE + ":" + path).decode().strip()
            require(actual == manifest[key], "source-tree-identity", key)
        bare = exported.git("rev-parse", "--is-bare-repository").strip() == b"true"
        if not bare:
            if Path(exported.git("rev-parse", "--show-toplevel").decode().strip()).resolve() != exported.root:
                raise ValueError("export-repo must name the exported worktree root")
            require(exported.git("rev-parse", "HEAD").decode().strip() == checkpoint, "export-worktree", "export HEAD differs from verified checkpoint")
            require(not exported.git("diff", "--cached", "--raw", checkpoint, "--"), "export-index", "export index differs from the mapped checkpoint")
            raw = exported.git("ls-files", "--others", "--exclude-standard", "-z")
            untracked = [os.fsdecode(path) for path in raw.split(b"\0") if path and not path.startswith(PROVENANCE)]
            require(not untracked, "export-worktree-extra-files", untracked[:20])
            expected_state = hashlib.sha256()
            for path in sorted(active):
                mode, oid = active[path]
                expected_state.update(path + b"\0" + mode.encode() + b" " + hashlib.sha256(exported.object(oid, "blob")).digest() + b"\0")
            require(filesystem_state(exported.root, set(active)) == expected_state.hexdigest(), "export-worktree-bytes-modes-symlinks", "working files differ from the mapped checkpoint")
        report["checkpoint"] = checkpoint
        report["active_projection"] = {"expected_files": len(records), "actual_files": len(active), "symlinks": symlinks, "excluded_exact_namespace_files": len(final_entries) - len(active), "checked_out_worktree": not bare}
        report["completed_checks"].append("1178-file final object and active path inventory; checked-out files when present")
        report["input_hashes"] = {"manifest_sha256": digest(manifest_bytes), "history_map_sha256": digest(args.history_map.read_bytes()), "commit_map_sha256": digest(args.commit_map.read_bytes())}
    finally:
        if before is not None:
            after = source_state(source, args.output)
            report["source_state_after"] = after
            require(before == after, "source-preservation", "source HEAD/refs/index/worktree registry or tracked/nonignored worktree bytes changed during verification")
            report["completed_checks"].append("before/after original source preservation observation")
        if exported is not None:
            exported.close()
        source.close()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source-repo", type=Path, required=True)
    parser.add_argument("--export-repo", type=Path, required=True)
    parser.add_argument("--checkpoint", required=True)
    parser.add_argument("--commit-map", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    task1 = Path(__file__).resolve().parent.parent / "task1"
    parser.add_argument("--manifest", type=Path, default=task1 / "source-manifest.json")
    parser.add_argument("--history-map", type=Path, default=task1 / "history-map.json")
    parser.add_argument("--root-head-baseline", help="exact original root HEAD captured before extraction, otherwise invocation HEAD is captured")
    args = parser.parse_args()
    args.output = args.output.resolve()
    if ".git" in args.output.parts:
        print("output must not be inside Git metadata", file=sys.stderr)
        return 2
    report = {"schema": 1, "source_commit": SOURCE, "started_at": datetime.datetime.now(datetime.timezone.utc).isoformat(), "completed_checks": [], "limitations": ["This verifies extraction objects/history and source preservation during this invocation, not standalone builds, generated freshness or paired consumer behavior.", "An optional root HEAD baseline detects HEAD movement since that checkpoint; pre-invocation index/worktree preservation needs a separately captured byte/mode baseline.", "Source preservation includes tracked and nonignored untracked root worktree files, refs, index entries and worktree registry; ignored bytes and other worktrees' content are not inventoried.", "Only the exact .history-support/ namespace is excluded from the final active manifest; all of its historical/current projected tree entries are nevertheless compared.", "A bare export has no checkout to inspect; its complete committed active tree is still verified. Historical blob identities/modes are compared in every projected snapshot; current selected blob payloads are rehashed."]}
    failures = {"count": 0, "samples": []}
    try:
        verify(args, report, failures)
    except OutputRefused as error:
        print(str(error), file=sys.stderr)
        return 2
    except (OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError) as error:
        failures["count"] += 1
        failures["samples"].append({"category": "verification-error", "detail": str(error)[:2000]})
    report["failures"] = failures
    report["status"] = "pass" if not failures["count"] else "fail"
    report["finished_at"] = datetime.datetime.now(datetime.timezone.utc).isoformat()
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2, ensure_ascii=True) + "\n")
    print(json.dumps({"status": report["status"], "failure_count": failures["count"], "output": str(args.output)}))
    return 0 if report["status"] == "pass" else 1


if __name__ == "__main__":
    sys.exit(main())
