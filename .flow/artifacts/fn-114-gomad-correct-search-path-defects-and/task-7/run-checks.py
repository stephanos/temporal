import hashlib,json,os,pathlib,subprocess,time
root=pathlib.Path(__file__).resolve().parents[4]
artifacts=pathlib.Path(__file__).resolve().parent
native=root / 'tools/gomad3/.toolchain/bin/go'
stock='/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go'
env=os.environ.copy()
for key in ('GOROOT','GOMADSEED','GOMAD3_CHILD_SEED'):env.pop(key,None)
env['GOWORK']='off';env['GOMAD3_STOCK_GO']=stock
env['PATH']=str(pathlib.Path(stock).parent)+os.pathsep+env['PATH']
mode=os.sys.argv[1]
commands={
 'focused':[str(native),'test','-tags','test_dep','-count=1','./runner','./cmd/gomad/internal/cli','-run','TestGuided|TestExcludeAnswered|TestRunGuidesFrom|TestRunResumesGuided|TestMixGuided|TestRunCampaignShard|TestMergeCampaignShards|TestFullyAnswered|TestGuidanceReports|TestResumeGuidance|TestRegressionGuidance|TestGuidanceRegression|TestGuideRegression','-v'],
 'integration-fix':[str(native),'test','-tags','test_dep','-count=1','./runner','./cmd/gomad/internal/cli','-run','TestCoordinatorTransportCoversEveryCampaignSpecField|TestCoordinatorTransportRoundTripsEveryTransportedField|TestCreateCampaignPlan|TestGuided|TestExcludeAnswered|TestRunGuidesFrom|TestRunResumesGuided|TestMixGuided|TestRunCampaignShard|TestMergeCampaignShards|TestFullyAnswered|TestGuidanceReports|TestResumeGuidance|TestRegressionGuidance|TestGuidanceRegression|TestGuideRegression','-v'],
 'review-green':[str(native),'test','-tags','test_dep','-count=1','./runner','./cmd/gomad/internal/cli','-run','TestCoordinatorTransportCoversEveryCampaignSpecField|TestCoordinatorTransportRoundTripsEveryTransportedField|TestCreateCampaignPlan|TestGuided|TestExcludeAnswered|TestRunGuidesFrom|TestRunResumesGuided|TestMixGuided|TestRunCampaignShard|TestMergeCampaignShards|TestFullyAnswered|TestGuidanceReports|TestResumeGuidance|TestRegressionGuidance|TestGuidanceRegression|TestGuideRegression|TestEmptyShardDoesNotReport','-v'],
 'review-red':[str(native),'test','-tags','test_dep','-count=1','./runner','./cmd/gomad/internal/cli','-run','TestFullyAnsweredGuidedPlanExecutesEmptyShardsAndMerges|TestEmptyShardDoesNotReport','-v'],
 'architecture':[str(native),'test','-tags','test_dep','-count=1','.'],
 'vet':[str(native),'vet','-tags','test_dep','./runner/...','./cmd/gomad/internal/cli'],
 'validate':['make','-C','tools/gomad3','validate'],
 'root-lint':['make','lint-code-fast'],
}
commands['review-architecture']=commands['architecture']
commands['review-vet']=commands['vet']
cmd=commands[mode]
if mode in ('validate','root-lint'):env.pop('GOMAD3_STOCK_GO',None)
start=time.monotonic()
with (artifacts/(mode+'.log')).open('wb') as out:
 completed=subprocess.run(cmd,cwd=root if mode in ('validate','root-lint') else root/'tools/gomad3',env=env,stdout=out,stderr=subprocess.STDOUT)
result={'command':cmd,'exit_code':completed.returncode,'elapsed_seconds':round(time.monotonic()-start,3),'log':mode+'.log'}
(artifacts/(mode+'.json')).write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(result))
raise SystemExit(completed.returncode)
