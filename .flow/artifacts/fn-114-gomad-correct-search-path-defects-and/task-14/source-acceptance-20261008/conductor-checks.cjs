const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const cp = require('node:child_process');

const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const go = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go';
const digest = value => crypto.createHash('sha256').update(value).digest('hex');
const audit = JSON.parse(fs.readFileSync(path.join(__dirname, 'final-docs.json')));
const verify = () => {
  for (const [name, expected] of Object.entries({...audit.inputs, ...audit.qualification_manifests})) {
    if (digest(fs.readFileSync(path.join(root, name))) !== expected) throw Error(`Changed frozen input: ${name}`);
  }
  if (audit.errors.length) throw Error('Documentation audit has errors');
};
verify();
const env = {...process.env, GOTOOLCHAIN: 'local', GOWORK: 'off', GOPROXY: 'off', GOSUMDB: 'off', GOENV: 'off', GOFLAGS: '', GOEXPERIMENT: 'nogreenteagc', GOMAXPROCS: '2', PATH: path.dirname(go) + ':' + process.env.PATH};
delete env.GOMADSEED;
delete env.GOMAD3_CHILD_SEED;
const selection = '^(TestCharacterize(InputRejection|ByteSizeAndFlagValueRejection|ExploreRequestDefaultsAndWiring)|TestWorkspace(Resume.*|KeepsStatePerParentArtifact|RejectsSymbolicLinkStateRoot|StateGatesInitialRunsAndResumes)|TestIdentityForProjectsCanonicalEnvironmentAcrossSeedsAndProfiles|TestCorpusRejects(IdentityChangesAndNonMatchingReplay|PreviousAndFutureSchemaBeforeChangedIdentity|CaseWithChangedEnvironment)|TestValidateProvenanceRejectsUnsupportedBuildModes|TestReplayBuildInfoRejectsMatchingCoverageInstrumentation)$';
const commands = [
  ['portable-documentation-contracts', [go, '-C', 'tools/gomad3', 'test', '-tags', 'test_dep', '-count=1', '-json', '-run', selection, './cmd/gomad/internal/cli', './runner/internal/minimizer', './runner/internal/corpus', './target', './runner']],
  ['generated-source-validation', ['make', '-C', 'tools/gomad3', 'validate']],
  ['documentation-whitespace', ['git', 'diff', '--check']],
];
const result = {head: cp.execFileSync('git', ['rev-parse', 'HEAD'], {cwd: root, encoding: 'utf8'}).trim(), go_sha256: digest(fs.readFileSync(go)), environment: Object.fromEntries(['GOTOOLCHAIN', 'GOWORK', 'GOPROXY', 'GOSUMDB', 'GOENV', 'GOFLAGS', 'GOEXPERIMENT', 'GOMAXPROCS', 'PATH'].map(key => [key, env[key]])), unset_environment: ['GOMADSEED', 'GOMAD3_CHILD_SEED'], audit_sha256: digest(fs.readFileSync(path.join(__dirname, 'final-docs.json'))), input_count: Object.keys(audit.inputs).length, qualification_manifest_count: Object.keys(audit.qualification_manifests).length, observations: []};
for (const [name, argv] of commands) {
  verify();
  const started = new Date().toISOString();
  const timer = performance.now();
  const ran = cp.spawnSync(argv[0], argv.slice(1), {cwd: root, env, timeout: 600000, maxBuffer: 16 * 1024 * 1024});
  const item = {name, argv, cwd: root, started, ended: new Date().toISOString(), elapsed_seconds: (performance.now() - timer) / 1000, exit: ran.status, signal: ran.signal, error: ran.error ? String(ran.error) : null, stdout_base64: (ran.stdout || Buffer.alloc(0)).toString('base64'), stderr_base64: (ran.stderr || Buffer.alloc(0)).toString('base64')};
  result.observations.push(item);
  fs.writeFileSync(path.join(__dirname, 'conductor-checks.json'), JSON.stringify(result, null, 2) + '\n');
  console.log(JSON.stringify({name, exit: ran.status, elapsed_seconds: item.elapsed_seconds}));
  if (ran.status !== 0 || ran.error) process.exit(1);
  verify();
}
console.log(JSON.stringify({frozen_inputs_verified: result.input_count, qualification_manifests_verified: result.qualification_manifest_count}));
