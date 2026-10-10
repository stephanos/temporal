import sys
sys.dont_write_bytecode = True
import run as r

module = r.ROOT/'tools/gomad3'
r.run('preservation-comparison',['/usr/bin/python3',str(r.PACKET/'check.py')],r.ROOT)
r.run('gofmt',[str(r.GO_ROOT/'bin/gofmt'),'-d',r.SELECTED],r.ROOT)
r.run('host-vet',[r.GO,'vet','-tags','test_dep','./runner','./internal/gomadtool/conformance','./runner/internal/execution'],module)
r.run('errortype',[r.GO,'vet','-tags','test_dep','-vettool='+str(r.TOOLS/'errortype'),'-style-check=false','./runner'],module)
r.run('runner-lint',[str(r.TOOLS/'golangci-lint-v2.13.0'),'run','--config',str(r.ROOT/'.github/.golangci.yml'),'--build-tags','test_dep','--timeout','10m','--fix=false','./runner'],module)
r.run('fast-lint',['/usr/bin/make','lint-code-fast',*r.LINT_ARGS],r.ROOT)
for goos,goarch in (('linux','amd64'),('darwin','arm64')):
    r.run('vet-'+goos+'-'+goarch,[r.GO,'vet','-tags','test_dep','./runner','./internal/gomadtool/conformance','./runner/internal/execution'],module,{'GOOS':goos,'GOARCH':goarch})
r.run('architecture',[r.GO,'test','-tags','test_dep','-count=1','-json','-timeout','300s','-run','^Test(PackageArchitecture|PureModulesHaveNoHostEffects|PublicPackagesDoNotExportTypeAliases|PublicPackagesDoNotExportForwardingAliases|RunnerRequestsCompileInExternalModule|RunnerExecutionInjectionIsPrivate|ArchitectureEffectFixtures|ArchitecturePublicSignatureFixtures|ExactModuleEdges)$','.'],module)
r.run('validate',['/usr/bin/make','-C','tools/gomad3','validate','SHELL=/bin/sh'],r.ROOT,{'GOCACHE':str(module/'.toolchain/generator-cache')})
