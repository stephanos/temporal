package main

import (
 "context"
 "debug/buildinfo"
 "encoding/json"
 "fmt"
 "os"
 "path/filepath"
 "runtime"
 "strings"

 "go.temporal.io/server/tools/gomad3/artifact"
 "go.temporal.io/server/tools/gomad3/deterministicio"
 "go.temporal.io/server/tools/gomad3/record"
 "go.temporal.io/server/tools/gomad3/runner"
 "go.temporal.io/server/tools/gomad3/target"
)

func main(){if err:=run();err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}}
func run() error {
 binary,toolchain,root:=os.Args[1],os.Args[2],os.Args[3]
 info,err:=buildinfo.ReadFile(binary);if err!=nil{return err}
 contents,err:=os.ReadFile(binary);if err!=nil{return err}
 identity,err:=target.ReadToolchainIdentity(toolchain);if err!=nil{return err}
 profile:=deterministicio.Default();world,payloads:=record.NoneWorld();exit:=record.Uint64String(2)
 manifest:=record.ExecutionRecord{
  SchemaVersion:record.SchemaVersion,ArtifactKind:record.ArtifactTargetFailure,CreatedAt:"2026-10-02T14:00:00Z",CampaignID:"coverage-evidence",Seed:7,ReplayMode:record.ReplayExact,
  Runner:record.Runner{RecordContract:record.RecordContract,RunnerBuild:"runner",HostOS:runtime.GOOS,HostArch:runtime.GOARCH},
  Toolchain:record.Toolchain{GoVersion:identity.GoVersion,BuildKey:identity.BuildKey,TargetGOOS:identity.TargetGOOS,TargetGOARCH:identity.TargetGOARCH},
  Target:record.Target{Kind:"exec",Source:binary,SHA256:record.HashBytes(contents),Size:record.Uint64String(len(contents)),Argv:[]string{"gomad3-target"},BuildTags:[]string{"test_dep"},Adapters:[]record.TargetAdapter{},Compatibility:[]record.CompatibilityPack{},BuildInfo:target.ProjectBuildInfo(info)},
  IOProfile:record.IOProfile{Name:profile.Name(),ImplementationSHA256:record.SHA256(profile.ImplementationSHA256()),Inventory:string(profile.Inventory()),InventorySHA256:record.SHA256(profile.InventorySHA256()),Transcript:&record.IOTranscript{Schema:"gomad3.io-transcript/v1",File:"io/transcript.bin",SHA256:record.HashBytes(nil)}},
  Environment:[]record.Environment{{Name:"GOMAD3_IO_PROFILE",Value:profile.Name()},{Name:"GOMADSEED",Value:"7"},{Name:"TZ",Value:"UTC"}},
  Limits:record.Limits{ExecutionTimeoutNanos:1000000000,OverallTimeoutNanos:2000000000,OutputBytes:64,WorldTransitionBytes:64,IOTranscriptBytes:64<<20},
  World:world,Outcome:record.Outcome{Domain:"target",Reason:"nonzero_exit",Termination:"exit",ExitCode:&exit},
  Streams:record.Streams{Stdout:record.Stream{FullSHA256:record.HashBytes(nil)},Stderr:record.Stream{FullSHA256:record.HashBytes(nil)}},
  Host:record.Host{StartedAt:"2026-10-02T14:00:00Z",FinishedAt:"2026-10-02T14:00:01Z",ElapsedNanos:1000000000},
 }
 published,err:=artifact.PublishArtifact(artifact.Store{Root:root},artifact.ArtifactInput{Manifest:manifest,TargetPath:binary,World:payloads});if err!=nil{return err}
 _,verifyErr:=runner.Replay(context.Background(),runner.ReplaySpec{ArtifactPath:published.Path,VerifyOnly:true,ToolchainRoot:toolchain})
 _,minimizeErr:=runner.Minimize(context.Background(),runner.MinimizeSpec{ArtifactPath:published.Path,OutputRoot:filepath.Join(root,"minimized"),AttemptBudget:1,ToolchainRoot:toolchain})
 if verifyErr==nil || !strings.Contains(verifyErr.Error(),"stored target uses unsupported coverage instrumentation") {return fmt.Errorf("verify rejection: %v",verifyErr)}
 if minimizeErr==nil || !strings.Contains(minimizeErr.Error(),"stored target uses unsupported coverage instrumentation") {return fmt.Errorf("minimize rejection: %v",minimizeErr)}
 out:=map[string]any{"binary":binary,"binary_sha256":record.HashBytes(contents),"settings":target.ProjectBuildInfo(info),"artifact":published.Path,"record_hash":published.Manifest.RecordHash,"verify_error":verifyErr.Error(),"minimize_error":minimizeErr.Error()}
 encoded,err:=json.MarshalIndent(out,"","  ");if err!=nil{return err}
 return os.WriteFile(filepath.Join(root,"results.json"),append(encoded,'\n'),0600)
}
