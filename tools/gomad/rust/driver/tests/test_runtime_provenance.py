import copy
import hashlib
import json
from pathlib import Path
import struct
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver
import receiving
import runtime_gate
import adapters
from test_runtime_gate import original_configuration


SOURCES = ('choice/schema/choicewire.json', 'choice/schema/choicewire.go.tmpl',
    'choice/schema/choicewire_runtime.go.tmpl', 'choice/schema/choicewire_test.go.tmpl',
    'runtimesource/gomad_semantics.go.txt', 'choice/trace.go', 'choice/tape.go')


class RuntimeProvenance(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.artifact = self.root/'artifact'
        self.artifact.mkdir()
        self.compiler = self.root/'stock/bin/go'
        self.source = self.root/'tools/gomad/internal/backends/wasm/testdata/runtime_probes/main.go'
        self.overlay = self.root/'cache/runtime-overlay'/('c'*64)/'runtime.go'
        self.files = {}
        for path in (self.compiler, self.source, self.overlay,
                     *(self.root/'tools/gomad/internal/model'/name for name in SOURCES)):
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(str(path.relative_to(self.root)).encode())
            self.files[str(path)] = driver.sha(path)
        self.identity = dict(host='a'*64, helper='b'*64)
        installed = dict(compiler_path=str(self.compiler), compiler_sha256=self.files[str(self.compiler)],
            runtime_implementation_sha256='d'*64, cache_root=str(self.root/'cache'),
            admission=dict(compiler_inputs={str(path):self.files[str(path)] for path in (self.compiler,self.overlay)}))
        self.installation = self.root/'installation.json'
        self.installation.write_bytes(receiving.canonical(installed))
        self.config = dict(root=str(self.root), repo=str(self.root), identity=self.identity, files=self.files,
            installations=dict(runtime=str(self.installation)), stages={'runtime-0':dict(installation='runtime')})
        self.configuration = original_configuration()
        self.module = b'\0asm\1\0\0\0\5\3\1\0\1'
        self.metadata = receiving.module_metadata(self.module)
        self.inputs = dict(schema='gomad.rust-wasm-build-inputs/v1', arguments=['literal $arg'],
            build_flags=['-mod=readonly','-trimpath','-buildvcs=false','-pgo=off'],
            compiler={str(path):self.files[str(path)] for path in (self.compiler,self.overlay)},
            compiler_path=str(self.compiler), compiler_sha256=self.files[str(self.compiler)],
            directory=str(self.source.parent), environment=dict(CGO_ENABLED='0',GOARCH='wasm',GOCACHEPROG='',GODEBUG='',
                GOENV='off',GOEXPERIMENT='nogreenteagc',GOFLAGS='',GONOPROXY='none',GOOS='wasip1',GOPRIVATE='',
                GOPROXY='off',GOSUMDB='off',GOTOOLCHAIN='local',GOVCS='*:off',GOWASM='',GOWORK='off'),
            go_version='go1.27.1', helper=self.identity['helper'],host=self.identity['host'],kind='go-run',
            module_metadata=self.metadata,network=None,runtime='c'*64,runtime_implementation='d'*64,
            scenario=None,soft_assertion_report_only=False,source='.',sources={str(self.source):self.files[str(self.source)]},tags=[])
        self.reader = receiving.Retained(self.config)
        self.raw_choice = self.choice()
        self.seal()

    @staticmethod
    def choice():
        result = []
        for ordinal,kind in enumerate((1,2,3)):
            raw = bytearray(96)
            struct.pack_into('>QBBHIIIQ',raw,0,ordinal,kind,2 if kind==3 else 1,1 if kind==3 else 0,2,0,1 if kind==3 else 0,1)
            if kind==3:
                struct.pack_into('>Q',raw,32,1)
            else:
                raw[32:64]=b'a'*32;raw[64:]=b'b'*32
            result.append(raw)
        return b''.join(result)

    def seal(self, edit=None):
        inputs,configuration = copy.deepcopy(self.inputs),copy.deepcopy(self.configuration)
        if edit:
            edit(inputs,configuration)
        inputs['configuration'] = dict(configuration,runtime_sha256=None)
        key = receiving.digest(receiving.canonical(inputs))
        source = hashlib.sha256(b'gomad-choice-implementation-source-v3')
        for name in SOURCES:
            raw = (self.root/'tools/gomad/internal/model'/name).read_bytes()
            source.update(len(raw).to_bytes(8,'big'));source.update(raw)
        implementation = hashlib.sha256(b'gomad-choice-implementation-v2\0'+source.digest()+bytes.fromhex(key)).digest()
        configuration['runtime_sha256'] = implementation.hex()
        target = receiving.digest(self.module)
        choice = dict(TargetSHA256=list(bytes.fromhex(target)),ToolchainBuildKey=key,GOOS='wasip1',GOARCH='wasm',
            ImplementationSHA256=list(implementation))
        provenance = dict(schema='gomad.rust-wasm-provenance/v1',inputs=inputs,configuration=configuration,
            build_key=key,module_sha256=target,module_metadata=inputs['module_metadata'],choice_identity=choice,network=None,scenario=None)
        replay = dict(schema='gomad.replay-inputs/v1',configuration=configuration,toolchain_sha256=key,
            host_sha256=inputs['host'],helper_sha256=inputs['helper'],runtime_implementation_sha256=inputs['runtime_implementation'])
        details = dict(choices=dict(identity=choice,implementation_sha256=self.identity['helper'],terminal_state='complete',
            records=3,trace=dict(file='evidence/choice',limit=1<<20,bytes=len(self.raw_choice),sha256=receiving.digest(self.raw_choice))))
        payloads = {'target/module.wasm':self.module,'target/provenance.json':receiving.canonical(provenance),
            'evidence/replay.json':receiving.canonical(replay),'evidence/inspection.json':receiving.canonical(details),
            'evidence/choice':self.raw_choice}
        finish=(1).to_bytes(4,'big')+(7).to_bytes(4,'big')+bytes(8)
        diagnostic=[]
        for ordinal in range(3):
            raw=bytearray(96);struct.pack_into('>Q',raw,0,ordinal);struct.pack_into('>Q',raw,16,1)
            struct.pack_into('>I',raw,24,1)
            for offset in range(40,96,8):struct.pack_into('>Q',raw,offset,1)
            diagnostic.append(raw)
        payloads.update({'evidence/runtime-terminal':finish,'evidence/diagnostic':b''.join(diagnostic),
            'output/stdout':b'entropy \nmap \nrunq \ntimer callbacks \ntimer parking\nnote negative park wake\ncollector automatic finalizer cleanup 1 1\ncanary \n'})
        entries = {name:dict(bytes=len(raw),sha256=receiving.digest(raw),module=name=='target/module.wasm') for name,raw in payloads.items()}
        record = dict(schema='gomad.execution-record/v2',seed=0,outcome=dict(termination='exit',exit_code=0),
            identity=dict(target=target,toolchain=key,implementation=receiving.digest(receiving.canonical(self.identity)),
                          configuration=receiving.digest(receiving.canonical(configuration))),
            arguments=configuration['arguments'],environment=configuration['environment'],limits=configuration['limits'],
            profile=configuration['profile'],streams={name:dict(bytes=row['bytes'],sha256=row['sha256']) for name,row in entries.items()})
        for name,raw in payloads.items():
            path=self.artifact/name;path.parent.mkdir(parents=True,exist_ok=True);path.write_bytes(raw)
        (self.artifact/'record.json').write_bytes(receiving.canonical(record))
        (self.artifact/'storage.json').write_bytes(receiving.canonical({'format':'gomad.artifact-storage/v2'}))
        manifest=dict(schema='gomad.artifact-manifest/v2',record=record,record_sha256=receiving.digest(receiving.canonical(record)),files=entries)
        (self.artifact/'manifest.json').write_bytes(receiving.canonical(manifest))
        summary=runtime_gate.choice_summary(self.raw_choice,finish,driver.require)
        summary.pop('peak_goroutines')
        self.inspected=dict(artifact=dict(record=record,files=entries,choices=summary))

    def test_complete_independent_preparation_binding(self):
        self.reader.runtime_artifact(self.artifact,self.inspected,0)

    def test_resealed_foreign_preparation_failure_matrix(self):
        changes = {
            'compiler':lambda inputs,config:inputs['compiler'].__setitem__(str(self.overlay),'f'*64),
            'compiler-path':lambda inputs,config:inputs.__setitem__('compiler_path',str(self.overlay)),
            'runtime-overlay':lambda inputs,config:inputs.__setitem__('runtime','e'*64),
            'runtime-implementation':lambda inputs,config:inputs.__setitem__('runtime_implementation','e'*64),
            'selected-source':lambda inputs,config:inputs['sources'].__setitem__(str(self.source),'f'*64),
            'source-selection':lambda inputs,config:inputs.__setitem__('source','foreign'),
            'argv':lambda inputs,config:inputs.__setitem__('arguments',[]),
            'flags':lambda inputs,config:inputs.__setitem__('build_flags',[]),
            'collector':lambda inputs,config:inputs['environment'].__setitem__('GOEXPERIMENT','greenteagc'),
            'compiler-environment':lambda inputs,config:inputs['environment'].__setitem__('GOOS','linux'),
            'module-metadata':lambda inputs,config:inputs.__setitem__('module_metadata',dict(imports=[],initial_memory_pages=2)),
            'clock':lambda inputs,config:config['model'].__setitem__('read_step_nanos',0),
            'limits':lambda inputs,config:config['limits'].__setitem__('fuel',1<<50),
            'profile':lambda inputs,config:config.__setitem__('profile','gomad.wasi-stock-exploratory/v1'),
        }
        for name,change in changes.items():
            with self.subTest(name=name):
                self.seal(change)
                self.reader.artifact(self.artifact)
                with self.assertRaises(driver.Refusal):
                    self.reader.runtime_artifact(self.artifact,self.inspected,0)

    def test_choice_summary_is_decoded_from_raw_records(self):
        finish = (1).to_bytes(4,'big')+(7).to_bytes(4,'big')+bytes(8)
        expected=dict(records=3,branching_records=2,runnable=1,select_poll=1,select_result=1,
            terminal_state='complete',peak_goroutines=7)
        self.assertEqual(runtime_gate.choice_summary(self.raw_choice,finish,driver.require),expected)
        for offset,value in ((0,9),(8,0),(9,7),(13,0)):
            changed=bytearray(self.raw_choice);changed[offset]=value
            if changed==self.raw_choice:
                continue
            with self.subTest(offset=offset),self.assertRaises(driver.Refusal):
                runtime_gate.choice_summary(bytes(changed),finish,driver.require)
        changed=bytearray(self.raw_choice);changed[32:64]=bytes(32)
        for raw in (bytes(changed),self.raw_choice[:-1]):
            with self.assertRaises(driver.Refusal):
                runtime_gate.choice_summary(raw,finish,driver.require)

    def test_retained_inspect_summary_mutation_refuses_without_changing_raw_artifact(self):
        linked=dict(host_sha256=self.identity['host'],helper_sha256=self.identity['helper'],runtime_implementation_sha256='d'*64)
        api=dict(require=driver.require,read=driver.read,sha=driver.sha,REPO=self.root,
            public=lambda config:[],runtime_artifact=self.reader.runtime_artifact,runtime_witness=adapters.runtime_witness,
            payload=lambda artifact,inspection,name:(Path(artifact)/name).read_bytes())
        api['command']=lambda *args,**kwargs:json.dumps(self.inspected)
        runtime_gate.observation(self.config,self.root,'inspect',self.artifact,0,linked,api)
        original=copy.deepcopy(self.inspected)
        for field in ('records','branching_records','runnable','select_poll','select_result','terminal_state'):
            changed=copy.deepcopy(original)
            changed['artifact']['choices'][field]='overflow' if field=='terminal_state' else 131
            api['command']=lambda *args,**kwargs:json.dumps(changed)
            with self.subTest(field=field),self.assertRaises(driver.Refusal):
                runtime_gate.observation(self.config,self.root,'inspect',self.artifact,0,linked,api)


if __name__=='__main__':
    unittest.main()
