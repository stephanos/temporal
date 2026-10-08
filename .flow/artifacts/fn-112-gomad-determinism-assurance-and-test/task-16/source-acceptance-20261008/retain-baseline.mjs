import assert from 'node:assert/strict';
import {existsSync, readFileSync, writeFileSync} from 'node:fs';
import {out, sha, stock} from './capture.mjs';

assert(!existsSync(out + '/baseline-artifact.json'));
const stdout = readFileSync('/tmp/fn11216-baseline-artifact.jsonl');
const stderr = readFileSync('/tmp/fn11216-baseline-artifact.stderr');
const events = stdout.toString().trim().split('\n').map(JSON.parse);
const top = events.filter(e => e.Test && !e.Test.includes('/') && ['pass', 'fail', 'skip'].includes(e.Action));
assert(top.length > 0 && top.every(e => e.Action === 'pass'));
const receipt = {argv: ['timeout', '600s', stock + '/go', '-C', 'tools/gomad3', 'test', '-tags', 'test_dep', '-count=1', '-json', './artifact'], cwd: '/Users/stephan/Workspace/skunkworks/gomad/temporal', environment: 'inherited shell environment was not snapshotted; no inline assignments', exit: 0, exit_observation: 'foreground exec tool returned suite_rc=0; enclosing shell observed status immediately', elapsed_seconds: null, tool_wall_seconds: 0.570100625, package_elapsed_seconds: events.findLast(e => e.Action === 'pass' && !e.Test).Elapsed, started_event: events[0].Time, ended_event: events.at(-1).Time, stdout_sha256: sha(stdout), stderr_sha256: sha(stderr), test_counts: {pass: top.length, fail: 0, skip: 0}, tests: top.map(e => ({package: e.Package, test: e.Test, action: e.Action})), source_binding: 'pre-edit HEAD 4a64eb2cd68cc4040045d9d09bbd8bfc82b81819; no source edits before invocation; full environment/command-duration/source-before-after snapshots unavailable for this first observation'};
writeFileSync(out + '/baseline-artifact.stdout', stdout);
writeFileSync(out + '/baseline-artifact.stderr', stderr);
writeFileSync(out + '/baseline-artifact.json', JSON.stringify(receipt, null, 2) + '\n');
console.log(JSON.stringify({baseline: 'green; directly observed exit0', test_counts: receipt.test_counts, durations: 'package duration and tool wall preserved; command duration not reconstructed'}));
