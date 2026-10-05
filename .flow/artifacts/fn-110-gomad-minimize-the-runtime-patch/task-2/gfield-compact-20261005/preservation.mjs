import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out = path.dirname(fileURLToPath(import.meta.url));
const scratch = path.join(root, 'tools/gomad3/.toolchain/fn-110/gfield-compact.gxrPVUi5');
const stock = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin';
const overlay = 'tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go';
const descriptorPath = 'tools/gomad3/toolchain/version/version.json';
const descriptor = JSON.parse(fs.readFileSync(path.join(root, descriptorPath)));
const paths = [...descriptor.patch_allowlist, overlay];
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const read = file => fs.readFileSync(file);
const source = name => path.join(name === overlay ? root : path.join(scratch, 'go'), name);
const names = { gomadIdentity: 'gomadID', gomadChildOrdinal: 'gomadChild', gomadTimerOrdinal: 'gomadTimer', gomadSimulationDomain: 'gomadDomain', gomadSimulationTransport: 'gomadSimIO' };
const expected = { 'src/runtime/runtime2.go': [1, 1, 1, 1, 1], 'src/runtime/proc.go': [0, 0, 0, 2, 1], [overlay]: [7, 4, 4, 3, 4] };
const userPaths = ['.turbo/plans/gomad3-glossary-update.md', '.turbo/technical-debt.md'];
function sources() {
  return Object.fromEntries(execFileSync('git', ['ls-files', '-z'], { cwd: root, maxBuffer: 32 << 20 }).toString().split('\0').filter(name => name && !name.startsWith('.flow/') && fs.existsSync(path.join(root, name)) && fs.statSync(path.join(root, name)).isFile()).sort().map(name => [name, hash(read(path.join(root, name)))]));
}
function format(bytes) { return execFileSync(path.join(stock, 'gofmt'), [], { input: bytes, maxBuffer: 16 << 20 }); }
if (process.argv[2] === 'save') {
  assert(!fs.existsSync(path.join(scratch, 'before')), 'baseline snapshot already exists');
  for (const name of paths) {
    const destination = path.join(scratch, 'before', name);
    fs.mkdirSync(path.dirname(destination), { recursive: true });
    fs.copyFileSync(source(name), destination);
  }
  const all = sources();
  const info = { base_commit: execFileSync('git', ['rev-parse', 'HEAD'], { cwd: root }).toString().trim(), sources: all, user_hashes: Object.fromEntries(userPaths.map(name => [name, hash(read(path.join(root, name)))])), archive_sha256: hash(read(path.join(root, 'tools/gomad3/.toolchain/downloads/go1.27.1.src.tar.gz'))), descriptor_sha256: hash(read(path.join(root, descriptorPath))) };
  assert.equal(info.base_commit, '1b0bc277589d141aca8b534b03135ab3e57fc050');
  assert.equal(info.archive_sha256, descriptor.archive.sha256);
  assert(read(path.join(scratch, 'baseline-U1.patch')).equals(read(path.join(root, 'tools/gomad3/toolchain/runtime/go1.27.1.patch'))));
  fs.writeFileSync(path.join(scratch, 'freeze.json'), JSON.stringify(info));
  console.log('saved 21 files; source count', Object.keys(all).length, 'aggregate', hash(JSON.stringify(all)), 'archive', info.archive_sha256);
} else {
  const freeze = JSON.parse(read(path.join(scratch, 'freeze.json')));
  assert.equal(hash(read(path.join(root, descriptorPath))), freeze.descriptor_sha256);
  assert.equal(descriptor.patch_allowlist.length, 20);
  assert.equal(descriptor.overlay_allowlist.length, 79);
  const files = [];
  let total = 0;
  for (const name of paths) {
    const before = read(path.join(scratch, 'before', name));
    const after = read(source(name));
    const fresh = name === overlay ? after : read(path.join(scratch, 'final', 'go', name));
    let alpha = before.toString();
    const counts = [];
    for (const [oldName, newName] of Object.entries(names)) {
      assert(!new RegExp('\\b' + newName + '\\b').test(alpha), 'short name collision: ' + name + ':' + newName);
      let count = 0;
      alpha = alpha.replace(new RegExp('\\.' + oldName + '\\b', 'g'), () => { count++; return '.' + newName; });
      if (name === 'src/runtime/runtime2.go') {
        alpha = alpha.replace(new RegExp('^(\\t)' + oldName + '(\\s+)', 'gm'), (_, indent, spacing) => { count++; return indent + newName + spacing; });
      }
      counts.push(count);
    }
    assert.deepEqual(counts, expected[name] ?? [0, 0, 0, 0, 0], name);
    const normalized = format(alpha);
    assert(normalized.equals(format(after)), 'alpha-gofmt mismatch: ' + name);
    assert(normalized.equals(format(fresh)), 'fresh materialization mismatch: ' + name);
    if (name !== overlay) assert(format(fresh).equals(fresh), 'fresh source not canonical: ' + name);
    total += counts.reduce((a, b) => a + b, 0);
    files.push({ path: name, counts, before_sha256: hash(before), final_sha256: hash(fresh), alpha_gofmt_sha256: hash(normalized), equal: true });
  }
  assert.equal(total, 30);
  for (const name of userPaths) assert.equal(hash(read(path.join(root, name))), freeze.user_hashes[name]);
  const current = sources();
  const changes = Object.keys(current).filter(name => current[name] !== freeze.sources[name]);
  const patch = read(path.join(root, 'tools/gomad3/toolchain/runtime/go1.27.1.patch'));
  const patchedPaths = [...patch.toString().matchAll(/^diff --git a\/(\S+) b\/\1$/gm)].map(match => match[1]);
  assert.deepEqual(patchedPaths, descriptor.patch_allowlist);
  const overlayPaths = [];
  function walk(dir, prefix = '') { for (const item of fs.readdirSync(dir, { withFileTypes: true })) { const relative = prefix + item.name; if (item.isDirectory()) walk(path.join(dir, item.name), relative + '/'); else { assert(item.isFile()); overlayPaths.push(relative); } } }
  walk(path.join(root, 'tools/gomad3/toolchain/runtime/overlay'));
  assert.deepEqual(overlayPaths.sort(), descriptor.overlay_allowlist);
  assert(patch.equals(read(path.join(scratch, 'final-U1.patch'))));
  const measurements = Object.fromEntries(['baseline', 'final'].flatMap(phase => [1, 3].map(context => { const bytes = read(path.join(scratch, `${phase}-U${context}.patch`)); return [`${phase}-U${context}`, { bytes: bytes.length, lines: bytes.toString().split('\n').length - 1, sha256: hash(bytes) }]; })));
  const g = read(path.join(scratch, 'final/go/src/runtime/runtime2.go')).toString().match(/^type g struct \{\n(.*?)^\}/ms)[1];
  const fields = [...g.matchAll(/^\t(\w+)\s+([^\n]+)$/gm)].map(match => [match[1], match[2].split('//')[0].trim()]);
  const position = fields.findIndex(([name]) => name === 'gomadID');
  assert.deepEqual(fields.slice(position, position + 5), [['gomadID', '[32]byte'], ['gomadChild', 'uint64'], ['gomadTimer', 'uint64'], ['gomadDomain', 'uint64'], ['gomadSimIO', 'bool']]);
  assert.equal(fields[position - 1][0], 'goid');
  assert.equal(fields[position + 5][0], 'schedlink');
  const info = { scratch, base_commit: freeze.base_commit, source_count: Object.keys(current).length, sources_before_sha256: hash(JSON.stringify(freeze.sources)), sources_final_sha256: hash(JSON.stringify(current)), changes, files, total_renames: total, g_field_position: position, g_field_layout_preserved_by_complete_alpha_gofmt_equality: true, user_hashes: freeze.user_hashes, archive_sha256: freeze.archive_sha256, descriptor_sha256: freeze.descriptor_sha256, exact_patch_paths: 20, exact_overlay_paths: 79, measurements };
  fs.writeFileSync(path.join(out, 'preservation.json'), JSON.stringify(info, null, 2) + '\n');
  fs.writeFileSync(path.join(scratch, 'final-sources.json'), JSON.stringify(current));
  console.log('complete alpha + pinned gofmt equality: 21 files, 30 field sites; exact allowlists 20/79; layout and user files unchanged');
  console.log(JSON.stringify(measurements));
  console.log('product changes', changes);
}
