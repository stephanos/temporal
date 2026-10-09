import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';

const repo = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const base = '4ce2d847afd8762728f30d153d725fd1c073ecb0';
const file = 'tools/gomad3/target/internal/gocommand/command_test.go';
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const git = (...args) => {
  const result = spawnSync('git', args, {cwd:repo, encoding:'utf8'});
  if (result.status !== 0) throw new Error(result.stderr);
  return result.stdout;
};
const before = git('show', `${base}:${file}`);
const after = fs.readFileSync(path.join(repo, file), 'utf8');
const functions = source => {
  const matches = [...source.matchAll(/^func (?:\([^)]*\) )?(\w+).*$/gm)];
  return new Map(matches.map((match, index) => [match[1], source.slice(match.index, matches[index+1]?.index ?? source.length).trim()]));
};
const original = functions(before);
const current = functions(after);
const admitted = new Set(['TestStructuredCancellationRemovesDescendant', 'TestCompatibilityCallerLifetimeKillsLeaderBeforeDescendants']);
const originalFunctions = [...original].map(([name, body]) => ({
  name, admitted_exception:admitted.has(name), baseline_sha256:hash(body),
  current_sha256:hash(current.get(name) ?? ''), preserved:body === current.get(name)
}));
const protectedFiles = [
  'tools/gomad3/internal/hostexec/command.go',
  'tools/gomad3/internal/hostexec/command_unix.go',
  'tools/gomad3/internal/hostexec/command_unix_test.go',
  'tools/gomad3/target/internal/gocommand/command.go'
].map(file => {
  const baseline = git('show', `${base}:${file}`);
  const current = fs.readFileSync(path.join(repo, file), 'utf8');
  return {path:file, baseline_sha256:hash(baseline), current_sha256:hash(current), preserved:baseline === current};
});
const comments = source => source.split('\n').filter(line => /^\s*\/\//.test(line));
const result = {
  baseline:base, current_head:git('rev-parse', 'HEAD').trim(),
  changed_go_scope:git('diff', '--name-only', base, '--', 'tools/gomad3').trim().split('\n').filter(Boolean),
  admitted_fixture_exceptions:[...admitted], original_functions:originalFunctions,
  original_comments_preserved:JSON.stringify(comments(before)) === JSON.stringify(comments(after)),
  protected_files:protectedFiles,
  additions:[...current.keys()].filter(name => !original.has(name)),
  source_file:{path:file, baseline_sha256:hash(before), current_sha256:hash(after)}
};
if (result.changed_go_scope.length !== 1 || result.changed_go_scope[0] !== file ||
    originalFunctions.some(entry => !entry.admitted_exception && !entry.preserved) ||
    protectedFiles.some(entry => !entry.preserved) || !result.original_comments_preserved) {
  throw new Error('preservation mismatch');
}
console.log(JSON.stringify(result, null, 2));
