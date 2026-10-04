"""Run the single current-source stock-host preservation selection."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time

ROOT=Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT=ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/preservation-audit'
GO='/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
names=[
    'TestCloneManifestSharesNoMemory',
    'TestOwnedWorldComposedManifestPreservesIdentity',
    'TestTimestampValidationPreservesGrammarAndParseErrors',
    'TestTimestampFinalizeRetainsOriginalStringsAndIdentities',
    'TestOwnedTerminalPreservesAllCategoriesAndRecordingBytes',
    'TestOwnedModelErrorPreservesMessagesAndCauseIdentity',
    'TestRecorderRejectsExternalCallbacksWithoutMutation',
    'TestRecorderAndModelIgnoreSentinelRebinding',
    'TestDetachedTerminalRejectsInvalidDataAndTypedNilWithoutClosing',
    'TestSessionNormalizesDetailBeforeOrderedClassification',
    'TestSessionRetainsJoinedErrorPrecedenceAndUnknownIdentity',
    'TestSessionValidatesBeforeCallingErrors',
    'TestCompatibilityPackProjectionPreservesCompleteEvidence',
    'TestCompatibilityPackProjectionPreservesNilAndEmpty',
    'TestCompatibilityPackProjectionPreservesOrderAndDetachesNestedStorage',
    'TestCapabilityReviewGoldenCanonicalBytes',
    'TestInspectionCapacityProjectionPreservesReports',
    'TestDiagnosticsOffReportFieldsRemainAbsent',
]
args=[GO,'test','-count=1','-tags','test_dep','-run','^('+'|'.join(names)+')$','-v','./artifact','./record','./world','./world/process','./target','./runner','./qualification']
env=dict(os.environ,GOWORK='off',GOTOOLCHAIN='local',GOFLAGS='')
for k in ('GOMADSEED','GOMAD3_CHILD_SEED'):env.pop(k,None)
inventory=json.loads((OUT/'inventory.json').read_text())
manifest=inventory['source_after']['current']
before_mismatches=[k for k,h in manifest.items() if not (ROOT/'tools/gomad3'/k).exists() or hashlib.sha256((ROOT/'tools/gomad3'/k).read_bytes()).hexdigest()!=h]
assert not before_mismatches,before_mismatches
start=datetime.datetime.now(datetime.timezone.utc).isoformat();tick=time.monotonic()
p=subprocess.run(args,cwd=ROOT/'tools/gomad3',env=env,capture_output=True,text=True,timeout=600)
end=datetime.datetime.now(datetime.timezone.utc).isoformat();elapsed=time.monotonic()-tick
output=p.stdout+p.stderr;(OUT/'focused-preservation.log').write_text(output)
after_mismatches=[k for k,h in manifest.items() if not (ROOT/'tools/gomad3'/k).exists() or hashlib.sha256((ROOT/'tools/gomad3'/k).read_bytes()).hexdigest()!=h]
selected=[line[len('=== RUN   '):] for line in output.splitlines() if line.startswith('=== RUN   ')]
passes=[line for line in output.splitlines() if line.startswith('--- PASS:')]
skips=[line for line in output.splitlines() if '--- SKIP:' in line]
result={'command':args,'cwd':str(ROOT/'tools/gomad3'),'environment_overrides':{'GOWORK':'off','GOTOOLCHAIN':'local','GOFLAGS':'','unset':['GOMADSEED','GOMAD3_CHILD_SEED']},'start':start,'end':end,'elapsed_seconds':elapsed,'exit_code':p.returncode,'expected_top_level_tests':names,'selected_test_names':selected,'top_level_pass_lines':passes,'skip_lines':skips,'log_sha256':hashlib.sha256(output.encode()).hexdigest(),'source_inventory':'inventory.json source_after.current','before_mismatches':before_mismatches,'after_mismatches':after_mismatches,'native_qualification':False,'limits':'Current pinned stock Linux arm64 preservation checks only. No matched first-task baseline execution or full eight-surface canonical comparison; no patched runtime/process execution.'}
(OUT/'focused-preservation-receipt.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps({'exit_code':p.returncode,'elapsed_seconds':elapsed,'top_level_passes':len(passes),'selected_cases':len(selected),'skips':skips,'source_mismatches':after_mismatches},indent=2))
raise SystemExit(p.returncode)
