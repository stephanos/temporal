import fs from 'node:fs';
import {spawnSync} from 'node:child_process';
const out='.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/source-acceptance-20261008';
const packages='./cmd/gomadtool ./upgrade/... ./deterministicio/... ./internal/compatibilitypack/... ./toolchain/version';
const selection=JSON.parse(fs.readFileSync(out+'/deterministicio-selection.json','utf8'));
const selected=selection.expression;
const cleanupNames=['TestAdapterCacheCleanupOwnerErrors','TestAdapterCacheCleanupRegistryPublicationReuseRetry','TestAdapterCacheCleanupValidationControls','TestAdapterCacheCleanupRegistryPrimary'];
const complement='^('+selection.selected.filter(test=>!cleanupNames.includes(test.name)).map(test=>test.name).join('|')+')$';
const overlay=process.cwd()+'/'+out+'/retained-assertions-overlay.json';
const commands={
  'accepted-public-packages': 'go -C tools/gomad3 test -tags test_dep -count=1 -json ./cmd/gomadtool ./upgrade/adapterregen ./toolchain/version ./internal/compatibilitypack/...',
  'final-bound-public-packages': 'go -C tools/gomad3 test -tags test_dep -count=1 -json ./cmd/gomadtool ./upgrade/adapterregen ./toolchain/version ./internal/compatibilitypack/...',
  'accepted-deterministicio': "go -C tools/gomad3 test -tags test_dep -count=1 -json -run '"+selected+"' ./deterministicio",
  'final-bound-deterministicio': "go -C tools/gomad3 test -tags test_dep -count=1 -json -run '"+selected+"' ./deterministicio",
  'accepted-deterministicio-workspace-complement': "go -C tools/gomad3 test -tags test_dep -count=1 -json -run '"+complement+"' ./deterministicio",
  'accepted-cache-tmpfs': "TMPDIR=/dev/shm/gomad-fn1132-cleanup-3pOD0l3j go -C tools/gomad3 test -tags test_dep -count=1 -json -run '^TestAdapterCacheCleanup(OwnerErrors|RegistryPublicationReuseRetry|ValidationControls|RegistryPrimary)$' ./deterministicio",
  'accepted-wire': 'go -C tools/gomad3 test -tags test_dep -count=1 -json ./deterministicio/internal/...',
  'accepted-architecture': 'go -C tools/gomad3 test -tags test_dep -count=1 -json .',
  'accepted-retained-generation': 'go run '+out+'/retained_overlay.go',
  'accepted-retained-assertions': "go -C tools/gomad3 test -tags test_dep -count=1 -json -overlay="+overlay+" -run '^TestPortableRetained' ./deterministicio",
  'accepted-retained-list': 'go -C tools/gomad3 list -tags test_dep -overlay='+overlay+' -json ./deterministicio',
  'accepted-vet': 'go -C tools/gomad3 vet -tags test_dep '+packages,
  'accepted-errortype': 'go -C tools/gomad3 vet -tags test_dep -vettool=/tmp/fn109-lint-tools.ZdNe1t50/errortype -errortype.style=false '+packages,
  'final-bound-errortype': 'go -C tools/gomad3 vet -tags test_dep -vettool=/tmp/fn109-lint-tools.ZdNe1t50/errortype -style-check=false '+packages,
  'accepted-darwin-static': 'GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go -C tools/gomad3 list -deps -test -tags test_dep '+packages+' && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go -C tools/gomad3 vet -tags test_dep '+packages,
  'accepted-linux-static': 'GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go -C tools/gomad3 list -deps -test -tags test_dep '+packages+' && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go -C tools/gomad3 vet -tags test_dep '+packages,
  'accepted-validate': 'make -C tools/gomad3 validate',
  'accepted-lint-fast': 'make lint-code-fast GOLANGCI_LINT_BASE_REV=ca6fd855868fac364b69cb31394c87ad2912e623 GOLANGCI_LINT_FIX=false GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype ALL_TEST_TAGS=test_dep',
  'accepted-lint-unfiltered': 'cd tools/gomad3 && /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --fix=false --build-tags=test_dep --timeout=10m --config=../../.github/.golangci.yml --path-mode=abs --max-issues-per-linter=0 --max-same-issues=0 ./upgrade/... ./cmd/gomadtool ./deterministicio ./internal/compatibilitypack/...',
  'accepted-version-lint': 'cd tools/gomad3 && /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --fix=false --build-tags=test_dep --timeout=10m --config=../../.github/.golangci.yml --path-mode=abs --max-issues-per-linter=0 --max-same-issues=0 ./toolchain/version',
  'accepted-release-counterfactual': "go -C tools/gomad3 test -tags test_dep -count=1 -json -overlay="+process.cwd()+"/"+out+"/release-counterfactual-overlay.json -run '^TestRegenerationLockReleaseComposition$' ./upgrade/adapterregen"
};
const name=process.argv[2];
if(!commands[name]) throw Error('unknown gate '+name);
const expected=name==='accepted-lint-unfiltered'||name==='accepted-release-counterfactual'?'1':name==='accepted-lint-fast'?'any':'0';
const result=spawnSync(process.execPath,[out+'/run.mjs',name,commands[name],expected],{stdio:'inherit'});
process.exit(result.status??1);
