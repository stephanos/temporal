import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
const out=path.dirname(new URL(import.meta.url).pathname),hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const controls=JSON.parse(fs.readFileSync(path.join(out,'caller-controls.json'))),inputs=[],overlays=[];
const fixture=fs.mkdtempSync('/Users/stephan/Workspace/skunkworks/.gomad-fn1098-classification-');
for(const [side,module]of [['original',controls.original],['current',controls.current]]){
 const Replace=JSON.parse(fs.readFileSync(controls.overlays[side==='original'?0:1])).Replace;
 const template=path.join(out,'classification_consumer_test.go'),driver=path.join(out,'classification-driver.mjs');
 const body=fs.readFileSync(template,'utf8')+'\nconst sourceClassificationDriver = '+JSON.stringify(driver)+'\nconst sourceClassificationFixture = '+JSON.stringify(fixture)+'\n';
 const replacement=path.join(controls.scratch,side+'-classification-test-'+hash(body)+'.go');
 fs.writeFileSync(replacement,body);
 Replace[path.join(module,'cmd/gomad/internal/cli/source_classification_test.go')]=replacement;
 const helper=path.join(controls.scratch,side+'-classification-helper.go');
 const common='package cli\nimport("context";"go.temporal.io/server/tools/gomad3/target";';
 const helperBody=side==='current'?common+'"go.temporal.io/server/tools/gomad3/internal/preparation")\nfunc sourceClassificationDependencies(remove func(string)error,observe func(error)) analyzeDependencies {d:=sourceConsumerDependencies(remove);inspect:=d.inspect;d.inspect=func(ctx context.Context,spec target.Spec)(preparation.Inspection,error){v,e:=inspect(ctx,spec);observe(e);return v,e};return d}\n':common+'"go.temporal.io/server/tools/gomad3/deterministicio";analysis "go.temporal.io/server/tools/gomad3/qualification/analysis")\nfunc sourceClassificationDependencies(remove func(string)error,observe func(error)) analyzeDependencies {d:=sourceConsumerDependencies(remove);prepare,analyze:=d.prepare,d.analyze;d.prepare=func(ctx context.Context,spec target.Spec)(target.Spec,[]deterministicio.Adapter,func()error,error){v,a,c,e:=prepare(ctx,spec);observe(e);return v,a,c,e};d.analyze=func(ctx context.Context,spec analysis.Spec)(analysis.Report,error){v,e:=analyze(ctx,spec);observe(e);return v,e};return d}\n';
 fs.writeFileSync(helper,helperBody);Replace[path.join(module,'cmd/gomad/internal/cli/source_classification_helpers_test.go')]=helper;
 const overlay=path.join(controls.scratch,side+'-classification-overlay-'+hash(body)+'.json');fs.writeFileSync(overlay,JSON.stringify({Replace},null,2)+'\n');overlays.push(overlay);
 for(const file of [template,driver,replacement,helper,path.join(module,'deterministicio/testdata/sprig/go.mod')])inputs.push({path:file,sha256:hash(fs.readFileSync(file))});
}
const bytes=JSON.stringify({...controls,overlays,inputs:[...controls.inputs,...inputs],scope:'additive actual full CLI invalid-sum and scripted malformed linked extraction; SOURCE only, no launch/native compiler'},null,2)+'\n';
fs.writeFileSync(path.join(out,'classification-controls-'+hash(bytes)+'.json'),bytes,{flag:'wx'});fs.writeFileSync(path.join(out,'classification-controls.json'),bytes);
