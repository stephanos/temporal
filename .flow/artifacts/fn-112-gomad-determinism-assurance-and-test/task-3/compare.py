#!/usr/bin/env python3
"""Compare two behavioral projections written by project.py.

usage: compare.py BASELINE.json CANDIDATE.json CANDIDATE_BUILD_KEY CANDIDATE_CHOICE_SOURCE_SHA256_HEX OUTPUT.json
Exit status 0 when every behavior field matches and every identity is the
expected new one; 1 otherwise.
"""
import hashlib
import json
import sys


def differences(left, right, path=""):
    if isinstance(left, dict) and isinstance(right, dict):
        found = []
        for key in sorted(set(left) | set(right)):
            found += differences(left.get(key), right.get(key), path + "/" + key)
        return found
    return [] if left == right else [path]


def main():
    baseline = json.load(open(sys.argv[1]))
    candidate = json.load(open(sys.argv[2]))
    build_key, source, output = sys.argv[3], sys.argv[4], sys.argv[5]
    # choice.ImplementationIdentity: domain, NUL, generated source digest, build key.
    implementation = "sha256:" + hashlib.sha256(b"gomad3-choice-implementation-v2\x00" + bytes.fromhex(source) + bytes.fromhex(build_key)).hexdigest()
    report = {"workloads": {}, "candidate_build_key": build_key, "candidate_choice_implementation_sha256": implementation}
    ok = set(baseline) == set(candidate)
    report["same_workload_set"] = ok
    for name in sorted(set(baseline) & set(candidate)):
        old, new = baseline[name], candidate[name]
        behavior = differences(old["behavior"], new["behavior"])
        identity = {key: {"baseline": old["identity"][key], "candidate": new["identity"][key], "changed": old["identity"][key] != new["identity"][key]} for key in sorted(old["identity"])}
        recorded = new["identity"]["toolchain_build_key"] == build_key and new["identity"]["choice_implementation_sha256"] == implementation
        report["workloads"][name] = {
            "behavior_matches": not behavior,
            "behavior_differences": behavior,
            "site_dependent_matches": old["site_dependent"] == new["site_dependent"],
            "identity": identity,
            "new_identities_recorded": recorded,
        }
        ok = ok and not behavior and recorded
    report["match"] = ok
    with open(output, "w") as handle:
        json.dump(report, handle, indent=2, sort_keys=True)
        handle.write("\n")
    for name, entry in report["workloads"].items():
        changed = [key for key, value in entry["identity"].items() if value["changed"]]
        print("%s: behavior=%s site_dependent=%s identities_recorded=%s changed=%s %s" % (name, entry["behavior_matches"], entry["site_dependent_matches"], entry["new_identities_recorded"], ",".join(changed), entry["behavior_differences"]))
    print("match=%s" % ok)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
