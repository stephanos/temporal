package main
import (
 "context"
 "encoding/json"
 "fmt"
 "os"
 "go.temporal.io/server/tools/gomad3/deterministicio"
 "go.temporal.io/server/tools/gomad3/target"
)
func main() {
 spec, adapters, err := deterministicio.Default().PrepareBuildAdapters(target.Spec{
  Kind: target.KindGoRun, Source: ".", WorkingDir: "/evidence/consumer", PreparationRoot: "/evidence/review-preparation-green",
  BuildTags: []string{"test_dep","integration","hashicorpmetrics","gomad"},
 }, "/gomodcache")
 if err!=nil { fmt.Fprintln(os.Stderr,err); os.Exit(1) }
 spec.ToolchainRoot="/usr/local/go"
 review, err := target.ReviewCapabilities(context.Background(),spec)
 if err!=nil { fmt.Fprintln(os.Stderr,err); os.Exit(1) }
 result:=struct { Adapters []deterministicio.BuildAdapter; Review target.CapabilityReview }{adapters,review}
 if err=json.NewEncoder(os.Stdout).Encode(result); err!=nil { fmt.Fprintln(os.Stderr,err); os.Exit(1) }
}
