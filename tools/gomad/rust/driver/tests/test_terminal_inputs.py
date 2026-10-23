"""No compiler execution: reject unknown or divergent terminal compiler/overlay inputs."""
import copy
import hashlib
import json
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import terminal_inputs
import native_roles
from driver import Refusal, sha


class TerminalInputs(unittest.TestCase):
    def fixture(self, root):
        repo = root/'repo'
        runtime = repo/'tools/gomad/internal/model/runtimesource'
        runtime.mkdir(parents=True)
        (runtime/'semantics.go').write_text('frozen runtime source')
        for name in native_roles.RUNTIME:
            path=repo/'tools/gomad/internal/backends/wasm/toolchain'/name
            path.parent.mkdir(parents=True,exist_ok=True)
            path.write_text('frozen generator '+name)
        compiler=root/'stock/bin/go'
        compiler.parent.mkdir(parents=True)
        compiler.write_text('stock compiler bytes')
        (root/'stock/VERSION').write_text('go1.27.1\n')
        cache=root/'cache'
        replacement=cache/'runtime-overlay/identity/runtime.go'
        replacement.parent.mkdir(parents=True)
        replacement.write_text('admitted generated overlay')
        overlay=replacement.parent/'overlay.json'
        overlay.write_text(json.dumps(dict(Replace={str(root/'stock/src/runtime/new.go'):str(replacement)})))
        inventory={str(path):sha(path) for path in (compiler,root/'stock/VERSION',runtime/'semantics.go',replacement,overlay)}
        config=dict(repo=str(repo),root=str(root/'attempt'),primary=dict(compiler=str(compiler),runtime_root=str(runtime),cache=str(cache)),
                    final_profiles=dict(stock_go=str(compiler)),installations={},identity=dict(host='host',helper='helper'),
                    files={str(path):sha(path) for path in root.rglob('*') if path.is_file()})
        runtime_id=terminal_inputs.runtime_identity(config)
        for name in terminal_inputs.INSTALLATIONS:
            path=root/(name+'.json')
            value=dict(schema='gomad.rust-wasm-installation/v1',profile='gomad.wasi-cooperative/v1',go_version='go1.27.1',
                compiler_path=str(compiler),compiler_sha256=sha(compiler),cache_root=str(cache),runtime_root=str(runtime),
                overlay_module_root=str(repo/'tools/gomad'),runtime_implementation_sha256=runtime_id,host_sha256='host',helper_sha256='helper',
                admission=dict(compiler_inputs=inventory,inputs_sha256=hashlib.sha256(json.dumps(inventory,sort_keys=True,separators=(',',':')).encode()).hexdigest()))
            path.write_text(json.dumps(value))
            config['installations'][name]=str(path)
            config['files'][str(path)]=sha(path)
        return config

    def test_request_uses_only_static_actual_shared_inputs(self):
        with tempfile.TemporaryDirectory() as temporary:
            config=self.fixture(Path(temporary))
            spec=terminal_inputs.preparation_spec(config)
            self.assertEqual(set(spec),{'schema','repo','stock_go','toolchain_cache'})
            self.assertEqual(spec['toolchain_cache'],config['primary']['cache'])
            self.assertNotIn('module_sha256',spec)

    def test_different_or_unsealed_cache_and_compiler_roots_refused(self):
        for field,value in (('cache_root','/foreign-cache'),('runtime_root','/foreign-runtime'),('compiler_path','/foreign-go'),
                            ('overlay_module_root','/foreign-module'),('runtime_implementation_sha256','foreign')):
            with self.subTest(field=field),tempfile.TemporaryDirectory() as temporary:
                config=self.fixture(Path(temporary))
                path=Path(config['installations']['onebox-cooperative'])
                data=json.loads(path.read_text());data[field]=value;path.write_text(json.dumps(data))
                config['files'][str(path)]=sha(path)
                with self.assertRaises(Refusal):terminal_inputs.preparation_spec(config)
        with tempfile.TemporaryDirectory() as temporary:
            config=self.fixture(Path(temporary))
            key=next(path for path in config['files'] if path.endswith('/runtime.go'))
            config['files'].pop(key)
            with self.assertRaises(Refusal):terminal_inputs.preparation_spec(config)

    def test_missing_overlay_and_divergent_closure_are_not_future_claims(self):
        with tempfile.TemporaryDirectory() as temporary:
            config=self.fixture(Path(temporary))
            path=Path(config['installations']['onebox-cooperative'])
            value=json.loads(path.read_text())
            inputs=value['admission']['compiler_inputs']
            inputs.pop(next(name for name in inputs if name.endswith('/overlay.json')))
            value['admission']['inputs_sha256']=hashlib.sha256(json.dumps(inputs,sort_keys=True,separators=(',',':')).encode()).hexdigest()
            path.write_text(json.dumps(value));config['files'][str(path)]=sha(path)
            with self.assertRaisesRegex(Refusal,'exact generated runtime overlay'):
                terminal_inputs.preparation_spec(config)


if __name__=='__main__':unittest.main()
