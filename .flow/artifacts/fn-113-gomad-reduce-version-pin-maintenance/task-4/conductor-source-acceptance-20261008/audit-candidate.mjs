import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import cp from 'node:child_process';

const root = cp.execFileSync('git', ['rev-parse', '--show-toplevel'], {encoding: 'utf8'}).trim();
const dir = path.join(root, '.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-4/source-acceptance-20261008');
const hash = value => crypto.createHash('sha256').update(value).digest('hex');
const bytes = file => fs.readFileSync(file);
const json = file => JSON.parse(bytes(file));
const check = (value, message) => { if (!value) throw new Error(message); };
const evidence = json(path.join(dir, 'evidence.json'));
const receipt = name => json(path.join(dir, `${name}-receipt.json`));
const events = rec => bytes(path.join(dir, rec.log)).toString().split('\n').flatMap(line => {
  try { return [JSON.parse(line)]; } catch { return []; }
});
const outcomes = rec => new Map(events(rec).filter(event => event.Test && ['pass', 'fail', 'skip'].includes(event.Action)).map(event => [`${event.Package}\0${event.Test}`, event.Action]));
const counts = map => Object.fromEntries(['pass', 'fail', 'skip'].map(action => [action, [...map.values()].filter(value => value === action).length]));
const eq = (a, b) => JSON.stringify(a) === JSON.stringify(b);
const manifests = new Map();
for (const item of evidence.source_manifests) {
  const data = bytes(path.join(dir, item.path));
  check(hash(data) === item.sha256 && data.length === item.bytes, `manifest bytes ${item.path}`);
  const entries = JSON.parse(data);
  const aggregate = hash(JSON.stringify(entries));
  check(item.path === `source-${aggregate}.json`, `manifest aggregate ${item.path}`);
  check(new Set(entries.map(entry => entry.path)).size === entries.length, `duplicate manifest ${item.path}`);
  manifests.set(item.path, entries);
}
const final = manifests.get(`source-${evidence.final_source_tree_sha256}.json`);
for (const item of final) check(hash(bytes(path.join(root, item.path))) === item.sha256, `current source ${item.path}`);
for (const name of ['final-version', 'final-pinimpact-portable', 'final-architecture', 'final-validate', 'final-lint-fast', 'final-version-lint', 'final-vet', 'final-errortype', 'final-darwin-static', 'final-linux-static', 'final-format', 'documentation-audit']) {
  const rec = receipt(name);
  check(rec.exit_code === 0 && rec.source_unchanged && rec.source_tree_sha256 === evidence.final_source_tree_sha256, `owned frozen gate ${name}`);
}
const toolHashes = new Map();
const tested = [];
for (const item of evidence.gate_receipts) {
  const data = bytes(path.join(dir, item.path));
  check(hash(data) === item.sha256, `receipt bytes ${item.name}`);
  const rec = JSON.parse(data);
  for (const key of ['name', 'command', 'exit_code', 'elapsed_seconds', 'source_manifest', 'source_tree_sha256', 'source_unchanged']) check(eq(item[key], rec[key]), `receipt field ${item.name}:${key}`);
  check(manifests.has(rec.source_manifest), `unknown manifest ${item.name}`);
  check(rec.source_manifest === `source-${rec.source_tree_sha256}.json`, `source association ${item.name}`);
  check(hash(bytes(path.join(dir, rec.log))) === rec.log_sha256, `log bytes ${item.name}`);
  for (const tool of rec.tools) {
    if (!toolHashes.has(tool.path)) toolHashes.set(tool.path, hash(bytes(tool.path)));
    check(toolHashes.get(tool.path) === tool.sha256, `tool bytes ${item.name}:${tool.path}`);
  }
  if (item.counts_parser_available) {
    const observed = counts(outcomes(rec));
    check(eq(observed, rec.counts) && eq(observed, item.test_event_counts), `test counts ${item.name}`);
    tested.push({name: item.name, exit: rec.exit_code, counts: observed});
  }
}
for (const item of evidence.artifact_bindings) {
  const data = bytes(path.join(dir, item.path));
  check(hash(data) === item.sha256 && data.length === item.bytes, `artifact bytes ${item.path}`);
}
const broad = receipt('final-authoring');
const rerun = receipt('resolved-refresh-cache');
check(broad.exit_code === 1 && eq(counts(outcomes(broad)), {pass:666, fail:1, skip:1}), 'raw broad remains red');
check(rerun.exit_code === 0 && eq(counts(outcomes(rerun)), {pass:4, fail:0, skip:0}), 'exact cache rerun');
check(rerun.command.includes('-run=^TestRunCompatibilityPackRefreshResolvesTwoModulesAndKeepsPartialApproval$'), 'one-case selection');
const cacheQuery = receipt('resolved-effective-cache');
check(cacheQuery.environment.XDG_CACHE_HOME === rerun.environment.XDG_CACHE_HOME, 'same XDG query');
check(bytes(path.join(dir, cacheQuery.log)).toString().trim() === `${rerun.environment.XDG_CACHE_HOME}/go-build`, 'effective target cache');
const union = outcomes(broad);
for (const [key, value] of outcomes(rerun)) union.set(key, value);
const mapping = json(path.join(dir, 'assertion-mapping.json'));
check(eq(counts(union), mapping.scoped_authoring_union_counts), 'authoring union count');
check(mapping.assertions.length === union.size, 'assertion coverage');
for (const item of mapping.assertions) check(outcomes(json(path.join(dir, item.receipt))).get(`${item.package}\0${item.test}`) === item.outcome, `authoring assertion ${item.test}`);
const pin = outcomes(receipt('final-pinimpact-portable'));
check(mapping.pinimpact_assertions.length === pin.size, 'pin assertion coverage');
for (const item of mapping.pinimpact_assertions) check(pin.get(`${item.package}\0${item.test}`) === item.outcome, `pin assertion ${item.test}`);
const fresh = new Map(union);
for (const name of ['final-version', 'final-pinimpact-portable', 'final-architecture']) for (const [key, value] of outcomes(receipt(name))) fresh.set(key, value);

const docs = json(path.join(dir, 'documentation-audit.json'));
const setup = json(path.join(dir, 'walk-setup.json'));
check(docs.errors.length === 0 && docs.missing_flags.length === 0, 'documentation errors');
for (const item of docs.documents) check(hash(bytes(path.join(root, item.path))) === item.sha256, `doc source ${item.path}`);
for (const item of docs.links) check(item.exists && item.fragment_resolves, `doc link ${item.destination}`);
for (const item of docs.actual_help) {
  check(hash(bytes(path.join(dir, item.log))) === item.sha256, `help log ${item.log}`);
  const expectedStatus = item.tool === 'gomadtool' && item.argv[0] === 'checked-run' ? 125 : 2;
  check(!item.signal && !item.error && item.status === expectedStatus, `help terminal ${item.log}`);
  const toolPath = item.tool === 'gomadtool' ? setup.binary : path.join(setup.work, 'gomad');
  check(hash(bytes(toolPath)) === item.binary_sha256, `actual help binary ${item.log}`);
}
const unfenced = text => {
  let fence = null;
  return text.split('\n').filter(line => {
    const match = line.match(/^\s{0,3}(`{3,}|~{3,})/);
    if (match) {
      if (!fence) fence = match[1];
      else if (match[1][0] === fence[0] && match[1].length >= fence.length) fence = null;
      return false;
    }
    return !fence;
  }).join('\n');
};
const anchors = file => {
  const text = unfenced(bytes(file).toString()), seen = new Map(), result = new Set();
  for (const match of text.matchAll(/^#{1,6}\s+(.+?)\s*#*\s*$/gm)) {
    const slug = match[1].replace(/\[([^\]]+)\]\([^)]*\)/g, '$1').toLowerCase().replace(/[^\p{L}\p{N}_\- ]/gu, '').replaceAll(' ', '-');
    const index = seen.get(slug) ?? 0;
    seen.set(slug, index+1);
    result.add(slug+(index ? '-'+index : ''));
  }
  for (const match of text.matchAll(/<a\s+(?:id|name)=["']([^"']+)/g)) result.add(match[1]);
  return result;
};
let observedLinks = 0;
for (const item of docs.documents) for (const match of unfenced(bytes(path.join(root, item.path)).toString()).matchAll(/\[([^\]]+)\]\(([^)\n]+)\)/g)) {
  const destination = match[2];
  if (/^[a-zA-Z][a-zA-Z0-9+.-]*:/.test(destination)) continue;
  const [target, fragment] = destination.split('#');
  const absolute = target ? path.resolve(path.dirname(path.join(root, item.path)), decodeURIComponent(target)) : path.join(root, item.path);
  check(fs.existsSync(absolute) && (!fragment || !absolute.endsWith('.md') || anchors(absolute).has(decodeURIComponent(fragment))), `actual doc destination ${item.path}:${destination}`);
  observedLinks++;
}
check(observedLinks === docs.links.length, 'complete link coverage');
for (const [tool, file] of Object.entries({gomad:'tools/gomad3/cmd/gomad/internal/cli/cli.go', gomadtool:'tools/gomad3/cmd/gomadtool/main.go'})) {
  const text = bytes(path.join(root,file)).toString();
  const dispatch = [...text.slice(0, text.indexOf('\nfunc ', text.indexOf('switch '))).matchAll(/case "([a-z][a-z-]*)":/g)].map(match => match[1]).sort();
  check(eq(dispatch, [...docs.dispatch[tool]].sort()), `actual dispatch ${tool}`);
  for (const inventory of [docs.index[tool], docs.normative[tool]]) check(eq([...new Set(inventory.filter(name => !name.includes(' ')))].sort(), dispatch), `complete documented commands ${tool}`);
}
const oldSpec = cp.execFileSync('git', ['show', `${evidence.base_commit}:tools/gomad3/SPEC.md`], {encoding:'utf8'});
const ids = text => new Set([...text.matchAll(/\[([A-Z][A-Z0-9_.]+)\]/g)].map(match => match[1]));
const oldIds = ids(oldSpec);
const newIds = ids(bytes(path.join(root, 'tools/gomad3/SPEC.md')).toString());
for (const id of oldIds) check(newIds.has(id), `preserved semantic identifier ${id}`);
check(eq([...newIds].filter(id => !oldIds.has(id)), [docs.intentional_added_identifier]), 'only intentional identifier addition');

const walk = json(path.join(dir, 'walkthrough.json'));
const binary = path.join(path.dirname(walk.parent), 'gomadtool');
check(hash(bytes(binary)) === walk.binary_sha256, 'walk binary');
for (const item of walk.outputs) check(hash(bytes(path.join(walk.root, item.path))) === item.sha256, `published output ${item.path}`);
const applied = json(path.join(dir, 'walk-adapter-apply.log'));
check(applied.applied === true && (!applied.warnings || applied.warnings.length === 0), 'clean original publication');
check(eq([...applied.published].sort(), walk.outputs.map(item=>item.path).sort()), 'all original CLI published outputs');
for (const item of setup.initial_scratch) {
  const relative = item.path.slice('tools/gomad3/'.length);
  if (!applied.published.includes(relative)) check(hash(bytes(path.join(walk.root,relative))) === item.sha256, `unchanged scratch Gomad ${relative}`);
}
for (const item of walk.candidate_after) check(hash(bytes(path.join(setup.candidate,item.path))) === item.sha256, `walk candidate ${item.path}`);
check(applied.regeneration.previous.version === 'v3.3.0' && applied.regeneration.proposed.version === 'v3.2.3', 'exact real bump');
check(receipt('walk-adapter-apply').command.includes(`--approve-review='${walk.approval.approval}'`) && !receipt('walk-adapter-apply').command.includes('--pipeline'), 'original default approved pipeline');
check(hash(bytes(path.join(dir, walk.approval.review_log))) === walk.approval.review_log_sha256, 'reviewed scratch log');
check(bytes(path.join(dir, walk.approval.review_log)).toString().includes(walk.approval.approval), 'matching rendered approval');
for (const command of walk.commands) {
  const rec = json(path.join(dir, command.receipt));
  check(rec.command === command.command && rec.exit_code === command.exit_code, `walk invocation ${command.name}`);
}
check(walk.observed_command_invocations === 9 && walk.actual_total_units === 10 && walk.instrumentation_inclusive_total_units === 16, 'honest effort accounting');
const preservation = json(path.join(dir, 'preservation.json'));
const surrounding = json(path.join(dir, preservation.scratch_surrounding_manifest));
check(hash(JSON.stringify(surrounding)) === '036dc064c0b15f6dae532d8275818d144697ed2ca6e95ecd392eea3afe55a7da', 'surrounding aggregate');
for (const item of surrounding) check(hash(bytes(path.join(walk.parent, item.path))) === item.sha256, `scratch surrounding ${item.path}`);
for (const item of [...preservation.historical_artifacts, ...preservation.user_files]) check(hash(bytes(path.join(root, item.path))) === item.sha256, `preserved original ${item.path}`);
for (const file of preservation.production_pins_exact) check(eq(bytes(path.join(root, file)), cp.execFileSync('git', ['show', `${evidence.base_commit}:${file}`])), `unchanged production pin ${file}`);
const changes = cp.execFileSync('git', ['diff', '--name-only', evidence.base_commit, '--', 'tools/gomad3'], {encoding:'utf8'}).trim().split('\n').filter(Boolean).sort();
check(eq(changes, [...preservation.changed_source_paths].sort()), 'exact Gomad change scope');
let preserved = 0;
for (const item of final.filter(item => item.path.startsWith('tools/gomad3/') && !changes.includes(item.path))) {
  check(hash(cp.execFileSync('git', ['show', `${evidence.base_commit}:${item.path}`])) === item.sha256, `unchanged Gomad ${item.path}`);
  preserved++;
}
check(preserved === preservation.unchanged_gomad_paths, 'unaffected count');
const matched = json(path.join(dir, 'matched-first-baseline.json'));
for (const item of [matched.original_baseline, matched.matched_measurement]) check(hash(bytes(path.join(root, item.path))) === item.sha256, `matched baseline ${item.path}`);
for (const item of matched.repair_paths) {
  check(hash(cp.execFileSync('git', ['show', `${matched.original_baseline.head}:${item.path}`])) === item.first_baseline_sha256, `original repair ${item.path}`);
  check(hash(bytes(path.join(root, item.path))) === item.current_sha256, `current repair ${item.path}`);
}

const reuse = json(path.join(dir, 'predecessor-source-reuse.json'));
check(hash(bytes(path.join(root, reuse.predecessor_manifest))) === reuse.predecessor_manifest_sha256, 'predecessor manifest');
const predecessor = new Map(json(path.join(root, reuse.predecessor_manifest)).map(item=>[item.path,item.sha256]));
check(hash(bytes(path.join(dir, receipt('reuse-deterministicio-dependencies').log))) === reuse.dependency_listing_sha256, 'dependency listing');
const listing = bytes(path.join(dir, receipt('reuse-deterministicio-dependencies').log)).toString();
const packages = [];
let start = -1, depth = 0, quoted = false, escaped = false;
for (let i = 0; i < listing.length; i++) {
  const char = listing[i];
  if (quoted) {
    if (escaped) escaped = false;
    else if (char === '\\') escaped = true;
    else if (char === '"') quoted = false;
  } else if (char === '"') quoted = true;
  else if (char === '{') {if (depth === 0) start=i; depth++;}
  else if (char === '}') {depth--; if (depth === 0) packages.push(JSON.parse(listing.slice(start,i+1)));}
}
check(depth === 0 && !quoted, 'complete dependency JSON stream');
const closure = new Set(['go.mod','go.sum','tools/gomad3/go.mod','tools/gomad3/go.sum']);
const generated = new Set();
for (const pkg of packages) {
  if (!pkg.Dir?.startsWith(path.join(root,'tools/gomad3'))) continue;
  for (const file of [...pkg.GoFiles??[], ...pkg.CgoFiles??[], ...pkg.SFiles??[], ...pkg.EmbedFiles??[]]) {
    if (path.isAbsolute(file)) generated.add(file);
    else closure.add(path.relative(root,path.join(pkg.Dir,file)));
  }
  if (pkg.ImportPath === 'go.temporal.io/server/tools/gomad3/deterministicio') for (const file of [...pkg.TestGoFiles??[], ...pkg.XTestGoFiles??[], ...pkg.TestEmbedFiles??[], ...pkg.XTestEmbedFiles??[]]) closure.add(path.relative(root,path.join(pkg.Dir,file)));
}
check(eq([...closure].sort(),reuse.dependency_paths.map(item=>item.path).sort()), 'actual full local dependency/test/embed coverage');
check(eq([...generated].sort(),reuse.generated_test_main.map(item=>item.path).sort()), 'actual test main coverage');
let exact = 0;
for (const item of reuse.dependency_paths) {
  check(hash(bytes(path.join(root, item.path))) === item.sha256, `reuse input ${item.path}`);
  check(predecessor.get(item.path) === item.predecessor_sha256, `actual predecessor input ${item.path}`);
  if (item.path !== reuse.descriptor_exception.path) check(!/\bgomadversion\.(Generate|GeneratedFiles)\b/.test(bytes(path.join(root,item.path)).toString()), `no guide generator in reused input ${item.path}`);
  if (item.exact) { check(item.sha256 === item.predecessor_sha256, `exact reuse ${item.path}`); exact++; }
}
const descriptor = reuse.descriptor_exception;
const oldDescriptor = cp.execFileSync('git', ['show', `ca6fd855868fac364b69cb31394c87ad2912e623:${descriptor.path}`], {encoding:'utf8'});
const currentDescriptor = bytes(path.join(root, descriptor.path)).toString();
check(hash(oldDescriptor) === descriptor.old_sha256 && hash(currentDescriptor) === descriptor.current_sha256, 'descriptor identity');
const outsideGuide = text => {
  const start = text.indexOf('func renderUpgradeGuide(');
  const end = text.indexOf('\nfunc ', start+1);
  check(start >= 0 && end > start, 'bounded guide function');
  return text.slice(0,start)+text.slice(end);
};
check(outsideGuide(oldDescriptor) === outsideGuide(currentDescriptor), 'all descriptor bytes outside guide exact');
for (const packet of reuse.packets) {
  check(hash(bytes(path.join(root, packet.receipt))) === packet.receipt_sha256, 'accepted predecessor receipt');
  const rec = json(path.join(root, packet.receipt));
  check(rec.command === packet.command && rec.source_tree_sha256 === packet.source_tree_sha256 && rec.exit_code === 0, 'predecessor binding');
  const priorDir = path.dirname(path.join(root, packet.receipt));
  check(hash(bytes(path.join(priorDir, rec.log))) === packet.log_sha256, 'predecessor raw log');
  const data = bytes(path.join(priorDir, rec.log)).toString().split('\n').flatMap(line => {try{return[JSON.parse(line)];}catch{return[];}});
  const observed = new Map(data.filter(event => event.Test && ['pass','fail','skip'].includes(event.Action)).map(event => [`${event.Package}\0${event.Test}`,event.Action]));
  check(eq(counts(observed), packet.counts), 'accepted predecessor counts');
}
for (const item of reuse.generated_test_main) check(hash(bytes(item.path)) === item.sha256, 'generated test-main bytes');

console.log(JSON.stringify({source:evidence.final_source_tree_sha256, manifests:manifests.size, current_source_paths:final.length, receipts:evidence.gate_receipts.length, tools:toolHashes.size, artifacts:evidence.artifact_bindings.length, observed_test_receipts:tested, source_authoring_union:counts(union), unique_fresh_source_assertions:counts(fresh), documents:docs.documents.length, links:docs.links.length, registered_commands:Object.values(docs.dispatch).reduce((sum,list)=>sum+list.length,0), actual_help:docs.actual_help.length, walk_outputs:walk.outputs.length, scratch_surrounding_paths:surrounding.length, unaffected_gomad_paths:preserved, historical_task4_artifacts:preservation.historical_artifacts.length, user_files:preservation.user_files.length, matched_baseline_paths:matched.repair_paths.length, reuse_paths:reuse.dependency_paths.length, reuse_exact:exact, reuse_packets:reuse.packets.length, native_execution:false, broad_full_pass:false, review_verdict:null},null,2));
