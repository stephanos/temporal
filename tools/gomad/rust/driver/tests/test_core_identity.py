"""Synthetic frozen sources and native identity mutations; no compilation."""
import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import core_identity as gate
import native_roles
from driver import Refusal, sha


def fixture(root):
    repo=root/'repo';crate=repo/'tools/gomad/rust/wasmhost'
    for name in set(gate.EXTRA)|set(gate.REQUIRED):
        path=(crate/name).resolve();path.parent.mkdir(parents=True,exist_ok=True);path.write_text('frozen '+name)
    config=dict(repo=str(repo),files={str(p):sha(p) for p in root.rglob('*') if p.is_file()})
    context=dict(compiler_utf8='rustc 1.99.1 (synthetic)\n',bindings=dict(PROFILE='release',TARGET='aarch64-unknown-linux-gnu'))
    files=gate.source_inputs(config)
    source=[dict(name=name,path=str(crate/name),bytes=len(raw),sha256=native_roles.hashlib.sha256(raw).hexdigest()) for name,raw in sorted(files.items())]
    common=dict(files=source,bindings=context['bindings'],compiler_utf8=context['compiler_utf8'],required_owners=list(gate.REQUIRED))
    before=gate.implementation(files,context['bindings'],context['compiler_utf8'])
    rows=[dict(schema=gate.SCHEMA,case='identity.compiled-native',inputs=copy.deepcopy(common),observed=dict(compiled=before,recomputed=before))]
    for case,owners in gate.OWNERS.items():
        for owner in owners:
            changed=dict(files,**{owner:files[owner]+b'changed implementation'})
            rows.append(dict(schema=gate.SCHEMA,case=case,inputs=dict(copy.deepcopy(common),owner=owner,mutation_append_utf8='changed implementation'),
                observed=dict(before=before,after=gate.implementation(changed,context['bindings'],context['compiler_utf8']),missing_error='missing implementation owner '+owner)))
    rows.append(dict(schema=gate.SCHEMA,case='identity.empty-closure',inputs=dict(copy.deepcopy(common),files=[]),observed=dict(error='missing implementation owner src/lib.rs')))
    return config,context,rows


def stdout(rows):
    return '\n'.join(gate.MARKER+json.dumps(row) for row in rows)


class IdentityFacts(unittest.TestCase):
    def test_reconstructs_all_eighteen_fixed_cases(self):
        with tempfile.TemporaryDirectory() as temporary:
            config,context,rows=fixture(Path(temporary));facts=gate.observations(config,stdout(rows),context)
            self.assertEqual(facts['case_count'],18)
            self.assertEqual(facts['implementation_sha256'],rows[0]['observed']['compiled'])

    def test_compiler_features_source_bytes_and_missing_owner_refusals(self):
        for mutation in ('compiler','feature','source','compiled','rotation','missing','owner','count','required'):
            with self.subTest(mutation=mutation),tempfile.TemporaryDirectory() as temporary:
                config,context,rows=fixture(Path(temporary))
                if mutation=='compiler':rows[0]['inputs']['compiler_utf8']='caller compiler'
                if mutation=='feature':rows[0]['inputs']['bindings']['CARGO_FEATURE_REFERENCE_HELPER']='1'
                if mutation=='source':rows[0]['inputs']['files'][0]['sha256']='0'*64
                if mutation=='compiled':rows[0]['observed']['compiled']='0'*64
                if mutation=='rotation':rows[1]['observed']['after']=rows[1]['observed']['before']
                if mutation=='missing':rows[1]['observed']['missing_error']='success'
                if mutation=='owner':rows[1]['inputs']['owner']='src/foreign.rs'
                if mutation=='count':rows.pop()
                if mutation=='required':rows[0]['inputs']['required_owners'].pop()
                with self.assertRaises(Refusal):gate.observations(config,stdout(rows),context)

    def test_frozen_closure_rejects_omitted_and_added_source(self):
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);config,context,rows=fixture(root)
            path=root/'repo/tools/gomad/rust/wasmhost/src/new.rs';path.write_text('new owner')
            with self.assertRaises(Refusal):gate.observations(config,stdout(rows),context)
            config['files'][str(path)]=sha(path)
            with self.assertRaises(Refusal):gate.observations(config,stdout(rows),context)


if __name__=='__main__':unittest.main()
