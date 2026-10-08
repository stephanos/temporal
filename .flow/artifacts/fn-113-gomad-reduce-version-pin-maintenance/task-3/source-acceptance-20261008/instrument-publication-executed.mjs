import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
const out=path.dirname(new URL(import.meta.url).pathname),repo=process.cwd(),logical=path.join(repo,'tools/gomad3/upgrade/upgrade_test.go');
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const original=fs.readFileSync(logical,'utf8'), start=original.indexOf('func TestRunReportsPublicationFailureAndKeepsPriorDossier('), end=original.indexOf('func assertPublishedDossier(',start), body=original.slice(start,end);
const insertions=[
 ['\t\t\t\tif err := os.Chmod(directory, 0o500); err != nil {','\t\t\t\tobserveFn1133Publication(t, "after prior creation", output)\n',false],
 ['\t\t\t\tt.Cleanup(func() {','\t\t\t\tobserveFn1133Publication(t, "after chmod", output)\n',false],
 ['\t\t\terr := Run(context.Background(), Spec{','\t\t\tobserveFn1133Publication(t, "immediately before Run", kept)\n',false],
 ['\t\t\tif err == nil || !strings.Contains(err.Error(), "upgrade dossier")','\t\t\tobserveFn1133Publication(t, "immediately after Run", kept)\n',false],
 ['\t\t\tcontents, readErr := os.ReadFile(kept)','\t\t\tobserveFn1133Publication(t, "before read assertion", kept)\n',false],
 ];
let instrumented=body;
for(const [needle,addition] of insertions){if(instrumented.split(needle).length!==2)throw Error('ambiguous instrumentation anchor '+needle);instrumented=instrumented.replace(needle,addition+needle);}
let reconstructed=instrumented;
for(const [,addition] of insertions){if(reconstructed.split(addition).length!==2)throw Error('ambiguous instrumentation removal');reconstructed=reconstructed.replace(addition,'');}
if(reconstructed!==body)throw Error('original assertions/body changed');
const helper=`func observeFn1133Publication(t *testing.T, stage, file string) {
	t.Helper()
	contents, readErr := os.ReadFile(file)
	info, statErr := os.Stat(file)
	directory := filepath.Dir(file)
	parent, parentErr := os.Stat(directory)
	names, listErr := os.ReadDir(directory)
	entries := []string{}
	for _, name := range names { entries = append(entries, name.Name()) }
	mode, parentMode := "unavailable", "unavailable"
	if info != nil { mode = fmt.Sprintf("%o", info.Mode().Perm()) }
	if parent != nil { parentMode = fmt.Sprintf("%o", parent.Mode().Perm()) }
	errno := func(err error) uint64 { var value syscall.Errno; if errors.As(err, &value) { return uint64(value) }; return 0 }
	observation := map[string]any{"stage": stage, "path": file, "read_error": fmt.Sprint(readErr), "read_errno": errno(readErr), "stat_error": fmt.Sprint(statErr), "stat_errno": errno(statErr), "parent_error": fmt.Sprint(parentErr), "parent_errno": errno(parentErr), "list_error": fmt.Sprint(listErr), "list_errno": errno(listErr), "mode": mode, "parent_mode": parentMode, "entries": entries, "sha256": fmt.Sprintf("%x", sha256.Sum256(contents)), "bytes": len(contents)}
	encoded, err := json.Marshal(observation)
	if err != nil { t.Fatal(err) }
	t.Logf("FN1133_PUBLICATION_OBSERVATION %s", encoded)
}
`;
const executing=original.slice(0,start)+instrumented+original.slice(end)+'\n'+helper, actual=path.join(out,'upgrade-publication-instrumented_test.go'), overlay=path.join(out,'upgrade-publication-diagnostic-overlay.json');
fs.writeFileSync(actual,executing);
fs.writeFileSync(overlay,JSON.stringify({Replace:{[logical]:actual}},null,2)+'\n');
const baseline=JSON.parse(fs.readFileSync(path.join(out,'baseline-quick-source.json'))).find(x=>x.path==='tools/gomad3/upgrade/upgrade_test.go');
if(baseline.sha256!==hash(original))throw Error('original test source changed');
const manifest=JSON.parse(fs.readFileSync(path.join(out,'final-cli-packs-source.json'))), effective=manifest.map(x=>x.path==='tools/gomad3/upgrade/upgrade_test.go'?{...x,sha256:hash(executing)}:x);
const binding={diagnostic_only:true,original_file:logical,original_sha256:hash(original),original_body_sha256:hash(body),original_body_reconstructed_exactly:true,inserted_observations:insertions.map(([,x])=>x.trim()),all_original_assertions_fixture_calls_and_cleanup_unchanged:true,actual_file:actual,executing_sha256:hash(executing),overlay,overlay_sha256:hash(fs.readFileSync(overlay)),effective_source_tree_sha256:hash(JSON.stringify(effective)),instrumenter_sha256:hash(fs.readFileSync(new URL(import.meta.url)))};
fs.writeFileSync(path.join(out,'upgrade-publication-overlay-binding.json'),JSON.stringify(binding,null,2)+'\n');
const command='go -C tools/gomad3 test -tags test_dep -count=1 -json -p=1 -overlay='+overlay+' -run="^TestRunReportsPublicationFailureAndKeepsPriorDossier$" ./upgrade';
const result=spawnSync('node',[path.join(out,'run.mjs'),'upgrade-publication-instrumented-diagnostic',command,'any'],{stdio:'inherit'});
if(result.status!==0)throw Error('diagnostic harness failed');
if(hash(fs.readFileSync(logical))!==binding.original_sha256||hash(fs.readFileSync(actual))!==binding.executing_sha256||hash(fs.readFileSync(overlay))!==binding.overlay_sha256)throw Error('diagnostic source mutated');
binding.original_overlay_and_effective_source_unchanged=true;
binding.receipt='upgrade-publication-instrumented-diagnostic-receipt.json';
fs.writeFileSync(path.join(out,'upgrade-publication-overlay-binding.json'),JSON.stringify(binding,null,2)+'\n');
