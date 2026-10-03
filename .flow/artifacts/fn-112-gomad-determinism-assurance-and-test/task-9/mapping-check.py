#!/usr/bin/env python3
"""fn-112.9 behavior mapping check.

Usage: mapping-check.py BEFORE.json AFTER.json [BEFORE.json AFTER.json ...] > mapping.tsv

Each pair is `go test -json` output of one package before and after the consolidation. A
behavior is one recorded test or subtest name. Every name recorded before and absent after
must map to a replacement name recorded after, or to a removal reason. The script prints one
row per removed or renamed behavior (old name, old status, replacement, new status, note),
then the retained rows count, and exits 1 when a removed behavior has no mapping, when a
replacement is missing after, or when a replacement's status is worse than the old status
(pass > skip > fail) without being a known harness-only failure listed in HARNESS_FAILS.
"""
import json
import re
import sys

# Removed architecture checks: each pins a past refactor rather than a defect.
ARCHITECTURE_REASONS = {
    "TestCleanupRemovesSupersededFiles": "deleted-file list: pins the fn-102..fn-105 cleanup; a recreated script or directory is not a defect by itself, and any Go package an importer reaches must still pass TestPackageArchitecture's owner and import rules",
    "TestCurrentVocabularyHasNoLegacyCampaignBoundary": "banned words: pins a naming migration (batch/run -> campaign); wording is a review concern, not a behavior, and no test or gate consumes the vocabulary",
    "TestCanonicalJSONHasOnePrivateOwner": "required/deleted filenames: TestPackageArchitecture requires the canonicaljson owner to have a package, an importer of a recreated evidence package fails as an ownerless import, and a duplicate private helper in world/codec.go is a review concern",
    "TestRecordAndArtifactHaveSeparateOwners": "required/deleted filenames: TestPackageArchitecture requires record and artifact owner packages and forbids artifact/record importing runner (moduleMayImport); the file names and the removed evidence package pin the fn-105 split",
    "TestReadOnlyMountHasOneDeepOwner": "required/deleted filenames: readonlymount is imported by the runner build, so a missing package fails compilation; the absence of deterministicio/readonly_*.go pins a file layout",
    "TestDeveloperToolingIsNotOwnedByToolchain": "required/deleted filenames: TestPackageArchitecture requires the developer owner, make validate runs cmd/gomadtool and its generators, and the toolchain owner may import only canonicaljson, hostexec, hostfs; the absent toolchain/cmd paths pin a move",
    "TestWorldProcessSessionHasOneDeepOwner": "required/deleted filenames: world/process is compiled into the runner, and the absent world/host, world/target, world/internal/transport pin a past layout",
    "TestCompatibilityPackHasOneOwner": "required/deleted filenames: TestPackageArchitecture requires the compatibility owner and cmd/gomadtool compiles authoring; the absent target/internal/compatibility and target/packdev pin a move",
    "TestUpgradeOrchestrationIsAboveToolchain": "required/deleted filenames: TestExactModuleEdges keeps toolchain from importing qualification or upgrade and upgrade depending on qualification/set and toolchain/version; the file names pin a move",
    "TestQualificationUseCasesHaveExplicitOwners": "required/deleted filenames: TestExactModuleEdges requires upgrade to import qualification/set and the CLI compiles the other use cases; the absent suite_legacy.go/suite_previous.go pin a codec removal",
    "TestConformanceRuntimeIsGroupedByBehavior": "required/deleted filenames: pins the fn-112.6 file split of the conformance runtime; the runtime tests themselves (internal/gomadtool/conformance) cover its behavior",
}

DRIFT_CASES = {
    "upstream-file": "upstream-file", "missing-anchor": "missing-anchor", "ambiguous-anchor": "ambiguous-anchor",
    "replacement-digest": "replacement-digest", "upstream-edit": "upstream-file", "replacement-edit": "replacement-digest",
    "source": "upstream-file", "anchor": "missing-anchor", "ambiguous": "ambiguous-anchor", "replacement": "replacement-digest",
}

ADAPTERS = {
    "Sprig": "sprig", "Validator": "validator", "Sentry": "sentry", "Pebble": "pebble",
    "CactusStatsD": "cactusstatsd", "Memberlist": "memberlist", "HashicorpMetrics": "hashicorp-metrics",
}

# Failing before and after on linux/arm64 only because no adapter is pinned for the host.
HARNESS_FAILS = set()

DRIFT = "TestRewrittenModuleRewritesPreserveCommentsAndRejectDrift"
GRPC_PROFILE = {
    "TestProfileRejectsUnsupportedGRPCVersion": "version",
    "TestProfileRejectsExistingGRPCReplacement": "replacement",
    "TestProfileRejectsExistingGRPCReplacementBlock": "replacement-block",
    "TestProfileRejectsChangedGRPCModuleSum": "sum",
    "TestProfileRejectsBuildModFileWithGRPCAdapter": "build-modfile",
}


def load(path):
    results = {}
    for line in open(path):
        try:
            event = json.loads(line)
        except ValueError:
            continue
        if event.get("Test") and event.get("Action") in ("pass", "fail", "skip"):
            results[event["Test"]] = event["Action"]
    return results


def rewrite_paths(after, adapter):
    prefix = DRIFT + "/" + adapter + "/"
    return sorted({name[len(prefix):].rsplit("/", 1)[0] for name in after if name.startswith(prefix) and name.endswith("/comments")})


def replacements(name, after):
    """Returns (replacement names, note) for a removed behavior."""
    top, _, sub = name.partition("/")
    if top in ARCHITECTURE_REASONS:
        return [], "removed: " + ARCHITECTURE_REASONS[top]
    match = re.fullmatch(r"Test(Sprig|Validator|Sentry|Pebble|CactusStatsD|Memberlist|HashicorpMetrics)(.*)", top)
    if match:
        adapter, rest = ADAPTERS[match.group(1)], match.group(2)
        paths = rewrite_paths(after, adapter)
        if rest == "AdapterConsumer":
            return ["TestRewrittenModuleConsumers/" + adapter], "consumer template row"
        if rest == "RejectsChangedIdentity":
            return ["TestRewrittenModulesRejectChangedIdentity"], "row " + adapter + ": the same three changed identities (module, version, sum), same 'identity mismatch' check"
        if rest == "RejectsUnrewrittenInventoryDrift":
            return ["TestRewrittenModulesRejectUnrewrittenInventoryDrift/" + adapter], "same unrewritten file changed"
        if rest in ("RewritePreservesComments",):
            return [DRIFT + "/" + adapter + "/" + path + "/comments" for path in paths], "comments row per rewritten file"
        if rest == "RewritePreservesCommentsAndMemFS":
            return [DRIFT + "/" + adapter + "/" + path + "/comments" for path in paths] + ["TestRewrittenModuleInventoriesMatchPinnedModules"], "comments row per rewritten file; the MemFS byte-equality check is the table's unrewritten-file copy check (vfs/mem_fs.go)"
        if rest in ("RewriteRejectsDrift", "RewritePreservesCommentsAndRejectsDrift"):
            parts = sub.split("/") if sub else []
            if not parts:
                cases = ["comments"] if "Comments" in rest else []
                return [DRIFT + "/" + adapter + "/" + path + "/" + case for path in paths for case in cases + list(dict.fromkeys(DRIFT_CASES.values()))], "every drift case of every rewritten file"
            case = parts[-1]
            if case in DRIFT_CASES:
                path = "/".join(parts[:-1])
                targets = [path] if path else paths
                return [DRIFT + "/" + adapter + "/" + p + "/" + DRIFT_CASES[case] for p in targets], "drift case " + DRIFT_CASES[case]
            path = "/".join(parts)
            return [DRIFT + "/" + adapter + "/" + path + "/" + c for c in ["comments"] + list(dict.fromkeys(DRIFT_CASES.values()))], "rewritten file: comments and every drift case"
    for old, adapter in (("TestProfileRejectsHashicorpMetricsConfigurationDrift", "hashicorp-metrics"), ("TestProfileRejectsSentryConfigurationDrift", "sentry")):
        if top == old:
            prefix = "TestProfileRejectsAdapterConfigurationDrift/" + adapter
            if sub:
                return [prefix + "/" + sub], "configuration-drift row"
            return sorted(name for name in after if name.startswith(prefix + "/")), "configuration-drift rows of the adapter"
    if top in GRPC_PROFILE:
        return ["TestProfileRejectsAdapterConfigurationDrift/grpc/" + GRPC_PROFILE[top]], "configuration-drift row (same go.mod/go.sum shape, want text, and invalid-configuration check)"
    if top == "TestFilesystemManifestInterceptsProfileOperationsBeforeHostDispatch":
        return ["TestManifestInterceptsBeforeHostDispatch/filesystem_profile_operations"], "same four os functions and gomadIOEnabled guard"
    if top == "TestNetworkManifestInterceptsConcreteTCPMethodsBeforeHostDispatch":
        return ["TestManifestInterceptsBeforeHostDispatch/concrete_TCP_methods"], "same six TCPConn methods and guards"
    if top in ("TestPinnedGRPCModuleInventory", "TestPinnedXNetModuleInventory", "TestPinnedModerncMemoryModuleInventory"):
        return ["TestPinnedAdapterModuleInventories"], "inventory row with the same module directory and expected digest"
    return None, ""


RANK = {"pass": 2, "skip": 1, "fail": 0}


def main(arguments):
    failures = 0
    print("package\told\told_status\treplacement\tnew_status\tnote")
    for index in range(0, len(arguments), 2):
        before, after = load(arguments[index]), load(arguments[index + 1])
        package = arguments[index].rsplit("/", 1)[-1].replace("before-", "").replace(".json", "")
        retained = 0
        for name in sorted(before):
            if name in after:
                retained += 1
                if RANK[after[name]] < RANK[before[name]]:
                    print(f"{package}\t{name}\t{before[name]}\t(retained)\t{after[name]}\tSTATUS WORSE")
                    failures += 1
                continue
            targets, note = replacements(name, after)
            if targets is None:
                print(f"{package}\t{name}\t{before[name]}\t-\t-\tUNMAPPED")
                failures += 1
                continue
            if not targets:
                print(f"{package}\t{name}\t{before[name]}\t-\t-\t{note}")
                continue
            for target in targets:
                status = after.get(target, "MISSING")
                flag = ""
                if status == "MISSING":
                    flag, failures = " MISSING", failures + 1
                elif RANK[status] < RANK[before[name]] and target not in HARNESS_FAILS:
                    flag, failures = " STATUS WORSE", failures + 1
                print(f"{package}\t{name}\t{before[name]}\t{target}\t{status}\t{note}{flag}")
        added = sorted(set(after) - set(before))
        print(f"# {package}: {len(before)} behaviors before, {len(after)} after, {retained} retained by name, {len(added)} new names")
    print(f"# problems: {failures}")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
