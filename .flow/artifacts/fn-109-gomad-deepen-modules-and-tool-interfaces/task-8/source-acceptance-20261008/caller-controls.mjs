import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
const out=path.dirname(new URL(import.meta.url).pathname);
const base=JSON.parse(fs.readFileSync(path.join(out,'source-controls.json')));
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const inputs=[],overlays=[],changes=[];
function replaceOne(text,before,after){if(text.split(before).length!==2)throw Error('not exactly one replacement: '+before);return text.replace(before,after);}
function overlay(name,module,file,transform){
 const source=path.join(module,file),original=fs.existsSync(source)?fs.readFileSync(source,'utf8'):'';
 const text=transform(original),replacement=path.join(base.scratch,name+'-caller-'+file.replaceAll('/','-'));
 fs.writeFileSync(replacement,text);
 inputs.push({path:replacement,sha256:hash(text)});
 changes.push({name,file,source_sha256:hash(original),overlay_sha256:hash(text),original_lines:original.split('\n').length,overlay_lines:text.split('\n').length});
 const diff=spawnSync('diff',['-u','--label',name+'/'+file,'--label',name+'/'+file+' (SOURCE only)',source,replacement],{encoding:'utf8'});
 if(original)fs.writeFileSync(path.join(out,name+'-'+file.replaceAll('/','-')+'.diff'),diff.stdout);
 return [source,replacement];
}
for(const [name,module] of [['original',base.original],['current',base.current]]){
 const Replace={...JSON.parse(fs.readFileSync(base.overlays[name==='original'?0:1])).Replace};
 const add=(file,transform)=>{const [source,replacement]=overlay(name,module,file,transform);Replace[source]=replacement;};
 if(name==='original'){
  for(const [file,expected] of [['runner/internal/execution/bootstrap_unix.go','fd907dba3cbd3c75371b63c08225f12fceffdab288572c1f146f520f1324fb53'],['runner/internal/execution/launch_plan_unix.go','b4ecb04475268ad39a5f08eb24bdb85788e5b79ba3353f30c055e8ff661a02c1']])add(file,s=>{if(hash(s)!==expected)throw Error('immutable runner source mismatch');return replaceOne(s,'syscall.Dup2','sourceUnreachableDup2');});
  add('runner/internal/execution/source_control.go',()=>`package execution\nfunc sourceUnreachableDup2(int,int)error{panic("SOURCE forbidden runner descriptor operation")}\n`);
  add('cmd/gomad/internal/cli/analyze.go',s=>{
   s=replaceOne(s,'func prepareAnalysisTarget(ctx context.Context, spec target.Spec) (target.Spec, []deterministicio.Adapter, func() error, error) {','func prepareAnalysisTarget(ctx context.Context, spec target.Spec) (target.Spec, []deterministicio.Adapter, func() error, error) {\n\treturn prepareAnalysisTargetWith(ctx, spec, os.RemoveAll)\n}\n\nfunc prepareAnalysisTargetWith(ctx context.Context, spec target.Spec, remove func(string) error) (target.Spec, []deterministicio.Adapter, func() error, error) {');
   return replaceOne(s,'cleanup := func() error { return os.RemoveAll(root) }','cleanup := func() error { return remove(root) }');
  });
  add('qualification/analysis/prepared_review.go',s=>{
   s=replaceOne(s,'root          string','root          string\n\tremove        func(string) error');
   s=replaceOne(s,'func PrepareCapabilityReview(ctx context.Context, spec target.Spec) (_ PreparedCapabilityReview, retErr error) {','func PrepareCapabilityReview(ctx context.Context, spec target.Spec) (PreparedCapabilityReview, error) {\n\treturn prepareCapabilityReviewWith(ctx, spec, os.RemoveAll)\n}\n\nfunc prepareCapabilityReviewWith(ctx context.Context, spec target.Spec, remove func(string) error) (_ PreparedCapabilityReview, retErr error) {');
   s=replaceOne(s,'retErr = errors.Join(retErr, os.RemoveAll(root))','retErr = errors.Join(retErr, remove(root))');
   s=replaceOne(s,'Adapters: deterministicio.SelectedAdapters(adapters), root: root,','Adapters: deterministicio.SelectedAdapters(adapters), root: root, remove: remove,');
   return replaceOne(s,'return os.RemoveAll(root)','return prepared.remove(root)');
  });
  add('qualification/analysis/source_control.go',()=>`package analysis\nimport ("context"; "go.temporal.io/server/tools/gomad3/target")\nfunc PrepareCapabilityReviewSourceControl(ctx context.Context,spec target.Spec,remove func(string)error)(PreparedCapabilityReview,error){return prepareCapabilityReviewWith(ctx,spec,remove)}\n`);
  add('cmd/gomad/internal/cli/source_control.go',()=>`package cli\nimport ("context"; capabilityanalysis "go.temporal.io/server/tools/gomad3/qualification/analysis"; "go.temporal.io/server/tools/gomad3/target"; "go.temporal.io/server/tools/gomad3/deterministicio")\nfunc sourceConsumerDependencies(remove func(string)error) analyzeDependencies {return analyzeDependencies{prepare:func(ctx context.Context,spec target.Spec)(target.Spec,[]deterministicio.Adapter,func()error,error){return prepareAnalysisTargetWith(ctx,spec,remove)},analyze:capabilityanalysis.Analyze}}\n`);
 }else{
  add('internal/preparation/source_control.go',()=>`package preparation\nimport ("context"; "go.temporal.io/server/tools/gomad3/target"; "go.temporal.io/server/tools/gomad3/deterministicio")\nfunc InspectSourceControl(ctx context.Context,spec target.Spec,adapters func(context.Context,target.Spec)(target.Spec,[]deterministicio.BuildAdapter,error),review func(context.Context,target.Spec)(target.CapabilityReview,error),remove func(string)error)(Inspection,error){return inspectWith(ctx,spec,inspectionServices{adapters:adapters,review:review,remove:remove})}\n`);
  add('qualification/analysis/prepared_review.go',s=>{
   s=replaceOne(s,'func PrepareCapabilityReview(ctx context.Context, spec target.Spec) (PreparedCapabilityReview, error) {','func PrepareCapabilityReview(ctx context.Context, spec target.Spec) (PreparedCapabilityReview, error) {\n\treturn prepareCapabilityReviewWith(ctx, spec, preparation.Inspect)\n}\n\nfunc prepareCapabilityReviewWith(ctx context.Context, spec target.Spec, inspect func(context.Context, target.Spec) (preparation.Inspection, error)) (PreparedCapabilityReview, error) {');
   return replaceOne(s,'inspected, err := preparation.Inspect(ctx, spec)','inspected, err := inspect(ctx, spec)');
  });
  add('qualification/analysis/source_control.go',()=>`package analysis\nimport ("context"; "go.temporal.io/server/tools/gomad3/target"; "go.temporal.io/server/tools/gomad3/deterministicio"; "go.temporal.io/server/tools/gomad3/internal/preparation")\nfunc PrepareCapabilityReviewSourceControl(ctx context.Context,spec target.Spec,remove func(string)error)(PreparedCapabilityReview,error){return prepareCapabilityReviewWith(ctx,spec,func(ctx context.Context,spec target.Spec)(preparation.Inspection,error){return preparation.InspectSourceControl(ctx,spec,deterministicio.Default().PrepareTargetBuildAdapters,target.ReviewCapabilities,remove)})}\n`);
  add('cmd/gomad/internal/cli/source_control.go',()=>`package cli\nimport ("context"; capabilityanalysis "go.temporal.io/server/tools/gomad3/qualification/analysis"; "go.temporal.io/server/tools/gomad3/target"; "go.temporal.io/server/tools/gomad3/deterministicio"; "go.temporal.io/server/tools/gomad3/internal/preparation")\nfunc sourceConsumerDependencies(remove func(string)error) analyzeDependencies {return analyzeDependencies{inspect:func(ctx context.Context,spec target.Spec)(preparation.Inspection,error){return preparation.InspectSourceControl(ctx,spec,deterministicio.Default().PrepareTargetBuildAdapters,target.ReviewCapabilities,remove)},build:capabilityanalysis.Build}}\n`);
 }
 add('cmd/gomadtool/compatibility_pack.go',s=>{
  s=replaceOne(s,'capabilityanalysis "go.temporal.io/server/tools/gomad3/qualification/analysis"','capabilityanalysis "go.temporal.io/server/tools/gomad3/qualification/analysis"\n\t"go.temporal.io/server/tools/gomad3/target"');
  const dependency='prepare func(context.Context, target.Spec) (capabilityanalysis.PreparedCapabilityReview, error)';
  const discover='func runCompatibilityPackDiscover(arguments []string, stdout, stderr io.Writer) int {';
  s=replaceOne(s,discover,discover+'\n\treturn runCompatibilityPackDiscoverWith(arguments, stdout, stderr, capabilityanalysis.PrepareCapabilityReview)\n}\n\nfunc runCompatibilityPackDiscoverWith(arguments []string, stdout, stderr io.Writer, '+dependency+') int {');
  if(name==='original'){
   const qualify='func runCompatibilityPackQualify(arguments []string, stdout, stderr io.Writer) int {';
   s=replaceOne(s,qualify,qualify+'\n\treturn runCompatibilityPackQualifyWith(arguments, stdout, stderr, capabilityanalysis.PrepareCapabilityReview)\n}\n\nfunc runCompatibilityPackQualifyWith(arguments []string, stdout, stderr io.Writer, '+dependency+') int {');
  }else{
   const qualify='func qualifyCompatibilityPackRequest(resolvedRoot, resolvedRequest, workingDirectory string, stdout, stderr io.Writer) (status int, outputErr error) {';
   s=replaceOne(s,qualify,qualify+'\n\treturn qualifyCompatibilityPackRequestWith(resolvedRoot, resolvedRequest, workingDirectory, stdout, stderr, capabilityanalysis.PrepareCapabilityReview)\n}\n\nfunc qualifyCompatibilityPackRequestWith(resolvedRoot, resolvedRequest, workingDirectory string, stdout, stderr io.Writer, '+dependency+') (status int, outputErr error) {');
  }
  if(s.split('prepared, err := capabilityanalysis.PrepareCapabilityReview(').length!==3)throw Error('prepare call count');
  return s.replaceAll('prepared, err := capabilityanalysis.PrepareCapabilityReview(','prepared, err := prepare(');
 });
 add('cmd/gomadtool/source_control.go',()=>name==='original'?`package main\nimport ("context"; "io"; capabilityanalysis "go.temporal.io/server/tools/gomad3/qualification/analysis"; "go.temporal.io/server/tools/gomad3/target")\nfunc sourceQualify(root,request,working string,stdout,stderr io.Writer,prepare func(context.Context,target.Spec)(capabilityanalysis.PreparedCapabilityReview,error))int{return runCompatibilityPackQualifyWith([]string{"--root="+root,"--request="+request,"--working-dir="+working},stdout,stderr,prepare)}\n`:`package main\nimport ("context"; "io"; capabilityanalysis "go.temporal.io/server/tools/gomad3/qualification/analysis"; "go.temporal.io/server/tools/gomad3/target")\nfunc sourceQualify(root,request,working string,stdout,stderr io.Writer,prepare func(context.Context,target.Spec)(capabilityanalysis.PreparedCapabilityReview,error))int{status,outputErr:=qualifyCompatibilityPackRequestWith(root,request,working,stdout,stderr,prepare);if status!=0{return status};if outputErr!=nil{return 3};return 0}\n`);
 for(const [artifact,file] of [['analyze_consumer_test.go','cmd/gomad/internal/cli/source_consumer_test.go'],['review_consumer_test.go','qualification/analysis/source_consumer_test.go'],['compatibility_consumer_test.go','cmd/gomadtool/source_consumer_test.go']]){if(fs.existsSync(path.join(out,artifact)))add(file,()=>fs.readFileSync(path.join(out,artifact),'utf8'));}
 const destination=path.join(base.scratch,name+'-caller-overlay.json');fs.writeFileSync(destination,JSON.stringify({Replace},null,2)+'\n');overlays.push(destination);
}
fs.writeFileSync(path.join(out,'caller-controls.json'),JSON.stringify({...base,overlays,inputs:[...base.inputs,...inputs],changes,production:false},null,2)+'\n');
console.log(JSON.stringify({overlays,changes}));
