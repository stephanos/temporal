import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import { fileURLToPath } from 'node:url';

const out = path.dirname(fileURLToPath(import.meta.url));
const previous = '.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/host-draw-field-compact-20261005';
const scratch = 'tools/gomad3/.toolchain/fn-110/gfield-compact.gxrPVUi5';
const [label, log, baselineLabel, reportName = label + '-observations.json'] = process.argv.slice(2);
assert(/^[a-z0-9-]+\.json$/.test(reportName));
assert(!fs.existsSync(path.join(out, reportName)), 'report already exists');
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
function events(file) {
  const bytes = fs.readFileSync(file);
  const tests = new Map(), packages = new Map();
  for (const line of bytes.toString().split('\n')) {
    let event; try { event = JSON.parse(line); } catch { continue; }
    if (!event.Action || !event.Package) continue;
    if (!event.Test) { if (['pass', 'fail', 'skip'].includes(event.Action)) packages.set(event.Package, event.Action); continue; }
    const key = event.Package + ':' + event.Test;
    const value = tests.get(key) ?? { errors: [] };
    if (['pass', 'fail', 'skip'].includes(event.Action)) value.result = event.Action;
    if (event.OutputType === 'error') value.errors.push(event.Output);
    if (event.Action === 'output') value.output = (value.output ?? '') + (event.Output ?? '');
    tests.set(key, value);
  }
  return { bytes, tests, packages };
}
const final = events(log);
const baseline = events(path.join(previous, baselineLabel + '.stdout'));
assert(final.tests.size > 0, 'empty selection');
const priorSources = JSON.parse(fs.readFileSync(path.join(previous, 'final-sources.json')));
const freeze = JSON.parse(fs.readFileSync(path.join(scratch, 'freeze.json')));
assert.deepEqual(priorSources, freeze.sources, 'preceding candidate differs from BASE');
const changedVerdicts = [], changedErrors = [], unmatched = [], unfinished = [];
for (const [key, value] of final.tests) {
  if (!value.result) unfinished.push(key);
  const prior = baseline.tests.get(key);
  if (!prior) unmatched.push(key);
  else if (prior.result !== value.result) changedVerdicts.push({ test: key, baseline: prior.result, final: value.result });
  if (value.result === 'fail' && JSON.stringify(prior?.errors) !== JSON.stringify(value.errors)) changedErrors.push(key);
}
assert.equal(unfinished.length, 0);
assert.equal(unmatched.length, 0);
assert.equal(changedVerdicts.length, 0);
assert.equal(changedErrors.length, 0);
const info = { receipt: label + '.json', raw_log: log, raw_log_sha256: hash(final.bytes), prior_receipt: previous + '/' + baselineLabel + '.json', baseline_source_map: previous + '/final-sources.json', baseline_source_matches_all_5076_product_files: true, package_results: Object.fromEntries(final.packages), counts: Object.fromEntries(['pass', 'fail', 'skip'].map(result => [result, [...final.tests.values()].filter(value => value.result === result).length])), unmatched, unfinished, changed_verdicts: changedVerdicts, changed_raw_errors: changedErrors, failures_and_skips: Object.fromEntries([...final.tests].filter(([, value]) => ['fail', 'skip'].includes(value.result))) };
fs.writeFileSync(path.join(out, reportName), JSON.stringify(info, null, 2) + '\n');
console.log(label, 'tests', final.tests.size, 'counts', info.counts, 'packages', final.packages.size, 'same preceding verdicts and raw error outputs');
