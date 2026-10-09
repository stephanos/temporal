import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
import {repo, proxyRoot, hash} from './run.mjs';

const prior = JSON.parse(fs.readFileSync(path.join(repo, '.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/fixture-qualification-evidence.json'))).adapter;
const sums = new Map([
  ['v3.2.3', 'h1:eL2fZNezLomi0uOLqjQoN6BfsDD+fyLtgbJMAj9n6YA='],
  ['v3.3.0', 'h1:mQh0Yrg1XPo6vjYXgtf5OtijNAKJRNcTdOOGZe3tPhs='],
]);
if (prior.module !== 'github.com/Masterminds/sprig/v3' || prior.version !== 'v3.2.3' || prior.sum !== sums.get(prior.version)) throw Error('retained previous checksum differs');
if (!fs.readFileSync(path.join(repo, 'go.sum'), 'utf8').includes('github.com/Masterminds/sprig/v3 v3.3.0 ' + sums.get('v3.3.0') + '\n')) throw Error('current pinned checksum differs');
const unzip = args => {
  const result = spawnSync('/usr/bin/unzip', args, {maxBuffer: 16 * 1024 * 1024});
  if (result.status !== 0) throw Error(result.stderr.toString());
  return result.stdout;
};
for (const [version, expected] of sums) {
  const prefix = path.join(proxyRoot, 'github.com/!masterminds/sprig/v3/@v', version);
  const info = JSON.parse(fs.readFileSync(prefix + '.info'));
  if (info.Version !== version) throw Error('proxy version metadata differs');
  const entries = unzip(['-Z1', prefix + '.zip']).toString().trimEnd().split('\n').sort();
  const digest = crypto.createHash('sha256');
  for (const entry of entries) {
    if (!entry.startsWith('github.com/Masterminds/sprig/v3@' + version + '/')) throw Error('archive entry identity differs');
    digest.update(hash(unzip(['-p', prefix + '.zip', entry])) + '  ' + entry + '\n');
  }
  const actual = 'h1:' + digest.digest('base64');
  if (actual !== expected || fs.readFileSync(prefix + '.ziphash', 'utf8').trim() !== expected) throw Error('module archive checksum differs: ' + version);
  if (!unzip(['-p', prefix + '.zip', 'github.com/Masterminds/sprig/v3@' + version + '/go.mod']).equals(fs.readFileSync(prefix + '.mod'))) throw Error('proxy mod bytes differ from archive');
  console.log(JSON.stringify({module: 'github.com/Masterminds/sprig/v3', version, expected, actual, zip_sha256: hash(fs.readFileSync(prefix + '.zip')), archive_and_proxy_mod_identical: true, files: entries.length, source: version === 'v3.2.3' ? 'retained fn113.2 fixture checksum only' : 'current root go.sum'}));
}
console.log(JSON.stringify({GOPROXY: 'file://' + proxyRoot, network: false, archive_mutation: false}));
