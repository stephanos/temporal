import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { out, worker, read, sha } from './capture.mjs';

const base = 'a7d99f4e64990839f81df4acd9bc150b842f8ea6';
const git = argv => {
  const result = spawnSync('git', argv, { encoding: 'utf8' });
  assert.equal(result.status, 0, result.stderr);
  return result.stdout;
};
const body = (source, name) => {
  const start = source.indexOf('func ' + name + '(');
  assert(start >= 0, name);
  const end = source.indexOf('\n}', start) + 3;
  assert(end > start, name);
  return source.slice(start, end);
};
const paths = ['tools/gomad3/architecture_test.go', 'tools/gomad3/deterministicio/adapter_rewrite_test.go'];
assert.deepEqual(git(['diff', '--name-only', base, '--', 'tools/gomad3']).trim().split('\n').sort(), paths);
const oldArchitecture = git(['show', base + ':' + paths[0]]);
const architecture = read(paths[0]).toString();
const oldAlias = body(git(['show', '97f221c7ea^:' + paths[0]]), 'TestPublicPackagesDoNotExportTypeAliases');
const addedAlias = body(architecture, 'TestPublicPackagesDoNotExportForwardingAliases');
assert.equal(addedAlias, oldAlias.replace('TestPublicPackagesDoNotExportTypeAliases', 'TestPublicPackagesDoNotExportForwardingAliases'));
assert.equal(architecture, oldArchitecture + '\n' + addedAlias);
const signatureName = 'TestPublicPackagesDoNotExportTypeAliases';
assert.equal(body(architecture, signatureName), body(oldArchitecture, signatureName));
assert.equal(read(resolve(worker, 'before-architecture_test.go')).toString(), oldArchitecture);

const oldAdapter = git(['show', base + ':' + paths[1]]);
const adapter = read(paths[1]).toString();
const identityName = 'TestRewrittenModulesRejectChangedIdentity';
const before = body(oldAdapter, identityName), after = body(adapter, identityName);
assert.equal(oldAdapter.replace(before, '<identity-function>'), adapter.replace(after, '<identity-function>'));
assert.equal(read(resolve(worker, 'before-adapter_rewrite_test.go')).toString(), oldAdapter);
const prefix = 'func ' + identityName + '(t *testing.T) {\n\tfor _, cache := range []string{"empty", "populated"} {\n\t\tt.Run(cache, func(t *testing.T) {\n\t\t\tmoduleCache := t.TempDir()\n\t\t\tif cache == "populated" {\n\t\t\t\tmoduleCache = pinnedModuleCache(t)\n\t\t\t}\n';
const suffix = '\t\t})\n\t}\n}\n';
assert(after.startsWith(prefix));
assert(after.endsWith(suffix));
let retained = after.slice(prefix.length, -suffix.length).split('\n').map(line => {
  if (!line) return line;
  assert(line.startsWith('\t\t'));
  return line.slice(2);
}).join('\n');
assert.equal(retained.split('if cache == "populated" && adapter.outsideServerGraph {').length, 2);
retained = retained.replace('if cache == "populated" && adapter.outsideServerGraph {', 'if adapter.outsideServerGraph {');
assert.equal(before, 'func ' + identityName + '(t *testing.T) {\n\tmoduleCache := pinnedModuleCache(t)\n' + retained + '}\n');
const userFiles = [
  ['.turbo/plans/gomad3-glossary-update.md', '97868a86c0a263fbea61c336bd9557e4d71cf43ae4390e9a2449e6f7cd815188'],
  ['.turbo/technical-debt.md', 'c219247c01fb305592f0314ec46971cee30f00e5985dbd280c1c9e3aafe60287'],
].map(([path, expected]) => {
  const digest = sha(read(path));
  assert.equal(digest, expected);
  return { path, sha256: digest };
});
const result = {
  verified: true, timestamp: new Date().toISOString(), base, head: git(['rev-parse', 'HEAD']).trim(),
  changed_test_files: paths.map(path => ({ path, sha256: sha(read(path)) })),
  original_alias_guard_exact_except_additive_name: true,
  existing_signature_guard_byte_exact: true,
  existing_populated_cache_assertions_exact_after_inverting_only_admitted_wrapper: true,
  all_off_function_adapter_bytes_exact: true,
  all_existing_architecture_bytes_exact: true,
  user_files: userFiles, production_changes: [], new_exceptions: [], native: false,
};
writeFileSync(resolve(out, 'restoration-source-proof.json'), JSON.stringify(result, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify(result));
