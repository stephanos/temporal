package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/target"
)

func must(err error) { if err != nil { panic(err) } }
func main() {
	if len(os.Args)!=3 { panic("usage: source-selection <stock-go> <scratch>") }
	goCommand, scratch := os.Args[1], os.Args[2]
	fixture := filepath.Join(scratch,"fixture")
	must(os.Mkdir(fixture,0700))
	files := map[string]string{
		"common.go":"package adapter\nconst Common = 1\n",
		"selected_darwin_arm64.go":"package adapter\nconst Platform = \"darwin/arm64\"\n",
		"selected_linux_amd64.go":"package adapter\nconst Platform = \"linux/amd64\"\n",
		"excluded_windows.go":"package adapter\nconst Excluded = true\n",
		"excluded_test.go":"package adapter\n",
		"disabled_cgo.go":"//go:build cgo\n\npackage adapter\nimport \"C\"\n",
		"common.s":"// Shared assembly fixture; listing only.\n",
		"selected_darwin_arm64.s":"// Darwin assembly fixture; listing only.\n",
		"selected_linux_amd64.s":"// Linux assembly fixture; listing only.\n",
		"common.h":"// Shared header fixture; listing only.\n",
	}
	for name,data := range files { must(os.WriteFile(filepath.Join(fixture,name),[]byte(data),0600)) }
	names:=make([]string,0,len(files));for name:=range files {names=append(names,name)};sort.Strings(names)
	manifest:=map[string]string{};for _,name:=range names {manifest[name]=fmt.Sprintf("%x",sha256.Sum256([]byte(files[name])))}
	must(json.NewEncoder(os.Stdout).Encode(map[string]any{"kind":"fixture","directory":fixture,"source_sha256":manifest,"source_bytes":files}))
	measure(goCommand,scratch,"nonempty-fixture",fixture,"example.test/adapter","fixture")
}

func measure(goCommand,scratch,label,directory,importPath,kind string) {
	for _,platform:=range []string{"darwin/arm64","linux/amd64"} {
		goos,goarch,_:=strings.Cut(platform,"/")
		ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second)
		gopath,err:=os.MkdirTemp(scratch,"gopath-");must(err)
		command:=exec.CommandContext(ctx,goCommand,"list","-e","-find","-json",".")
		command.Dir=directory
		reserved:=map[string]bool{"CGO_ENABLED":true,"GOMADSEED":true,"GOMAD3_CHILD_SEED":true,"GOCACHE":true,"GOENV":true,"GOEXPERIMENT":true,"GOFLAGS":true,"GOROOT":true,"GOTOOLCHAIN":true,"GOWORK":true,"TZ":true}
		for _,entry:=range os.Environ(){name,_,_:=strings.Cut(entry,"=");if !reserved[name]{command.Env=append(command.Env,entry)}}
		command.Env=append(command.Env,"CGO_ENABLED=0","GOENV=off","GOEXPERIMENT=nogreenteagc","GOFLAGS=","GOTOOLCHAIN=local","GOWORK=off","TZ=UTC","GO111MODULE=off","GOPATH="+gopath,"GOOS="+goos,"GOARCH="+goarch)
		var stdout,stderr bytes.Buffer;command.Stdout=&stdout;command.Stderr=&stderr
		runErr:=command.Run()
		var listing map[string]any;decodeErr:=json.Unmarshal(stdout.Bytes(),&listing)
		digest,helperErr:=target.AdapterPreparedSourceSetSHA256(ctx,goCommand,directory,importPath,goos,goarch)
		inventory:=map[string]any{};sourceHashes:=map[string]string{}
		for _,key:=range []string{"GoFiles","CgoFiles","CFiles","CXXFiles","MFiles","HFiles","FFiles","SFiles","SwigFiles","SwigCXXFiles","SysoFiles","IgnoredGoFiles","IgnoredOtherFiles","TestGoFiles"} {
			if value,ok:=listing[key];ok {inventory[key]=value;if key!="IgnoredGoFiles"&&key!="IgnoredOtherFiles"&&key!="TestGoFiles" {for _,raw:=range value.([]any) {name:=raw.(string);contents,err:=os.ReadFile(filepath.Join(directory,name));must(err);sourceHashes[name]=fmt.Sprintf("%x",sha256.Sum256(contents))}}}
		}
		environment:=map[string]string{};for _,entry:=range command.Env {name,value,_:=strings.Cut(entry,"=");if reserved[name]||name=="GO111MODULE"||name=="GOPATH"||name=="GOOS"||name=="GOARCH"||name=="GOPROXY"||name=="GOSUMDB"||name=="GOMAXPROCS" {environment[name]=value}}
		out:=map[string]any{"kind":kind,"label":label,"platform":platform,"directory":directory,"import_path":importPath,"command":command.Args,"controlled_environment":environment,"stdout_bytes":stdout.Len(),"stderr_bytes":stderr.Len(),"stdout_sha256":fmt.Sprintf("%x",sha256.Sum256(stdout.Bytes())),"stderr_sha256":fmt.Sprintf("%x",sha256.Sum256(stderr.Bytes())),"stdout":stdout.String(),"stderr":stderr.String(),"inventory":inventory,"source_sha256":sourceHashes,"digest":digest,"command_exit":0}
		if runErr!=nil {out["command_error"]=runErr.Error();out["command_exit"]=command.ProcessState.ExitCode()};if decodeErr!=nil {out["decode_error"]=decodeErr.Error()};if helperErr!=nil {out["helper_error"]=helperErr.Error()}
		must(os.RemoveAll(gopath));cancel();must(json.NewEncoder(os.Stdout).Encode(out))
		if kind=="fixture"&&(runErr!=nil||decodeErr!=nil||helperErr!=nil||digest==""||len(sourceHashes)==0){panic("nonempty fixture failed")}
	}
}
