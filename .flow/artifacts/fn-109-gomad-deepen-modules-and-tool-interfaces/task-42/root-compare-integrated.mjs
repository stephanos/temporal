import fs from 'node:fs';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
const root='/Users/stephan/Workspace/skunkworks/gomad/temporal';
const artifact=`${root}/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces`;
const basePath=`${artifact}/task-41/root-integrated-final.log`;
const finalPath='/tmp/fn109-policy-integrated.hwEkegMn/gate.log';
function diagnostics(path) {
  const raw=fs.readFileSync(path,'utf8');
  const lines=raw.split('\n');
  const starts=[];
  for(let i=0;i<lines.length;i++) if(/^\S[^:]*\.go:\d+:\d+: .+ \([^)]+\)$/.test(lines[i])) starts.push(i);
  const blocks=starts.map((start,index)=>{
    let end=index+1<starts.length?starts[index+1]:lines.findIndex((line,i)=>i>start&&/^\d+ issues:/.test(line));
    if(end<0) end=lines.length;
    return {header:lines[start],raw:lines.slice(start,end).join('\n')+'\n'};
  });
  return {raw,blocks,sha256:crypto.createHash('sha256').update(raw).digest('hex')};
}
const base=diagnostics(basePath), final=diagnostics(finalPath);
const removed=base.blocks.filter(block=>!final.blocks.some(candidate=>candidate.header===block.header));
const added=final.blocks.filter(block=>!base.blocks.some(candidate=>candidate.header===block.header));
assert.equal(removed.length,1);
assert.match(removed[0].header,/compatibilitypack\/policy\.go:196:4: missing cases.*FactMalformedLinkname.*FactNoReviewedGoSource.*\(exhaustive\)$/);
assert.equal(added.length,0);
for(const block of final.blocks) {
  const original=base.blocks.find(candidate=>candidate.header===block.header);
  assert.equal(block.raw,original.raw,`changed residual diagnostic: ${block.header}`);
}
assert.equal(base.blocks.length,324);
assert.equal(final.blocks.length,323);
const counts={};
for(const block of final.blocks) { const kind=block.header.match(/\(([^)]+)\)$/)[1]; counts[kind]=(counts[kind]??0)+1; }
assert.deepEqual(counts,{errcheck:252,exhaustive:3,forbidigo:11,gci:1,staticcheck:56});
process.stdout.write(JSON.stringify({baseline:{path:basePath,sha256:base.sha256,count:base.blocks.length},final:{path:finalPath,sha256:final.sha256,count:final.blocks.length},removed,added,unchanged_residual_blocks:final.blocks.length,counts},null,2)+'\n');
