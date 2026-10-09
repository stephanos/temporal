import {run, lint, errortype, base, original} from './run.mjs';

run('tool-provenance', 'go version && go env GOOS GOARCH GOROOT && uname -m && ' + lint + ' version', 0);
run('controls-baseline', "go -C tools/gomad3 test -json -count=1 -tags test_dep ./cmd/gomadtool -run '^TestRunGeneratorDiagnosticOutputPreservesStatus$'", 0);
run('scoped-analyzer-red', 'cd tools/gomad3 && ' + lint + ' run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./cmd/gomadtool', 1);
run('integrated-original-before', 'make lint-code-gomad3 GOLANGCI_LINT_FIX=false GOLANGCI_LINT=' + lint + ' ERRORTYPE=' + errortype + ' GOLANGCI_LINT_BASE_REV=' + original, 2);
console.log(JSON.stringify({baseline: base, production_edits: false, execution_lane: 'terminal baseline driver'}));
