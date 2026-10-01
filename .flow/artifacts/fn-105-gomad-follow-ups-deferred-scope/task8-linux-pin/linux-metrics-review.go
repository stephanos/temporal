package main
import (
 "context"
 "encoding/json"
 "fmt"
 "os"
 "os/exec"
 "go.temporal.io/server/tools/gomad3/deterministicio"
 "go.temporal.io/server/tools/gomad3/target"
)
func main() {
 spec, adapters, err := deterministicio.Default().PrepareBuildAdapters(target.Spec{
  Kind: target.KindGoTest, Source: ".", WorkingDir: "/evidence/metrics-consumer", PreparationRoot: "/evidence/metrics-preparation",
  BuildTags: []string{"test_dep","integration","hashicorpmetrics","gomad"},
 }, "/gomodcache")
 if err!=nil { fmt.Fprintln(os.Stderr,err); os.Exit(1) }
 cmd:=exec.Command("/usr/local/go/bin/go","test","-v","-p=2","-mod=readonly","-modfile="+spec.BuildModFile,"-tags=test_dep,hashicorpmetrics",".")
 cmd.Dir=spec.WorkingDir
 output, err:=cmd.CombinedOutput()
 fmt.Fprint(os.Stderr,string(output))
 if err!=nil { fmt.Fprintln(os.Stderr,err); os.Exit(1) }
 spec.ToolchainRoot="/usr/local/go"
 review, err := target.ReviewCapabilities(context.Background(),spec)
 if err!=nil { fmt.Fprintln(os.Stderr,err); os.Exit(1) }
 result:=struct { Adapters []deterministicio.BuildAdapter; Review target.CapabilityReview }{adapters,review}
 if err=json.NewEncoder(os.Stdout).Encode(result); err!=nil { fmt.Fprintln(os.Stderr,err); os.Exit(1) }
}
