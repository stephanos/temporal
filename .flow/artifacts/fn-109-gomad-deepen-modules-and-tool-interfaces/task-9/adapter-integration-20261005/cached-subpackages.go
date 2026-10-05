package main

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"go.temporal.io/server/tools/gomad3/target"
)

func main() {
	if len(os.Args) != 6 { panic("usage: cached-subpackages <go> <directory> <import> <goos> <goarch>") }
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	digest, err := target.AdapterPreparedSourceSetSHA256(ctx, os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5])
	result := map[string]any{"digest":digest}
	if err != nil { result["error"] = err.Error() }
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil { panic(err) }
	if err != nil { os.Exit(1) }
}
