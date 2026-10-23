"""Independent projection of retained native compiler command observations."""
import hashlib
import json
from pathlib import Path

from driver import Refusal, Unexecuted, read, require

PREFIX = 'target/preparation-inputs'
STATIC = (
    'internal/model/choice/schema/choicewire.json', 'internal/model/choice/schema/choicewire.go.tmpl',
    'internal/model/choice/schema/choicewire_runtime.go.tmpl', 'internal/model/choice/schema/choicewire_test.go.tmpl',
    'internal/model/choice/trace.go', 'internal/model/choice/tape.go', 'internal/backends/wasm/toolchain/overlay.go',
    'internal/backends/wasm/toolchain/runtime_wasm.go.txt', 'internal/backends/wasm/toolchain/runtime_inspect_wasm.go.txt',
    'internal/backends/wasm/toolchain/net_stream_wasm.go.txt', 'internal/backends/wasm/toolchain/net_stream_status_generated.go.txt',
    'internal/backends/wasm/toolchain/cmd/overlay/main.go')
SOURCE_FIELDS = ('GoFiles','EmbedFiles','TestGoFiles','XTestGoFiles','TestEmbedFiles','XTestEmbedFiles','SFiles','SysoFiles','HFiles')


def unique(pairs):
    value = {}
    for key,item in pairs:
        require(key not in value,'duplicate native preparation JSON key')
        value[key] = item
    return value


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def canonical(value):
    return json.dumps(value,sort_keys=True,separators=(',',':'),ensure_ascii=False,allow_nan=False).encode()


def documents(raw):
    text = raw.decode('utf-8')
    decoder = json.JSONDecoder(object_pairs_hook=unique)
    index, values = 0, []
    while index < len(text):
        while index < len(text) and text[index] in ' \r\n\t': index += 1
        if index == len(text): break
        value,index = decoder.raw_decode(text,index)
        require(isinstance(value,dict),'native package stream requires objects')
        values.append(value)
    require(values,'empty native package stream')
    return values


class Inventory:
    def __init__(self, config):
        self.config = config
        self.files = config['files']

    def pin(self, path):
        path = Path(path)
        require(path.is_absolute() and '..' not in path.parts and str(path) in self.files,
                'native preparation inventory member is not frozen: '+str(path))
        return self.files[str(path)]

    def file_bytes(self, path):
        path = Path(path)
        expected = self.pin(path)
        require(path.is_file() and not any(parent.is_symlink() for parent in (path,*path.parents)),
                'native preparation source is missing/symlinked')
        require(path.stat().st_size <= 512<<20,'native preparation source exceeds original bound')
        raw = path.read_bytes()
        require(digest(raw) == expected,'native preparation source bytes changed')
        return raw

    def observed(self, artifact, phase):
        artifact = Path(artifact)
        path = artifact/PREFIX/phase/'metadata.json'
        if not path.is_file(): raise Unexecuted('actual native preparation observations missing; fresh preparation required')
        require(not any(parent.is_symlink() for parent in (path,*path.parents)) and path.stat().st_nlink == 1 and path.stat().st_size <= 16<<20,'native observation metadata is aliased or oversized')
        raw = path.read_bytes()
        value = json.loads(raw,object_pairs_hook=unique)
        require(raw == canonical(value) and set(value) == {'schema','phase','commands','snapshots','blobs'}
                and value['schema'] == 'gomad.native-preparation-observations/v1' and value['phase'] == phase,
                'wrong/noncanonical native preparation observation')
        require(type(value['blobs']) is dict and len(value['blobs']) <= 32,'native observation raw blob table exceeds bound')
        blobs = {}
        for identity,entry in value['blobs'].items():
            require(set(entry) == {'path','bytes','sha256'} and entry['path'] == f'{PREFIX}/{phase}/{identity}.bin'
                    and identity == entry['sha256'] and len(identity) == 64 and all(char in '0123456789abcdef' for char in identity)
                    and type(entry['bytes']) is int and 0 <= entry['bytes'] <= 128<<20,'invalid native preparation blob metadata')
            blob_path = artifact/entry['path']
            require(blob_path.is_file() and not any(parent.is_symlink() for parent in (blob_path,*blob_path.parents)) and blob_path.stat().st_nlink == 1 and blob_path.stat().st_size == entry['bytes'],'native observation raw blob is missing, aliased or oversized')
            body = blob_path.read_bytes()
            require(len(body) == entry['bytes'] and digest(body) == identity,'native preparation raw command stream changed')
            blobs[identity] = body
        commands, pids, owners, used = {},set(),set(),set()
        fields = {'name','argv','cwd','environment','owner_pid','pid','start_ns','end_ns','exit','output_limit','stdout','stderr'}
        for command in value['commands']:
            require(set(command) == fields and command['name'] not in commands and type(command['pid']) is int and command['pid'] > 0
                    and type(command['owner_pid']) is int and command['owner_pid'] > 0 and command['pid'] != command['owner_pid']
                    and command['pid'] not in pids and type(command['exit']) is int and command['exit'] == 0
                    and type(command['start_ns']) is int and type(command['end_ns']) is int
                    and 0 < command['start_ns'] <= command['end_ns']
                    and command['end_ns']-command['start_ns'] <= 600_000_000_000,
                    'native preparation command missing/duplicate/failed/outside original deadline')
            require(command['argv'] and command['argv'][0] in self.files
                    and isinstance(command['environment'],dict) and all(isinstance(key,str) and isinstance(item,str) for key,item in command['environment'].items()),
                    'native preparation actual compiler/environment is not frozen')
            for stream in ('stdout','stderr'):
                require(command[stream] in blobs,'native preparation command stream absent')
                used.add(command[stream])
            commands[command['name']] = command
            pids.add(command['pid']); owners.add(command['owner_pid'])
        require(len(owners) == 1,'native preparation observations borrowed another owner')
        for identity in value['snapshots'].values():
            require(identity in blobs,'native preparation snapshot absent')
            used.add(identity)
        require(used == set(blobs),'native preparation unreferenced/missing raw blobs')
        return commands,{key:blobs[identity] for key,identity in value['snapshots'].items()},blobs

    def compiler(self, environment, snapshot):
        value = json.loads(environment,object_pairs_hook=unique)
        require(set(value) == {'GOTOOLDIR','GOROOT'},'native compiler env output shape changed')
        root,tools = Path(value['GOROOT']),Path(value['GOTOOLDIR'])
        require(root.is_absolute() and tools.is_absolute(),'native compiler distribution is not absolute')
        include = root/'pkg/include'
        headers = {str(path) for path in include.iterdir() if path.suffix == '.h'}
        require(headers,'native compiler assembler headers missing')
        expected = {str(root/'VERSION'),*(str(tools/name) for name in ('asm','compile','link')),*headers}
        require(json.loads(snapshot,object_pairs_hook=unique) == sorted(expected),'native complete compiler directory inventory changed')
        return {path:self.pin(path) for path in sorted(expected)},root

    def sources(self, raw, root, replacements):
        paths = set()
        def add(path):
            path = str(path)
            self.pin(path)
            paths.add(path)
        for package in documents(raw):
            directory = Path(package['Dir'])
            require(directory.is_absolute(),'native selected package directory is not absolute')
            module = package.get('Module')
            if module is not None:
                require(module.get('Replace') is None,'native ambient module replacement unsupported')
                if module.get('GoMod'):
                    mod = Path(module['GoMod']); add(mod)
                    if mod.with_name('go.sum').exists(): add(mod.with_name('go.sum'))
            for field in SOURCE_FIELDS:
                values = package.get(field,[])
                require(isinstance(values,list) and all(isinstance(value,str) for value in values),'invalid native package filename list')
                for filename in values:
                    require(not Path(filename).is_absolute() and '..' not in Path(filename).parts,'unsafe native package filename')
                    path = directory/filename
                    if str(path) in replacements:
                        replacement = replacements[str(path)]
                        require(replacement,'native runtime overlay removed selected input')
                        add(replacement)
                        if not path.exists(): continue
                    add(path)
            for filename in package.get('SFiles',[]):
                self.assembly(directory/filename,directory,(directory/filename).parent,root/'pkg/include',paths,set())
        require(paths,'empty native complete selected inventory')
        return {path:self.pin(path) for path in sorted(paths)}

    def assembly(self, path, package, source, include, paths, visited):
        if path in visited: return
        visited.add(path)
        text = self.file_bytes(path).decode('utf-8')
        in_comment = False
        for line in text.splitlines():
            code, index, quote = '',0,None
            while index < len(line):
                char = line[index]
                next_char = line[index+1:index+2]
                if in_comment:
                    if char == '*' and next_char == '/': in_comment=False; index+=2
                    else: index+=1
                    continue
                if quote:
                    code += char; index+=1
                    if char == '\\' and quote != '`' and index < len(line): code+=line[index]; index+=1
                    elif char == quote: quote=None
                    continue
                if char in '\"\'`': quote=char; code+=char; index+=1; continue
                if char == '/' and next_char == '*': in_comment=True; code+=' '; index+=2; continue
                if char == '/' and next_char == '/': break
                code+=char; index+=1
            code = code.lstrip()
            if not code.startswith('#'): continue
            directive = code[1:].lstrip()
            if not directive.startswith('include') or len(directive)>7 and not (directive[7].isspace() or directive[7]=='"'): continue
            name = json.loads(directive[7:].strip())
            require(isinstance(name,str),'unsupported native assembler include')
            candidates = [package/name,source/name]+([] if name=='go_asm.h' else [include/name])
            selected = next((candidate for candidate in candidates if candidate.exists()),None)
            if selected is None:
                require(name=='go_asm.h','native assembler include missing')
                continue
            paths.add(str(selected)); self.pin(selected)
            self.assembly(selected,package,source,include,paths,visited)
        require(not in_comment,'unterminated native assembler comment')

    def project(self, artifact, installation, inputs, owner_pid=None):
        compile_rows,compile_snapshots,compile_blobs = self.observed(artifact,'compile')
        admission_rows,admission_snapshots,admission_blobs = self.observed(artifact,'admission')
        if owner_pid is not None:
            require(all(command['owner_pid'] == owner_pid for command in admission_rows.values()),'native admission observations borrowed another actual producer')
        cooperative = installation['profile'] == 'gomad.wasi-cooperative/v1'
        expected_names = {'compiler-version','compiler-environment','selected-package','selected-source-closure','compile-module'}
        if cooperative: expected_names |= {'compiler-module-cache','runtime-overlay'}
        require(set(compile_rows) == expected_names and set(admission_rows) == {'selected-source-closure','compiler-environment'}
                and set(compile_snapshots) == ({'compiler-base-paths','overlay-replacements'} if cooperative else {'compiler-base-paths'})
                and set(admission_snapshots) == {'compiler-base-paths'},'native preparation exact phase/command/snapshot coverage changed')
        def stream(rows,blobs,name,kind='stdout'):
            return blobs[rows[name][kind]]
        compiler,root = self.compiler(stream(compile_rows,compile_blobs,'compiler-environment'),compile_snapshots['compiler-base-paths'])
        after_compiler,after_root = self.compiler(stream(admission_rows,admission_blobs,'compiler-environment'),admission_snapshots['compiler-base-paths'])
        require(compiler == after_compiler and root == after_root,'native compiler pre/post complete inventory changed')
        executable = inputs['compiler_path']
        require(executable == installation['compiler_path'] and self.pin(executable) == installation['compiler_sha256'] == inputs['compiler_sha256'],
                'native actual compiler differs from admitted executable')
        require(inputs['kind'] in ('go-test','go-run') and inputs['go_version'] == installation['go_version']
                and stream(compile_rows,compile_blobs,'compiler-version').startswith(('go version '+installation['go_version']+' ').encode()),
                'native compiler version or target kind changed')
        forced = dict(GOENV='off',GOPROXY='off',GOSUMDB='off',GOPRIVATE='',GONOPROXY='none',GOVCS='*:off',GOFLAGS='',GOTOOLCHAIN='local',GOWORK='off',
                      GOOS='wasip1',GOARCH='wasm',CGO_ENABLED='0',GOEXPERIMENT='nogreenteagc',GOCACHEPROG='',GOWASM='',GODEBUG='')
        require(inputs['environment'] == forced,'native original forced compiler controls changed')
        flags = ['-mod=readonly','-pgo=off']
        actual_flags = inputs['build_flags']
        require(actual_flags in (['-mod=readonly','-trimpath','-buildvcs=false','-pgo=off'],
                                  ['-mod=readonly','-trimpath','-buildvcs=false','-pgo=off','-p=1']), 'native compiler flags changed')
        if '-p=1' in actual_flags: flags += ['-p=1']
        replacements = {}
        module = Path(installation.get('overlay_module_root') or str(Path(installation['runtime_root']).parents[2])) if cooperative else None
        overlay_path = None
        if cooperative:
            generated = json.loads(stream(compile_rows,compile_blobs,'runtime-overlay'),object_pairs_hook=unique)
            require(set(generated) == {'path','identity','implementation_sha256'} and generated['implementation_sha256'] == installation['runtime_implementation_sha256']
                    and generated['identity'] == inputs['runtime'],'native runtime overlay output identity changed')
            overlay_path = str(Path(installation['cache_root'])/'runtime-overlay'/generated['identity']/'overlay.json')
            require(generated['path'] == overlay_path and digest(compile_snapshots['overlay-replacements']) == installation['admission']['compiler_inputs'].get(overlay_path),
                    'native raw overlay is not frozen/admitted')
            overlay = json.loads(compile_snapshots['overlay-replacements'],object_pairs_hook=unique)
            require(set(overlay) == {'Replace'} and isinstance(overlay['Replace'],dict),'native runtime overlay shape changed')
            replacements = overlay['Replace']
            for original,replacement in replacements.items():
                require(isinstance(replacement,str) and replacement,'native runtime overlay removed source')
                for path in (original,replacement):
                    if Path(path).is_file(): compiler[path] = self.pin(path)
            for name in STATIC:
                path = str(module/name); compiler[path] = self.pin(path)
            for path in (overlay_path,str(Path(installation['runtime_root'])/'gomad_semantics.go.txt'),str(Path(installation['runtime_root'])/'gomad_choicewire_generated.go.txt')):
                compiler[path] = self.pin(path)
            flags += ['-overlay',overlay_path]
            cache = Path(stream(compile_rows,compile_blobs,'compiler-module-cache').decode('utf-8').strip())
            require(cache.is_absolute() and not root.is_relative_to(cache),'native cooperative compiler is inside module cache')
        tags = inputs['tags']
        require(isinstance(tags,list) and tags == sorted(set(tags)),'native build tags not canonical')
        if tags: flags += ['-tags',','.join(tags)]
        selected_list = [executable,'list',*flags,'-deps','-json']+(['-test'] if inputs['kind']=='go-test' else [])+[inputs['source']]
        expected = {
            'compiler-version':[executable,'version'],
            'compiler-environment':[executable,'env','-json','GOTOOLDIR','GOROOT'],
            'selected-package':[executable,'list',*flags,inputs['source']],
            'selected-source-closure':selected_list,
        }
        if cooperative:
            expected['compiler-module-cache'] = [executable,'env','GOMODCACHE']
            expected['runtime-overlay'] = [executable,'run','-mod=readonly','./internal/backends/wasm/toolchain/cmd/overlay',str(root),installation['runtime_root'],str(Path(installation['cache_root'])/'runtime-overlay')]
        caps = {'compiler-version':64<<10,'compiler-environment':64<<10,'compiler-module-cache':64<<10,'runtime-overlay':4<<20,'selected-package':1<<20,'selected-source-closure':128<<20,'compile-module':4<<20}
        for rows,blobs,phase in ((compile_rows,compile_blobs,'compile'),(admission_rows,admission_blobs,'admission')):
            for name,command in rows.items():
                require(type(command['output_limit']) is int and command['output_limit'] == caps[name]
                        and all(len(blobs[command[kind]]) <= caps[name] for kind in ('stdout','stderr')),'native original command output cap changed')
                cwd = str(module) if name=='runtime-overlay' else inputs['directory']
                require(command['cwd'] == cwd,'native compiler command source directory changed')
                if name!='compile-module': require(command['argv'] == expected[name],'native actual preparation command argv changed')
                environment = command['environment']
                allowed = set(forced)|{'PATH','TMPDIR','HOME','GOPATH','GOMODCACHE','XDG_CACHE_HOME','GOMAXPROCS','GOTMPDIR','GOROOT','GOCACHE'}
                require(set(environment) <= allowed,'native compiler inherited unrelated host environment')
                if name!='compile-module': require('GOCACHE' not in environment,'native compiler used an unbound inherited object cache')
                has_root = name=='runtime-overlay' or cooperative and (name in ('selected-package','selected-source-closure','compile-module') or phase=='admission')
                if not has_root: require('GOROOT' not in environment,'native compiler inherited a foreign GOROOT')
                if name=='runtime-overlay':
                    require(all(environment.get(key) == item for key,item in forced.items() if key not in ('GOOS','GOARCH','GOEXPERIMENT'))
                            and all(key not in environment for key in ('GOOS','GOARCH','GOEXPERIMENT')) and environment.get('GOROOT') == str(root),
                            'native overlay host compiler controls changed')
                else:
                    require(all(environment.get(key) == item for key,item in forced.items()),'native preparation compiler controls changed')
                    if cooperative and (name in ('selected-package','selected-source-closure','compile-module') or phase=='admission'):
                        require(environment.get('GOROOT') == str(root),'native preparation GOROOT changed')
        command = compile_rows['compile-module']
        argv = command['argv']
        prefix = [executable,'test','-c'] if inputs['kind']=='go-test' else [executable,'build']
        prefix += [*flags,'-trimpath','-buildvcs=false','-o']
        require(len(argv) == len(prefix)+2 and argv[:-2] == prefix and argv[-1] == inputs['source'],'native actual compile command shape changed')
        output = Path(argv[-2])
        temporary = output.parent
        stem = '.rust-preparation-'+str(command['owner_pid'])+'-'
        require(output.name == 'module.wasm' and temporary.parent == Path(installation['cache_root'])
                and temporary.name.startswith(stem) and temporary.name[len(stem):].isascii() and temporary.name[len(stem):].isdigit(),
                'native compile output is outside actual owner private preparation')
        key_fields = ('compiler','compiler_path','compiler_sha256','go_version','sources','runtime','runtime_implementation','kind','source','directory','tags','build_flags','environment')
        compile_inputs = dict(schema='gomad.native-compilation-inputs/v1',profile=installation['profile'],**{key:inputs[key] for key in key_fields})
        key = digest(canonical(compile_inputs))
        require(command['environment'].get('GOCACHE') == str(Path(installation['cache_root'])/'compilation-v1'/key/'objects'),
                'native compiler did not use its own original per-entry object cache')
        before = self.sources(stream(compile_rows,compile_blobs,'selected-source-closure'),root,replacements)
        after = self.sources(stream(admission_rows,admission_blobs,'selected-source-closure'),root,replacements)
        require(before == after == inputs['sources'] and compiler == inputs['compiler'],
                'native complete source/compiler membership differs from independent raw projection')
        return compiler,before
