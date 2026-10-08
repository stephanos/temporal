import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import zlib from 'node:zlib';

const repo = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const dir = path.join(repo, '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-7/conductor-source-acceptance-20261008');
const fanout = path.join(repo, '.flow/review-fanout/97bb53d7a90346ef879e1e6b0ba38bd0');
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const output = path.join(dir, 'review-first-round-evidence.json');
if (fs.existsSync(output)) throw new Error('First-round archive already exists');
const originals = [...fs.readdirSync(fanout).sort().map(name => path.join(fanout, name)), path.join(dir, 'review-receipt.json')];
const entries = originals.map(file => {
  const bytes = fs.readFileSync(file);
  const encoded = zlib.gzipSync(bytes).toString('base64');
  if (!zlib.gunzipSync(Buffer.from(encoded, 'base64')).equals(bytes)) throw new Error('Lossless archive check');
  return {path: path.relative(repo, file), bytes: bytes.length, sha256: hash(bytes), encoding: 'gzip+base64', data: encoded};
});
const meta = JSON.parse(fs.readFileSync(path.join(fanout, 'meta.json')));
fs.writeFileSync(output, JSON.stringify({rid: meta.rid, task: meta.id, reviewed_base_sha: meta.reviewed_base_sha, reviewed_head_sha: meta.reviewed_head_sha, verdict: 'NEEDS_WORK', retained_finding: 'contracts:1', rejected_findings: [], optional_phases: 0, writer_reviewer_family: 'same Codex family; requested worker gpt-6.1-sol/high; actual reviewer gpt-6.1-sol/high; worker actual model metadata unavailable', originals: entries}, null, 2) + '\n');
const retained = JSON.parse(fs.readFileSync(output));
for (const entry of retained.originals) {
  const bytes = zlib.gunzipSync(Buffer.from(entry.data, 'base64'));
  if (bytes.length !== entry.bytes || hash(bytes) !== entry.sha256 || !bytes.equals(fs.readFileSync(path.join(repo, entry.path)))) throw new Error('Archive verification: ' + entry.path);
}
console.log(JSON.stringify({retained_originals: entries.length, archive_bytes: fs.statSync(output).size, verdict: retained.verdict, rid: retained.rid, lossless: true}));
