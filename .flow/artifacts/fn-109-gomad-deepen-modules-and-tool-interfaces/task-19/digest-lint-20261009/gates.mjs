import fs from 'node:fs';
import path from 'node:path';
import {spawnSync} from 'node:child_process';

const out = path.dirname(new URL(import.meta.url).pathname);
const runner = path.join(out, 'run.mjs');
const lint = 'GOLANGCI_LINT_FIX=false GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype';
const gates = [
  ['digest-generated-serialized', 'GOFLAGS=-tags=test_dep make -C tools/gomad3 validate', 0],
  ['digest-architecture-after', 'cd tools/gomad3 && go test -json -count=1 -tags test_dep . -run "^(TestPackageArchitecture|TestPublicPackagesDoNotExportTypeAliases|TestPureModulesHaveNoHostEffects|TestHostPackageVet|TestInspectionSourceOwnershipDelegation|TestExactModuleEdges|TestDomainModulesDoNotExportWireFraming)$"', 0],
  ['digest-source-darwin', 'cd tools/gomad3 && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go list -json -tags test_dep ./internal/gomadtool/architecture', 0],
  ['digest-source-linux', 'cd tools/gomad3 && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go list -json -tags test_dep ./internal/gomadtool/architecture', 0],
  ['digest-errortype-after', 'cd tools/gomad3 && go vet -tags test_dep -vettool=/tmp/fn109-lint-tools.ZdNe1t50/errortype -style-check=false ./internal/gomadtool/architecture', 0],
  ['digest-lint-fast-after', 'make lint-code-fast ' + lint + ' GOLANGCI_LINT_BASE_REV=21b30b4604788c9e2e54b6c33f3e47045725daf9', 0],
  ['digest-integrated-after', 'make lint-code-gomad3 ' + lint + ' GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c', 2],
  ['digest-integrated-later-after', 'make lint-code-gomad3 ' + lint + ' GOLANGCI_LINT_BASE_REV=d635e23f00d926a43b942f25a9d05bd0ccb72025', 2],
  ['digest-format-after', 'bash .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-9/source-acceptance-20261008/format-gate.sh gofmt -l tools/gomad3/internal/gomadtool/architecture/initialization.go tools/gomad3/internal/gomadtool/architecture/standard.go tools/gomad3/internal/gomadtool/architecture/source_digest_test.go && git diff --check', 0],
  ['digest-preservation-after', 'node .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-19/digest-lint-20261009/preservation.mjs', 0],
];
for (const [name, command, expected] of gates) {
  console.log(JSON.stringify({starting: name}));
  const result = spawnSync(process.execPath, [runner, name, command], {stdio: 'inherit'});
  const receipt = JSON.parse(fs.readFileSync(path.join(out, name + '-receipt.json'), 'utf8'));
  if (receipt.exit_code !== expected || !receipt.source_unchanged || !receipt.tools_unchanged || !receipt.controls_unchanged || result.signal || result.error) throw Error('unexpected gate result: ' + name);
}
console.log(JSON.stringify({sequence: 'all required gates terminal', execution_lane: 'free'}));
