import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
const repo='/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out=path.join(repo,'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-7/source-acceptance-20261008');
const scratch=fs.mkdtempSync('/Users/stephan/Workspace/skunkworks/.gomad-fn1097-source-');
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const replacements={},inputs=[];
for(const file of ['deterministicio/profile.go','deterministicio/adapter_registry.go','target/target.go']){
 const source=path.join(repo,'tools/gomad3',file),bytes=fs.readFileSync(source),original=bytes.toString();
 let text=original.replaceAll('runtime.GOOS','"linux"').replaceAll('runtime.GOARCH','"amd64"');
 if(!text.includes('runtime.'))text=text.replace('\n\t"runtime"','');
 const target=path.join(scratch,file.replaceAll('/','-'));
 fs.writeFileSync(target,text);replacements[source]=target;
 inputs.push({path:source,sha256:hash(bytes),overlay:target,overlay_sha256:hash(text),change:'compile-time linux/amd64 source inputs only; original validators and policy unchanged'});
}
const overlay=path.join(scratch,'overlay.json');fs.writeFileSync(overlay,JSON.stringify({Replace:replacements},null,2)+'\n');
fs.writeFileSync(path.join(out,'source-controls.json'),JSON.stringify({actual_host:'linux/arm64',source_profile:'linux/amd64',scope:'SOURCE controls, cross-built targets never executed, no native qualification',scratch,overlay,overlay_sha256:hash(fs.readFileSync(overlay)),inputs},null,2)+'\n');
console.log(overlay);
