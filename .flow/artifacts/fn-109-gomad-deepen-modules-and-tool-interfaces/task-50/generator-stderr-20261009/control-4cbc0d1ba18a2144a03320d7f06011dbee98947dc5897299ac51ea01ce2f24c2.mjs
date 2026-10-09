import {run, lint, errortype, base, original} from './run.mjs';

run('controls-final', "go -C tools/gomad3 test -json -count=1 -tags test_dep ./cmd/gomadtool -run '^TestRunGeneratorDiagnosticOutputPreservesStatus$'", 0);
run('scoped-analyzer-final', 'cd tools/gomad3 && ' + lint + ' run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./cmd/gomadtool', 1);
run('focus-existing-final', "go -C tools/gomad3 test -json -count=1 -tags test_dep ./cmd/gomadtool -run '^(TestRunGeneratorDiagnosticOutputPreservesStatus|TestRunMaintainerOutput.*|TestRunBuildKey|TestRunPatchValidate|TestRunScriptValidate|TestRunCompatibilityPackRefresh.*|TestCompatibilityPackPaths|TestDiagnosticDiffStatusesAndNoPartialInvalidResult)$'", 0);
run('ordinary-package-final', 'go -C tools/gomad3 test -json -count=1 -tags test_dep ./cmd/gomadtool', 0);
run('validate-final', 'GOFLAGS=-tags=test_dep make -C tools/gomad3 validate', 0);
run('architecture-final', "go -C tools/gomad3 test -json -count=1 -tags test_dep . -run '^(TestPackageArchitecture|TestPublicPackagesDoNotExportTypeAliases|TestPureModulesHaveNoHostEffects|TestHostPackageVet|TestInspectionSourceOwnershipDelegation|TestExactModuleEdges|TestDomainModulesDoNotExportWireFraming|TestRunnerExecutionInjectionIsPrivate)$'", 0);
run('source-darwin-final', 'GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go -C tools/gomad3 list -json -tags test_dep ./cmd/gomadtool', 0);
run('source-linux-final', 'GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go -C tools/gomad3 list -json -tags test_dep ./cmd/gomadtool', 0);
run('vet-final', 'go -C tools/gomad3 vet -tags test_dep ./cmd/gomadtool', 0);
run('errortype-final', 'go -C tools/gomad3 vet -tags test_dep -vettool=' + errortype + ' -style-check=false ./cmd/gomadtool', 0);
run('fast-make-final', 'make lint-code-fast GOLANGCI_LINT_FIX=false GOLANGCI_LINT=' + lint + ' ERRORTYPE=' + errortype + ' GOLANGCI_LINT_BASE_REV=' + base, 0);
run('integrated-original-final', 'make lint-code-gomad3 GOLANGCI_LINT_FIX=false GOLANGCI_LINT=' + lint + ' ERRORTYPE=' + errortype + ' GOLANGCI_LINT_BASE_REV=' + original, 2);
run('format-final', "bash .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-9/source-acceptance-20261008/format-gate.sh gofmt -l tools/gomad3/cmd/gomadtool/qualification_manifest.go tools/gomad3/cmd/gomadtool/protocol.go tools/gomad3/cmd/gomadtool/version.go tools/gomad3/cmd/gomadtool/boundary.go tools/gomad3/cmd/gomadtool/generator_diagnostic_output_test.go && git diff --check", 0);
console.log(JSON.stringify({source: 'frozen', execution_lane: 'all final gate children terminal'}));
