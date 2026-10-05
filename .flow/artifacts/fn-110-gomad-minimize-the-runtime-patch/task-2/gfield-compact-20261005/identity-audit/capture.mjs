import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import { spawnSync, execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

// Emits a receipt only; the conductor retains stdout under this audit directory.
const root = execFileSync('git', ['rev-parse', '--show-toplevel'], { encoding: 'utf8' }).trim();
const helper = path.join(path.dirname(fileURLToPath(import.meta.url)), 'derive.mjs');
const digest = bytes => 'sha256:' + crypto.createHash('sha256').update(bytes).digest('hex');
const paths = [
  'choice/schema/choicewire.json', 'choice/schema/choicewire.go.tmpl',
  'choice/schema/choicewire_runtime.go.tmpl', 'toolchain/runtime/overlay/src/runtime/gomad.go',
  'toolchain/runtime/go1.27.1.patch', 'choice/trace.go', 'choice/tape.go',
  'runner/runner_test.go', 'runner/diagnostic_identity_test.go',
  'record/identity.go', 'record/record.go', 'internal/canonicaljson/canonical.go',
  'runner/portable_plan.go', 'internal/gomadtool/generation/protocol/protocol.go',
  'runner/testdata/diagnostic-identity-choices.json', 'runner/testdata/diagnostic-identity-plain.json',
  'choice/internal/wire/wire_generated.go',
];
const snapshot = () => Object.fromEntries(paths.map(relative => [relative, digest(fs.readFileSync(path.join(root, 'tools/gomad3', relative)))]));
const before = snapshot();
const argv = [process.execPath, helper, ...process.argv.slice(2)];
const started_at = new Date().toISOString();
const start = process.hrtime.bigint();
const result = spawnSync(argv[0], argv.slice(1), { cwd: root, encoding: 'utf8', maxBuffer: 2 << 20 });
const elapsed_ms = Number(process.hrtime.bigint() - start) / 1e6;
const finished_at = new Date().toISOString();
const after = snapshot();
const stdout = result.stdout ?? '';
const stderr = result.stderr ?? '';
const receipt = {
  argv, cwd: root, started_at, finished_at, elapsed_ms,
  exit_status: result.status, signal: result.signal, error: result.error?.message ?? null,
  node_version: process.version, host: { platform: process.platform, arch: process.arch },
  environment: Object.fromEntries(['LANG', 'LC_ALL', 'TZ', 'GOOS', 'GOARCH', 'GOFLAGS', 'GOTOOLCHAIN', 'GOWORK', 'GOENV', 'CGO_ENABLED'].map(key => [key, process.env[key] ?? null])),
  helper_sha256: digest(fs.readFileSync(helper)),
  capture_sha256: digest(fs.readFileSync(fileURLToPath(import.meta.url))),
  stdout_sha256: digest(stdout), stderr_sha256: digest(stderr), stdout, stderr,
  product_before: before, product_after_changes: Object.fromEntries(paths.filter(relative => before[relative] !== after[relative]).map(relative => [relative, after[relative]])),
  product_before_snapshot_sha256: digest(JSON.stringify(before)),
  product_after_snapshot_sha256: digest(JSON.stringify(after)),
  product_writes: 0, native_darwin_guard_executed: false,
};
process.stdout.write(JSON.stringify(receipt, null, 2) + '\n');
assert.deepEqual(after, before, 'product inputs changed during independent calculation');
process.exitCode = result.status ?? 1;
