import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const artifacts = dirname(fileURLToPath(import.meta.url));
const base = '13df4f16f90d49938ea123a29859da62ad2cab9f';
const prefix = 'tools/gomad3/internal/compatibilitypack/';
const candidates = ['schema.go', 'mutation_test.go', 'schema_timezone_test.go'].map(p => prefix + p);
const git = args => {
  const result = spawnSync('git', args, { cwd: root, encoding: 'utf8', maxBuffer: 64 * 1024 * 1024 });
  if (result.status !== 0) throw new Error(result.stderr);
  return result.stdout;
};
const hash = bytes => createHash('sha256').update(bytes).digest('hex');
const save = (name, data) => writeFileSync(resolve(artifacts, name + '.json'), JSON.stringify(data, null, 2) + '\n');
const paths = git(['ls-files', '--', 'tools/gomad3', '.github', 'Makefile', 'go.mod', 'go.sum', '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-42', '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-43']).trim().split('\n');
const toolPaths = Object.keys(JSON.parse(readFileSync(resolve(artifacts, 'baseline-lint.json'))).before.binaries);
const manifest = () => ({ captured_at: new Date().toISOString(), head: git(['rev-parse', 'HEAD']).trim(), files: Object.fromEntries(paths.map(p => [p, hash(readFileSync(resolve(root, p)))])), binaries: Object.fromEntries(toolPaths.map(p => [p, hash(readFileSync(p))])), gci: 'embedded configured analyzer in hashed golangci-lint executable; no standalone executable', status: git(['status', '--short']) });
if (process.argv[2] === 'before' || process.argv[2] === 'final') {
  const label = process.argv[2];
  const data = manifest();
  if (label === 'before') {
    const baseline = JSON.parse(readFileSync(resolve(artifacts, 'baseline-lint.json')));
    if (!Object.entries(baseline.before.files).every(([p, digest]) => data.files[p] === digest)) throw new Error('baseline source/config mismatch');
    if (JSON.stringify(data.binaries) !== JSON.stringify(baseline.before.binaries)) throw new Error('baseline tool mismatch');
  }
  save(label, data);
  console.log(JSON.stringify({ label, files: paths.length, binaries: toolPaths.length }));
} else {
  const before = JSON.parse(readFileSync(resolve(artifacts, 'before.json')));
  const final = JSON.parse(readFileSync(resolve(artifacts, 'final.json')));
  const protectedPaths = paths.filter(p => !candidates.includes(p));
  const generated = protectedPaths.filter(p => /_generated\.|\/compatibilitypack\/(packs|requests|reports)\/|\/generation\.json$|\/version\/version\.json$/.test(p));
  const original = p => git(['show', base + ':' + p]);
  const current = p => readFileSync(resolve(root, p), 'utf8');
  const schema = original(candidates[0]).replace('sources = append(sources, Source{Name: source.Name, SHA256: source.SHA256})', 'sources = append(sources, Source(source))');
  const mutation = original(candidates[1]).replace('goSources[index] = Source{Name: source.Name, SHA256: source.SHA256}', 'goSources[index] = Source(source)').replace('foreignSources[index] = ForeignSource{Kind: source.Kind, Name: source.Name, SHA256: source.SHA256}', 'foreignSources[index] = ForeignSource(source)');
  const timezone = original(candidates[2]).replace('\t"bytes"\n\t"go.temporal.io/server/tools/gomad3/internal/canonicaljson"\n\t"go.temporal.io/server/tools/gomad3/record"\n\t"strings"\n\t"testing"', '\t"bytes"\n\t"strings"\n\t"testing"\n\n\t"go.temporal.io/server/tools/gomad3/internal/canonicaljson"\n\t"go.temporal.io/server/tools/gomad3/record"');
  const changed = paths.filter(p => before.files[p] !== final.files[p]);
  const lintBase = JSON.parse(readFileSync(resolve(artifacts, 'baseline-lint.json')));
  const lintFinal = JSON.parse(readFileSync(resolve(artifacts, 'final-lint.json')));
  const blocks = text => text.split(/(?=^tools\/gomad3\/.*:\d+:\d+: )/m).filter(x => /^tools\/gomad3\//.test(x)).map(x => x.replace(/\n\d+ issues:[\s\S]*$/, '\n'));
  const baseBlocks = blocks(lintBase.stdout), finalBlocks = blocks(lintFinal.stdout);
  const removed = baseBlocks.filter(b => !finalBlocks.includes(b));
  const added = finalBlocks.filter(b => !baseBlocks.includes(b));
  const terminalEvents = label => JSON.parse(readFileSync(resolve(artifacts, label + '.json'))).stdout.split('\n').flatMap(line => {
    try { const e = JSON.parse(line); return e.Test && ['pass', 'fail', 'skip'].includes(e.Action) ? [e.Package + ':' + e.Test + ':' + e.Action] : []; } catch { return []; }
  }).sort();
  const checks = {
    exact_schema_conversion_only: current(candidates[0]) === schema,
    exact_mutation_conversions_only: current(candidates[1]) === mutation,
    exact_import_regrouping_only: current(candidates[2]) === timezone,
    only_three_candidate_files_changed: changed.length === 3 && changed.every(p => candidates.includes(p)),
    protected_bytes_unchanged: protectedPaths.every(p => before.files[p] === final.files[p]),
    generated_pins_unchanged: generated.every(p => before.files[p] === final.files[p]),
    generated_pin_count_is_51: generated.length === 51,
    tools_unchanged: JSON.stringify(before.binaries) === JSON.stringify(final.binaries),
    scoped_lint_exact_four_removed: baseBlocks.length === 7 && finalBlocks.length === 3 && removed.length === 4 && added.length === 0 && removed.filter(b => b.includes('S1016:')).length === 3 && removed.filter(b => b.includes('(gci)')).length === 1,
    residual_blocks_byte_identical: finalBlocks.every(b => baseBlocks.includes(b)),
    package_terminal_events_match_baseline: JSON.stringify(terminalEvents('baseline-packages')) === JSON.stringify(terminalEvents('final-packages')),
    target_control_terminal_events_match_baseline: JSON.stringify(terminalEvents('baseline-target-controls')) === JSON.stringify(terminalEvents('final-target-controls')),
    current_manifest_matches_final: paths.every(p => final.files[p] === hash(readFileSync(resolve(root, p))))
  };
  const output = { captured_at: new Date().toISOString(), source_base: base, admission: before.head, checks, candidate_hashes: Object.fromEntries(candidates.map(p => [p, final.files[p]])), changed, protected_count: protectedPaths.length, generated_or_pin_count: generated.length, generated_or_pin_paths: generated, scoped_lint: { baseline: baseBlocks.length, final: finalBlocks.length, removed, added, residual: finalBlocks }, diffs: Object.fromEntries(candidates.map(p => [p, git(['diff', base, '--', p])])) };
  save('preservation-proof', output);
  console.log(JSON.stringify({ checks, protected_count: protectedPaths.length, generated_or_pin_count: generated.length }));
  if (Object.values(checks).some(value => !value)) process.exitCode = 1;
}
