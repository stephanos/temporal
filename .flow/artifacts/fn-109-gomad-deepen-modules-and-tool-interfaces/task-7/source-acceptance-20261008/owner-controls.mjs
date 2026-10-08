import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
const repo='/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out=path.join(repo,'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-7/source-acceptance-20261008');
const current=JSON.parse(fs.readFileSync(path.join(out,'source-controls.json'))),original=JSON.parse(fs.readFileSync(path.join(out,'caller-controls.json')));
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const controls=[];
for(const [side,root,base] of [['current',path.join(repo,'tools/gomad3'),current.overlay],['original',original.original,original.overlay]]){
 const replacements=JSON.parse(fs.readFileSync(base)).Replace;
 for(const name of ['internal/preparation/preparation_test.go','runner/replay_operation_test.go']){
  const source=path.join(root,name),bytes=fs.readFileSync(source);
  let text=bytes.toString();
  if(name.includes('internal/preparation'))text=text.replaceAll('filepath.Join(root, ".toolchain")','os.Getenv("GOMAD3_SOURCE_TOOLCHAIN_ROOT")').replaceAll('filepath.Join(testModuleRoot(t), ".toolchain")','os.Getenv("GOMAD3_SOURCE_TOOLCHAIN_ROOT")');
  else text=text.replace('filepath.Abs(filepath.Join("..", ".toolchain"))','filepath.Abs(os.Getenv("GOMAD3_SOURCE_TOOLCHAIN_ROOT"))');
  if(text===bytes.toString())throw Error('missing driver literal '+source);
  const overlay=path.join(current.scratch,side+'-'+name.replaceAll('/','-'));fs.writeFileSync(overlay,text);replacements[source]=overlay;
  controls.push({side,source,sha256:hash(bytes),overlay,overlay_sha256:hash(text),change:'test-only pinned-driver input; actual original test bodies retained'});
 }
 const overlay=path.join(current.scratch,side+'-owner-overlay.json');fs.writeFileSync(overlay,JSON.stringify({Replace:replacements},null,2)+'\n');
 controls.push({side,overlay,overlay_sha256:hash(fs.readFileSync(overlay))});
}
const source=path.join(repo,'tools/gomad3/internal/preparation/preparation.go'),bytes=fs.readFileSync(source);
const mutation=path.join(current.scratch,'missing-adapter-attachment.go');
const text=bytes.toString().replace('prepared.Adapters = executionAdapters(selectedAdapters)','_ = selectedAdapters\n\tprepared.Adapters = nil');
if(text===bytes.toString())throw Error('missing mutation site');fs.writeFileSync(mutation,text);
const replacements=JSON.parse(fs.readFileSync(current.overlay)).Replace;replacements[source]=mutation;
const overlay=path.join(current.scratch,'missing-adapter-overlay.json');fs.writeFileSync(overlay,JSON.stringify({Replace:replacements},null,2)+'\n');
controls.push({side:'mutation-red',source,sha256:hash(bytes),overlay:mutation,overlay_sha256:hash(text),manifest:overlay,change:'test-only remove attachment; production unchanged'});
fs.writeFileSync(path.join(out,'owner-controls.json'),JSON.stringify(controls,null,2)+'\n');
