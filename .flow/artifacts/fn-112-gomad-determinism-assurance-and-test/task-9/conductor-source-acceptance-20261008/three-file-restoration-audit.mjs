import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { out, worker, read, sha } from './capture.mjs';

const helper = resolve(out, 'restoration-source-audit.mjs');
let source = read(helper).toString();
const paths = "const paths = ['tools/gomad3/architecture_test.go', 'tools/gomad3/deterministicio/adapter_rewrite_test.go'];";
assert.equal(source.split(paths).length, 2);
source = source.replace(paths, "const paths = ['tools/gomad3/architecture_test.go', 'tools/gomad3/deterministicio/adapter_rewrite_test.go', 'tools/gomad3/runner/completion_test.go'];");
const insertion = 'const userFiles = [';
assert.equal(source.split(insertion).length, 2);
source = source.replace(insertion, `
const completionPath = paths[2];
const oldCompletion = git(['show', base + ':' + completionPath]);
const completion = read(completionPath).toString();
const name = 'TestAssessCompletionProjectsCoverageInOrderAndClassifies';
const beforeCompletion = body(oldCompletion, name), afterCompletion = body(completion, name);
assert.equal(oldCompletion.replace(beforeCompletion, '<completion-function>'), completion.replace(afterCompletion, '<completion-function>'));
const originalCompletion = body(git(['show', '59ca3d17395be5501906586006e2539d33bea28c:' + completionPath]), name);
let expectedCompletion = beforeCompletion;
const reasonField = '\\t\\treason string\\n';
assert.equal(expectedCompletion.split(reasonField).length, 2);
expectedCompletion = expectedCompletion.replace(reasonField, reasonField + '\\t\\tcause  string\\n');
const causeRows = ['malformed semantic coverage', 'malformed semantic coverage before malformed choices', 'malformed choices', 'malformed semantic coverage of a watchdog kill'];
for (const row of causeRows) {
  const marker = 'name: ' + JSON.stringify(row) + ',';
  const oldRows = beforeCompletion.split('\\n').filter(line => line.includes(marker));
  const originalRows = originalCompletion.split('\\n').filter(line => line.includes(marker));
  assert.equal(oldRows.length, 1);
  assert.equal(originalRows.length, 1);
  assert.equal(originalRows[0].replace(/, cause:.*(?=},$)/, ''), oldRows[0]);
  expectedCompletion = expectedCompletion.replace(oldRows[0], originalRows[0]);
}
const nilGuard = 'hostError == nil || hostError.Reason != test.reason || hostError.Err == nil';
assert.equal(expectedCompletion.split(nilGuard).length, 2);
expectedCompletion = expectedCompletion.replace(nilGuard, nilGuard + ' || hostError.Err.Error() != test.cause');
const oldMessage = 't.Fatalf("assessCompletion() error = %v, want %s", hostError, test.reason)';
const newMessage = 't.Fatalf("assessCompletion() error = %v, want %s: %s", hostError, test.reason, test.cause)';
assert.equal(expectedCompletion.split(oldMessage).length, 2);
expectedCompletion = expectedCompletion.replace(oldMessage, newMessage);
assert.equal(afterCompletion, expectedCompletion);
${insertion}`);
source = source.replace("writeFileSync(resolve(out, 'restoration-source-proof.json')", "writeFileSync(resolve(out, 'three-file-restoration-proof.json')");
source = source.replace('existing_populated_cache_assertions_exact_after_inverting_only_admitted_wrapper: true,', 'existing_populated_cache_assertions_exact_after_inverting_only_admitted_wrapper: true, private_completion_delta_exactly_original_four_causes_plus_comparison_and_message: true, private_completion_nil_error_guard_preserved: true, all_off_function_completion_bytes_exact: true,');
source = source.replace('production_changes: [], new_exceptions: [], native: false,', 'production_changes: [], new_exceptions: [], native: false, imported_assertion_helper_sha256: sha(read(helper)), current_audit_sha256: sha(read(resolve(out, "three-file-restoration-audit.mjs"))),');
source = source.replace(/^import .*;\n/gm, '');
new Function('assert', 'spawnSync', 'writeFileSync', 'resolve', 'out', 'worker', 'read', 'sha', 'helper', source)(assert, spawnSync, writeFileSync, resolve, out, worker, read, sha, helper);
