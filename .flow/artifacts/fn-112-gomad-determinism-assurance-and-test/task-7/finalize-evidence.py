from pathlib import Path
import hashlib,json,shlex,subprocess
root=Path('/Users/stephan/Workspace/temporal/gomad')
a=root/'.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-7'
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
key=(root/'tools/gomad3/.toolchain/build-key').read_text().strip()
expected='56e4a2f0c5514d43b9a0682d030964588dd3d5a58978ba0b989459bb556843a3'
assert key==expected
frozen=json.loads((a/'frozen-source-hashes.json').read_text())
live={f:sha(root/f) for f in frozen}
assert live==frozen
records=[json.loads(line) for line in (a/'commands.jsonl').read_text().splitlines()]
bylabel={r['label']:r for r in records}
parent_gate=next(r['label'] for r in reversed(records) if r['label'].startswith('parent-') and r['label'].endswith('final3') and 'overlay-test' in r['command'] and 'test-runtime' in r['command'])
passing=['generate-validate-final3','toolchain-rebuild-final3','focused-backend-libc-final3','focused-os-final3','focused-os-unseeded-final3','focused-os-stock-final3','focused-model-final3','test-host-final3','format-final3','focused-vet-final3','focused-fixture-vet-final3',parent_gate]
for label in passing:assert bylabel[label]['exit_code']==0,label
for label in ['root-lint-final3','root-lint-head-final3']:assert bylabel[label]['exit_code']==2
assert (a/'format-final3.log').stat().st_size==0
(a/'final-source-hashes.json').write_text(json.dumps(live,indent=2)+'\n')
(a/'final-source-validation.json').write_text(json.dumps({'frozen_sources_match':True,'files':len(live),'build_key':key},indent=2)+'\n')
(a/'final-status.txt').write_bytes(subprocess.check_output(['git','status','--short'],cwd=root))
(a/'disk-final.txt').write_bytes(subprocess.check_output(['df','-k',str(root)],cwd=root))
summary=(a/'handover-summary.md').read_text()
summary=summary.replace('Full test-host and overlay-test pass; the runtime final3 gate must finish before the final evidence receipt.', 'Full test-host passes; complete overlay-test and test-runtime also pass on this final identity (the conductor executed '+parent_gate+'; exact argv, exit and raw output are retained).')
summary=summary.replace('Final3 verification completed so far:', 'Final3 verification passes:')
summary=summary.replace('handover-evidence.json and evidence-bindings.json will bind completed final gates to this source/build identity.','handover-evidence.json and evidence-bindings.json bind completed final gates to this source/build identity.')
(a/'handover-summary.md').write_text(summary)
files={str(p.relative_to(a)):sha(p) for p in sorted(a.rglob('*')) if p.is_file() and p.name not in ['evidence-bindings.json','handover-evidence.json']}
(a/'evidence-bindings.json').write_text(json.dumps({'toolchain_build_key':key,'frozen_source_manifest_sha256':sha(a/'frozen-source-hashes.json'),'artifacts_sha256':files},indent=2)+'\n')
evidence={
 'task':'fn-112-gomad-determinism-assurance-and-test.7','commits':[],'prs':[],
 'tests':[shlex.join(bylabel[label]['command'])+' (cwd '+bylabel[label]['cwd']+')' for label in passing],
 'test_records':[bylabel[label] for label in passing],
 'lint_limits':[bylabel[label] for label in ['root-lint-final3','root-lint-head-final3']],
 'toolchain_build_key':key,'retained_baseline_key':'6b775117cc6b13d04c2d00926818e102edb540f8c74ece2794b5e6f3cd2c19ee','retained_intermediate_keys':['d15ad896e542b9f64381c2f184567355bac8ffd62ae2347501fe8fcdd96f6eba','d49ef0309636e2301c46cfb36cda8a9b21323d87601c93c429260b3eefd4e37d'],
 'platform':'darwin/arm64','linux_status':'unverified; no native Linux host available',
 'frozen_sources_match':True,'frozen_source_files':len(live),'changed_files':json.loads((a/'changed-source-files.json').read_text()),
 'task_only_patch_sha256':sha(a/'task-only.patch'),'before_copy_manifest_sha256':sha(a/'beforecopies/manifest.json'),'before_task_spec_sha256':sha(a/'before-task-spec.md'),
 'final_source_manifest_sha256':sha(a/'final-source-hashes.json'),'evidence_bindings_sha256':sha(a/'evidence-bindings.json'),
 'summary_sha256':sha(a/'handover-summary.md'),
 'declared_differences':'modelDeclaredDifferences in tools/gomad3/runner/internal/execution/model_conformance_test.go; only exact successful stat-dir workspace/dir allocation size on darwin/arm64 and linux/amd64',
 'comparisons':{'repetitions':3,'fixtures':2,'seeds':[0,1,7,42,89],'full_length':64,'complete_comparisons':30,'operations_per_side':1920,'prefix_checks':'16 representative shorter successful prefixes per fixture/seed/repetition','failure_search':'ascending fresh-process prefixes; tested nonmonotonic control'},
 'model_findings':'findings.md; original public file/network sentinel loss and self-rename deletion corrected; captured libc errno regression and closed-directory wrapper/sentinel mismatch corrected; no semantic defect normalized',
 'disk_exception':'Second rebuild began with 7928832 KiB and third with 6618044 KiB available (>6 GiB approved measured threshold); first build peak additional use about1.5 GiB; baseline/intermediate/active builds and shared caches preserved',
 'parent_proofs':{name:sha(a/name) for name in ['parent-conformance-validation.json','parent-conformance-final2-validation.json','parent-conformance-final3-validation.json','parent-protected-source-check.json','parent-patch-validation.json','parent-patch-final3-validation.json','parent-built-sources-final3-validation.json']},
 'intermediate_gate_failure':{'record':bylabel['parent-runtime-overlay-final2'],'runtime_tier':'passed','overlay_tier':'failed new directory-read expectations; exact native/seeded probes and approved correction retained','disposition':'final3 corrected and requalified; no skipped tests'},
 'review_status':'ready for conductor independent review; writer did not self-review or mark task done',
 'git_mutations':'none; no staging, commits, pushes, stash, worktrees or reverting others',
}
(a/'handover-evidence.json').write_text(json.dumps(evidence,indent=2)+'\n')
print(json.dumps({'build_key':key,'artifacts_bound':len(files),'frozen_sources':len(live),'passing_gates':len(passing),'patch_sha256':evidence['task_only_patch_sha256'],'receipt_sha256':sha(a/'handover-evidence.json')},indent=2))
