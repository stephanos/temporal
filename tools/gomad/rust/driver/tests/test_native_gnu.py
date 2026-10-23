"""Synthetic GNU probe and ELF closure evidence; never execute a compiler or probe."""
import json
import os
from pathlib import Path
import struct
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
import native_gnu as gate
from driver import Refusal,sha


def elf(needed=(),search=(),interpreter=None):
    strings=b'\0';entries=[]
    for name in needed:
        entries.append((1,len(strings)));strings+=name.encode()+b'\0'
    for name in search:
        entries.append((29,len(strings)));strings+=name.encode()+b'\0'
    ident=b'\x7fELF\x02\x01\x01'+b'\0'*9
    if not entries and interpreter is None:
        return struct.pack('<16sHHIQQQIHHHHHH',ident,3,183,1,0,0,0,0,64,0,0,0,0,0)
    count=2+(interpreter is not None);dynamic_offset=64+56*count
    string_offset=dynamic_offset+16*(len(entries)+3)
    dynamic=b''.join(struct.pack('<qQ',tag,value) for tag,value in entries+[(5,0x400000+string_offset),(10,len(strings)),(0,0)])
    end=string_offset+len(strings);interp_raw=(interpreter.encode()+b'\0') if interpreter else b''
    headers=[(1,5,0,0x400000,0,end+len(interp_raw),end+len(interp_raw),4096),
             (2,4,dynamic_offset,0x400000+dynamic_offset,0,len(dynamic),len(dynamic),8)]
    if interpreter:headers.append((3,4,end,0x400000+end,0,len(interp_raw),len(interp_raw),1))
    header=struct.pack('<16sHHIQQQIHHHHHH',ident,3,183,1,0,64,0,0,64,56,count,0,0,0)
    return header+b''.join(struct.pack('<IIQQQQQQ',*fields) for fields in headers)+dynamic+strings+interp_raw


def fixture(root):
    system=root/'system';work=root/'prefreeze';work.mkdir(mode=0o700)
    def binary(path,raw=None):
        path.parent.mkdir(parents=True,exist_ok=True);path.write_bytes(elf() if raw is None else raw);path.chmod(0o755);return path
    library=binary(system/'usr/lib/libc.so.6')
    loader_elf=binary(system/'usr/lib/ld-aarch64.so.1')
    (system/'lib').symlink_to('usr/lib',target_is_directory=True)
    linker=binary(system/'usr/bin/aarch64-linux-gnu-gcc-12',elf(['libc.so.6'],interpreter=str(system/'lib/ld-aarch64.so.1')))
    loader=binary(system/'usr/sbin/ldconfig',elf(['libc.so.6']))
    git=binary(system/'usr/bin/git',elf(['libc.so.6']))
    version=system/'usr/lib/gcc/aarch64-linux-gnu/12'
    programs={name:str(binary((version if name in ('cc1','cc1plus','collect2','lto-wrapper') else system/'usr/bin')/name,elf(['libc.so.6']))) for name in gate.DELEGATES}
    include=system/'usr/include';include.mkdir();(include/'stdint.h').write_text('frozen native headers')
    (system/'etc/ld.so.conf.d').mkdir(parents=True)
    for name in ('ld.so.cache','ld.so.conf'):(system/'etc'/name).write_text('frozen loader '+name)
    (system/'etc/ld.so.conf.d/native.conf').write_text(str(system/'usr/lib'))
    rust=root/'rust';tools={name:str(binary(rust/'bin'/name,elf(['libc.so.6']))) for name in ('rustc','cargo','rustdoc')}
    tools['python']=str(binary(root/'python',elf(['libc.so.6'])))
    tools_path=root/'static-plan.json';tools_path.write_text(json.dumps(dict(schema='gomad.driver-static-plan/v1',repo=str(root),tools=tools)))
    for name in ('home','tmp','bin'):(work/name).mkdir(mode=0o700)
    output={'version':'aarch64-linux-gnu-gcc-12 (Debian) 12.2.0\n','machine':'aarch64-linux-gnu\n','gcc-version':'12.2.0\n','sysroot':'\n',
        'specs':'*link:\nGNU native defaults\n','search':'programs: ='+str(version)+':'+str(system/'usr/bin')+'\nlibraries: ='+str(system/'usr/lib')+'\n',
        'loader-cache':'1 libs found in cache\n libc.so.6 (libc6,AArch64) => '+str(system/'lib/libc.so.6')+'\n'}
    output.update({name:value+'\n' for name,value in programs.items()})
    for index,(name,argv) in enumerate(gate.probes(str(linker),str(loader)).items()):
        (work/(name+'.stdout')).write_text(output.get(name,''));(work/(name+'.stderr')).write_text('#include <...> search starts here:\n '+str(include)+'\nEnd of search list.\n' if name.startswith('include-') else '')
        row=dict(argv=argv,cwd=str(root),pid=200+index,parent_pid=100,clock='CLOCK_MONOTONIC',started_monotonic_nanos=20+index,reaped_monotonic_nanos=80+index,
                 environment=gate.probe_environment(work),effective_environment=gate.probe_environment(work),environment_mode='exact',exit=0,observation='completed',start_ns=100,end_ns=200)
        (work/(name+'.command.json')).write_text(json.dumps(row));(work/(name+'.process.json')).write_text(json.dumps(dict(row,stage_pid=100,active=False)))
    (work/'producer.json').write_text(json.dumps(dict(pid=100,start_monotonic_nanos=10,end_monotonic_nanos=1000,sources={str(gate.HERE/name):sha(gate.HERE/name) for name in gate.SOURCES})))
    for name,target in dict(cc=str(linker),gcc=str(linker),**{'c++':str(linker),'g++':str(linker),'git':str(git)},**{name:programs[name] for name in gate.BINUTILS}).items():
        (work/'bin'/name).symlink_to(target)
    return system,work,tools_path,str(linker),library


class GNUClosure(unittest.TestCase):
    def setUp(self):
        selected=patch.object(gate,'native_machine',return_value=183);selected.start();self.addCleanup(selected.stop)

    def test_elf_needed_search_and_interpreter_from_actual_bytes(self):
        result=gate.elf(elf(['libc.so.6'],['$ORIGIN/../lib'],'/lib/ld.so'))
        self.assertEqual(result,dict(machine=183,needed=['libc.so.6'],search=['$ORIGIN/../lib'],interpreter=['/lib/ld.so']))
        self.assertIsNone(gate.elf(b'not an ELF'))
        with self.assertRaises(Refusal):gate.elf(b'\x7fELF'+b'\0'*64)

    def test_complete_fixed_probe_delegate_header_loader_and_alias_closure(self):
        with tempfile.TemporaryDirectory() as temporary:
            system,root,tools,linker,library=fixture(Path(temporary))
            with patch.object(gate,'SYSTEM',system):result=gate.derive(root,tools,linker)
            self.assertIn(str(library),result['files'])
            self.assertIn(str(system/'usr/include/stdint.h'),result['files'])
            self.assertIn(str(system/'lib'),result['aliases'])
            self.assertIn(str(root/'bin/cc'),result['aliases'])
            self.assertNotIn(str(root/'gnu-toolchain.json'),result['files'])
            (root/'gnu-toolchain.json').write_text(json.dumps(result))
            config=dict(identity_harness=dict(toolchain_plan=str(root/'gnu-toolchain.json')))
            with patch.object(gate,'SYSTEM',system):self.assertEqual(gate.inspect(config,tools)[1],result)

    def test_library_delegate_header_or_alias_mutation_cannot_keep_old_plan(self):
        for mutation in ('library','delegate','header','alias','loader','preload','extra-executable'):
            with self.subTest(mutation=mutation),tempfile.TemporaryDirectory() as temporary:
                system,root,tools,linker,library=fixture(Path(temporary))
                with patch.object(gate,'SYSTEM',system):result=gate.derive(root,tools,linker)
                path=root/'gnu-toolchain.json';path.write_text(json.dumps(result));config=dict(identity_harness=dict(toolchain_plan=str(path)))
                if mutation=='library':library.write_bytes(elf()+b'changed library')
                if mutation=='delegate':Path(result['programs']['collect2']).write_bytes(elf()+b'changed delegate')
                if mutation=='header':(system/'usr/include/stdint.h').write_text('changed header')
                if mutation=='alias':(root/'bin/cc').unlink();(root/'bin/cc').symlink_to(result['programs']['as'])
                if mutation=='loader':(system/'etc/ld.so.cache').write_text('changed loader cache')
                if mutation=='preload':(system/'etc/ld.so.preload').write_text('/foreign/preload.so')
                if mutation=='extra-executable':
                    path=root/'bin/pkg-config';path.write_text('foreign executable');path.chmod(0o755)
                with patch.object(gate,'SYSTEM',system),self.assertRaises(Refusal):gate.inspect(config,tools)

    def test_unsupported_family_unresolved_needed_and_foreign_process_refuse(self):
        for mutation in ('family','dependency','process','search'):
            with self.subTest(mutation=mutation),tempfile.TemporaryDirectory() as temporary:
                system,root,tools,linker,library=fixture(Path(temporary))
                if mutation=='family':(root/'version.stdout').write_text('clang 22\n')
                if mutation=='dependency':Path(linker).write_bytes(elf(['foreign.so']))
                if mutation=='process':
                    path=root/'version.command.json';row=json.loads(path.read_text());row['parent_pid']=777;path.write_text(json.dumps(row))
                if mutation=='search':(root/'search.stdout').write_text('programs: ='+str(system/'usr/bin')+'\nlibraries: ='+str(system/'usr/lib/../../foreign')+'\n')
                with patch.object(gate,'SYSTEM',system),self.assertRaises(Refusal):gate.derive(root,tools,linker)


if __name__=='__main__':unittest.main()
