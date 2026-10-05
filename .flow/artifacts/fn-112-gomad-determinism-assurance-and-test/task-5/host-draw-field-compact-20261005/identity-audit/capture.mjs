import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { execFileSync, spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';

const directory = path.dirname(fileURLToPath(import.meta.url));
const root = execFileSync('git', ['rev-parse', '--show-toplevel'], { encoding: 'utf8' }).trim();
const [label, ...args] = process.argv.slice(2);
assert(/^[a-z0-9-]+$/.test(label), 'receipt label must be a simple name');
const outputs = ['stdout', 'stderr', 'json'].map(extension => path.join(directory, `${label}.${extension}`));
assert(outputs.every(output => !fs.existsSync(output)), 'receipt label already exists');
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const sources = () => Object.fromEntries(execFileSync('git', ['ls-files', '-z', 'tools/gomad3'], { cwd: root, maxBuffer: 16 << 20 }).toString().split('\0').filter(Boolean).map(relative => [relative, hash(fs.readFileSync(path.join(root, relative)))]));
const before = sources();
const argv = [path.join(directory, 'derive.mjs'), ...args];
const start = new Date();
const result = spawnSync(process.execPath, argv, { cwd: root, encoding: null, maxBuffer: 16 << 20 });
const end = new Date();
const after = sources();
const stdout = result.stdout ?? Buffer.alloc(0);
const stderr = result.stderr ?? Buffer.from(String(result.error ?? ''));
fs.writeFileSync(outputs[0], stdout, { flag: 'wx' });
fs.writeFileSync(outputs[1], stderr, { flag: 'wx' });
const receipt = {
  argv: [process.execPath, ...argv], cwd: root, node_version: process.version,
  started_utc: start.toISOString(), ended_utc: end.toISOString(), elapsed_ms: end - start,
  exit_code: result.status, signal: result.signal, error: result.error?.message ?? null,
  stdout_sha256: hash(stdout), stderr_sha256: hash(stderr),
  sources_before_sha256: hash(JSON.stringify(before)), sources_after_sha256: hash(JSON.stringify(after)),
  source_changes: Object.keys(before).filter(relative => before[relative] !== after[relative]),
  source_count: Object.keys(before).length,
  derive_helper_sha256: hash(fs.readFileSync(argv[0])), capture_helper_sha256: hash(fs.readFileSync(fileURLToPath(import.meta.url))),
};
fs.writeFileSync(outputs[2], JSON.stringify(receipt, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify(receipt));
process.exitCode = result.status ?? 1;
