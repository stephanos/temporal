import sys
sys.dont_write_bytecode = True
import run as r

HISTORICAL = r.ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-70/manifest-2a1b63fe97199753cc29e63535c34003238dc9547dad4040aa4228faa4a6bc74'

module = r.ROOT/'tools/gomad3'
commands = [
    ('gofmt',[str(r.GO_ROOT/'bin/gofmt'),'-d',r.SELECTED],r.ROOT,None),
    ('diff-check',['/usr/bin/git','diff','--check'],r.ROOT,None),
    ('host-vet',[r.GO,'vet','-tags','test_dep','./runner','./internal/gomadtool/conformance','./runner/internal/execution'],module,None),
    ('errortype',[r.GO,'vet','-tags','test_dep','-vettool='+str(r.TOOLS/'errortype'),'-style-check=false','./runner'],module,None),
    ('runner-lint',[str(r.TOOLS/'golangci-lint-v2.13.0'),'run','--config',str(r.ROOT/'.github/.golangci.yml'),'--build-tags','test_dep','--timeout','10m','--fix=false','./runner'],module,None),
    ('fast-lint',['/usr/bin/make','lint-code-fast',*r.LINT_ARGS],r.ROOT,None),
]
for goos,goarch in (('linux','amd64'),('darwin','arm64')):
    commands.append(('vet-'+goos+'-'+goarch,[r.GO,'vet','-tags','test_dep','./runner','./internal/gomadtool/conformance','./runner/internal/execution'],module,{'GOOS':goos,'GOARCH':goarch,'CGO_ENABLED':'0'}))
commands += [
    ('architecture',[r.GO,'test','-tags','test_dep','-count=1','-json','-run','^Test(PackageArchitecture|PureModulesHaveNoHostEffects|PublicPackagesDoNotExportTypeAliases|PublicPackagesDoNotExportForwardingAliases|RunnerRequestsCompileInExternalModule|RunnerExecutionInjectionIsPrivate|ArchitectureEffectFixtures|ArchitecturePublicSignatureFixtures|ExactModuleEdges)$','.'],module,None),
    ('validation-domain',['/usr/bin/python3',str(r.PACKET/'check.py'),'domain'],r.ROOT,None),
    ('validate',['/usr/bin/make','-C','tools/gomad3','validate','SHELL=/bin/sh'],r.ROOT,{'GOCACHE':str(module/'.toolchain/generator-cache')}),
]
for name,argv,cwd,extra in commands:
    inputs = [HISTORICAL] if name in ('validation-domain','validate') else []
    result = r.run(name,argv,cwd,extra,inputs)
    assert result == (1 if name == 'runner-lint' else 2 if name == 'fast-lint' else 0), (name,result)
