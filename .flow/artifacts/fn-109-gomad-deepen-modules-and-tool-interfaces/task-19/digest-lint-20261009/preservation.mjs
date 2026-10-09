import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';

const repo = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out = path.dirname(new URL(import.meta.url).pathname);
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const git = (...args) => {
  const result = spawnSync('git', args, {cwd: repo, encoding: 'utf8'});
  if (result.status !== 0) throw Error(result.stderr);
  return result.stdout;
};
const framing = '7b39baa38a2ec2b8d111bbbd8e448e80226477ab40105d9d2123d4dc18067438  a.go\nfc852e86c6ea2bc13f6521e13cfb58dd977aa91a136e37ad3ff5acc8a81170cb  z.go\n';
if (hash('package a\n') !== framing.slice(0, 64) || hash('package z\n') !== framing.split('\n')[1].slice(0, 64) || hash(framing) !== '67ad170a9788e8b8d82ab27c22a54c6f98af43713537fa5775a11f1424d4bcd5') throw Error('literal fixture identity differs');
console.log(JSON.stringify({literal_framing: framing, literal_digest: hash(framing)}));
const old = '\t\tfmt.Fprintf(digest, "%x  %s\\n", sha256.Sum256(data), entry.Name())';
const replacement = '\t\t_, _ = digest.Write(fmt.Appendf(nil, "%x  %s\\n", sha256.Sum256(data), entry.Name()))';
for (const file of ['initialization.go', 'standard.go']) {
  const relative = 'tools/gomad3/internal/gomadtool/architecture/' + file;
  const before = git('show', '21b30b4604788c9e2e54b6c33f3e47045725daf9:' + relative);
  const first = git('show', '4a646710d15148f1a3cf75bdeba7bdfd2fd2edf7:' + relative);
  const after = fs.readFileSync(path.join(repo, relative), 'utf8');
  if (before.split(old).length !== 2 || first.split(old).length !== 2 || before.replace(old, replacement) !== after) throw Error('production scope differs: ' + file);
  console.log(JSON.stringify({path: relative, base_sha256: hash(before), first_architecture_checkpoint_sha256: hash(first), final_sha256: hash(after), exact_statement_only: true, original_statement_present_in_architecture_checkpoint: true}));
}
const protectedFiles = git('ls-tree', '-r', '--name-only', '21b30b4604788c9e2e54b6c33f3e47045725daf9', '--', 'tools/gomad3').trim().split('\n').filter(file => file.endsWith('_test.go') || file.endsWith('startup_sources.go') || file.endsWith('memory_sources.go') || file.endsWith('go.mod') || file.endsWith('go.sum'));
if (protectedFiles.length !== 401) throw Error('baseline protected inventory scope differs');
for (const file of protectedFiles) {
  if (hash(fs.readFileSync(path.join(repo, file))) !== hash(git('show', '21b30b4604788c9e2e54b6c33f3e47045725daf9:' + file))) throw Error('protected input changed: ' + file);
}
console.log(JSON.stringify({protected_inputs_unchanged: protectedFiles.length}));
const frozenSource = 'source-eb4136469addda8629a61bab47a4a95c015ca60f1547dc17b10edb048b93ecfd.json';
const sourceBytes = fs.readFileSync(path.join(out, frozenSource));
if (hash(sourceBytes) !== 'eb4136469addda8629a61bab47a4a95c015ca60f1547dc17b10edb048b93ecfd') throw Error('frozen source manifest changed');
const additive = 'tools/gomad3/internal/gomadtool/architecture/source_digest_test.go';
const fixture = JSON.parse(sourceBytes).find(entry => entry.path === additive);
if (!fixture || hash(fs.readFileSync(path.join(repo, additive))) !== fixture.sha256) throw Error('additive fixture changed');
console.log(JSON.stringify({additive_fixture: additive, sha256: fixture.sha256, frozen_source_manifest: frozenSource}));
for (const [name, total, errcheck] of [['digest-integrated', 215, 160], ['digest-integrated-later', 24, 20]]) {
  const before = fs.readFileSync(path.join(out, name + '-before.stdout'), 'utf8');
  const after = fs.readFileSync(path.join(out, name + '-after.stdout'), 'utf8');
  let expected = before;
  for (const [file, line] of [['initialization.go', 123], ['standard.go', 392]]) {
    const block = `tools/gomad3/internal/gomadtool/architecture/${file}:${line}:14: Error return value of \`fmt.Fprintf\` is not checked (errcheck)\n\t\tfmt.Fprintf(digest, "%x  %s\\n", sha256.Sum256(data), entry.Name())\n\t\t           ^\n`;
    if (expected.split(block).length !== 2) throw Error('missing exact diagnostic ' + file);
    expected = expected.replace(block, '');
  }
  expected = expected.replace(`${total} issues:\n`, `${total - 2} issues:\n`).replace(`* errcheck: ${errcheck}\n`, `* errcheck: ${errcheck - 2}\n`);
  if (expected !== after) throw Error('residual diagnostic report changed: ' + name);
  console.log(JSON.stringify({report: name, before: total, after: total - 2, residual_stdout_byte_identical: true, before_sha256: hash(before), after_sha256: hash(after)}));
}
const scoped = fs.readFileSync(path.join(out, 'digest-lint-red.stdout'), 'utf8');
if (!scoped.endsWith('2 issues:\n* errcheck: 2\n') || fs.readFileSync(path.join(out, 'digest-lint-green.stdout'), 'utf8') !== '0 issues.\n') throw Error('scoped analyzer delta differs');
console.log(JSON.stringify({scoped_analyzer: '2 to 0', source_acceptance: 'open', native_qualification: 'deferred unverified'}));
