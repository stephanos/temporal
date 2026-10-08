const fs = require('node:fs');
const path = require('node:path');
const cp = require('node:child_process');
const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const scratch = path.join(root, '.flow/tmp/fn10928-exception');
const go = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go';
const controller = 'tools/gomad3/runner/internal/campaign/controller.go';
const source = fs.readFileSync(path.join(root, controller), 'utf8');
const variants = {
  duplicate: source + '\nfunc unintended(active bool) {\n\tif !active {\n\t\tpanic("gomad3: completed an inactive campaign attempt")\n\t}\n}\n',
  moved: source.replace(') Complete(completion Completion)', ') Other(completion Completion)'),
  condition: source.replace('if controller.active == 0 {', 'if controller.active < 0 {'),
};
for (const [name, mutated] of Object.entries(variants)) {
  if (mutated === source) throw new Error('mutation did not apply: ' + name);
  const fixture = fs.mkdtempSync(path.join(scratch, name + '-'));
  for (const file of ['go.mod', 'go.sum', '.github/.golangci.yml', controller]) {
    const dest = path.join(fixture, file);
    fs.mkdirSync(path.dirname(dest), {recursive:true});
    fs.copyFileSync(path.join(root, file), dest);
  }
  fs.cpSync(path.join(root, 'cmd/tools/lintcode'), path.join(fixture, 'cmd/tools/lintcode'), {recursive:true});
  fs.writeFileSync(path.join(fixture, controller), mutated);
  const result = cp.spawnSync('node', [path.join(__dirname, 'run.cjs'), 'source-binding-' + name, fixture, go, '-C', fixture, 'test', '-count=1', '-tags', 'test_dep', '-v', '-run', '^TestLintPolicyCampaignInvariantSourceBinding$', './cmd/tools/lintcode'], {cwd:root, stdio:'inherit'});
  if (result.status !== 1) throw new Error('expected source-policy rejection for ' + name + ', got ' + result.status);
}
