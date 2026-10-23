import copy
import hashlib
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver
import receiving


class RuntimePreparationBinding(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.config = dict(root=str(self.root), repo=str(self.root), files={}, identity=dict(host='a'*64, helper='b'*64), binary_sha256='c'*64)
        def pin(path, raw):
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(raw)
            self.config['files'][str(path)] = receiving.digest(raw)
            return str(path)
        compiler = pin(self.root/'compiler', b'compiler')
        self.source = pin(self.root/'tools/gomad/internal/backends/wasm/testdata/runtime_probes/main.go', b'package main\n')
        self.choice = hashlib.sha256(b'gomad-choice-implementation-source-v3')
        for name in ('choice/schema/choicewire.json','choice/schema/choicewire.go.tmpl','choice/schema/choicewire_runtime.go.tmpl','choice/schema/choicewire_test.go.tmpl','runtimesource/gomad_semantics.go.txt','choice/trace.go','choice/tape.go'):
            raw = name.encode()
            pin(self.root/'tools/gomad/internal/model'/name, raw)
            self.choice.update(len(raw).to_bytes(8,'big')); self.choice.update(raw)
        self.policy = dict(Profile='gomad.wasi-cooperative/v1', Args=['guest','literal $arg'], Environment=['PWD=/workspace','TZ=UTC','ONLY=captured'],
            WorkingDirectory='/workspace', WritableDirectories=['/workspace','/tmp'], EntropyKey=[1,2,3]+[0]*29,
            Clock=dict(EpochNanos=946684800000000000,ReadStepNanos=1000),
            Limits=dict(OutputBytes=1<<20,TranscriptBytes=8<<20,Calls=10000,Descriptors=64,Files=1000,FilesystemBytes=8<<20,PendingEvents=4096))
        model, namespace, environment, _ = receiving.inspection_policy_projection(self.policy)
        inventory = {compiler:self.config['files'][compiler]}
        self.installation = dict(schema='gomad.rust-wasm-installation/v1',profile='gomad.wasi-cooperative/v1',fuel=4_000_000_000,memory_bytes=256<<20,
            host_sha256='a'*64,helper_sha256='b'*64,model=model,namespace=namespace,storage=None,network=None,
            compiler_path=compiler,compiler_sha256=self.config['files'][compiler],runtime_implementation_sha256='d'*64,go_version='go1.27.1',
            admission=dict(environment=environment,arguments=self.policy['Args'],host_executable_sha256='c'*64,guest_configuration=self.policy,
                configuration_sha256=receiving.digest(receiving.canonical(self.policy)),compiler_inputs=inventory,inputs_sha256=receiving.digest(receiving.canonical(inventory))))
        path = self.root/'installation.json'
        path.write_bytes(receiving.canonical(self.installation))
        self.config['installations'] = {'runtime':str(path)}
        self.module = b'\0asm\x01\0\0\0\x05\x03\x01\0\x01'
        configuration = receiving.runtime_configuration(self.installation, None)
        inputs = dict(schema='gomad.rust-wasm-build-inputs/v1',configuration=configuration,network=None,scenario=None,
            compiler_path=compiler,compiler_sha256=self.config['files'][compiler],module_metadata=receiving.module_metadata(self.module),
            compiler=inventory,host='a'*64,helper='b'*64,sources={self.source:self.config['files'][self.source]},
            arguments=['literal $arg'],directory=str(self.root/'tools/gomad'),kind='go-run',source='./internal/backends/wasm/testdata/runtime_probes',
            go_version='go1.27.1',runtime_implementation='d'*64,runtime='e'*64,tags=[],soft_assertion_report_only=False,
            build_flags=['-mod=readonly','-trimpath','-buildvcs=false','-pgo=off'],environment=dict(GOOS='wasip1',GOARCH='wasm',CGO_ENABLED='0',GOEXPERIMENT='nogreenteagc',GOENV='off',GOTOOLCHAIN='local',GOWORK='off',
                GOPROXY='off',GOSUMDB='off',GOPRIVATE='',GONOPROXY='none',GOVCS='*:off',GOFLAGS='',GOCACHEPROG='',GOWASM='',GODEBUG=''))
        self.provenance = dict(schema='gomad.rust-wasm-provenance/v1',inputs=inputs,configuration=configuration,network=None,scenario=None,
            module_metadata=inputs['module_metadata'],module_sha256=receiving.digest(self.module))
        self.reseal()

    def reseal(self):
        p = self.provenance
        p['inputs']['configuration'] = dict(p['configuration'],runtime_sha256=None)
        p['build_key'] = receiving.digest(receiving.canonical(p['inputs']))
        implementation = hashlib.sha256(b'gomad-choice-implementation-v2\0'+self.choice.digest()+bytes.fromhex(p['build_key'])).hexdigest()
        p['configuration']['runtime_sha256'] = implementation
        p['choice_identity'] = dict(ImplementationSHA256=list(bytes.fromhex(implementation)),ToolchainBuildKey=p['build_key'],TargetSHA256=list(bytes.fromhex(p['module_sha256'])),GOOS='wasip1',GOARCH='wasm')
        self.record = dict(identity=dict(target=p['module_sha256'],toolchain=p['build_key'],configuration=receiving.digest(receiving.canonical(p['configuration'])),
            implementation=receiving.digest(receiving.canonical(self.config['identity'])),platform=receiving.runtime_platform()),
            seed=3*1000003,profile=p['configuration']['profile'],arguments=p['configuration']['arguments'],environment=p['configuration']['environment'],limits=p['configuration']['limits'])

    def validate(self):
        receiving.Retained(self.config).runtime_preparation(self.installation,self.provenance,self.record,self.module,3)

    def test_genuine_preparation_and_all_seed_bounds(self):
        self.validate()
        for index in (-1,32,True):
            with self.subTest(index=index), self.assertRaises(driver.Refusal):
                receiving.Retained(self.config).runtime_preparation(self.installation,self.provenance,self.record,self.module,index)

    def test_self_consistent_foreign_source_compiler_or_configuration_refuses(self):
        original = copy.deepcopy(self.provenance)
        edits = [lambda p:p['inputs'].update(go_version='go1.27.2'),lambda p:p['inputs'].update(arguments=[]),
            lambda p:p['inputs']['sources'].update({self.source:'f'*64}),lambda p:p['inputs'].update(compiler={}),
            lambda p:p['inputs']['environment'].update(GOEXPERIMENT=''),lambda p:p['configuration']['limits'].update(wall_millis=60001),
            lambda p:p['configuration']['environment'].update(ONLY='foreign'),lambda p:p['inputs'].update(runtime_implementation='f'*64),
            lambda p:p['inputs'].update(soft_assertion_report_only=True)]
        for edit in edits:
            self.provenance = copy.deepcopy(original)
            edit(self.provenance); self.reseal()
            with self.subTest(edit=edit), self.assertRaises(driver.Refusal): self.validate()

    def test_record_role_seed_module_and_preimage_tamper_refuse(self):
        original = copy.deepcopy(self.record)
        for field,value in [('seed',0),('arguments',[]),('environment',{}),('limits',{}),('profile','gomad.wasi-stock-exploratory/v1')]:
            self.record = dict(original,**{field:value})
            with self.subTest(field=field), self.assertRaises(driver.Refusal): self.validate()
        for field in original['identity']:
            self.record = copy.deepcopy(original)
            self.record['identity'][field] = 'f'*64
            with self.subTest(identity=field), self.assertRaises(driver.Refusal): self.validate()
        self.record = original
        with self.assertRaises(driver.Refusal):
            receiving.Retained(self.config).runtime_preparation(self.installation,self.provenance,self.record,self.module+b'\0',3)

    def test_all_native_forced_build_environment_controls_are_bound(self):
        original = copy.deepcopy(self.provenance)
        for key in original['inputs']['environment']:
            self.provenance = copy.deepcopy(original)
            self.provenance['inputs']['environment'][key] = 'foreign'
            self.reseal()
            with self.subTest(changed=key), self.assertRaises(driver.Refusal): self.validate()
            self.provenance = copy.deepcopy(original)
            del self.provenance['inputs']['environment'][key]
            self.reseal()
            with self.subTest(missing=key), self.assertRaises(driver.Refusal): self.validate()
