package main
import (
 "encoding/json"
 "fmt"
 "os"
 "go.temporal.io/server/tools/gomad3/deterministicio"
 "go.temporal.io/server/tools/gomad3/target"
)
func main() {
 spec, adapters, err := deterministicio.Default().PrepareBuildAdapters(target.Spec{
  Kind: target.KindGoRun, Source: ".", WorkingDir: "/private/tmp/fn105-d8-sentry-linux/consumer",
  PreparationRoot: "/private/tmp/fn105-d8-sentry-linux/host-preparation",
 }, "/Users/stephan/go/pkg/mod")
 if err != nil { panic(err) }
 if len(adapters)!=1 { panic(fmt.Sprint(adapters)) }
 b, err := json.MarshalIndent(adapters[0], "", "  "); if err!=nil { panic(err) }
 if err=os.WriteFile("/private/tmp/fn105-d8-sentry-linux/host-adapter.json", b, 0600); err!=nil { panic(err) }
 fmt.Println(adapters[0].ReplacementRoot)
 _=spec
}
