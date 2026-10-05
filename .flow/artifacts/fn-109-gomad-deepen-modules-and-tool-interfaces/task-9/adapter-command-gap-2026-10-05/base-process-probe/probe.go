package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"go.temporal.io/server/tools/gomad3/target"
)

type config struct { Root, Name, Mode, Stdout, Stderr string; Exit int }
type started struct { PID int; CWD, GOPATH string; Argv []string; At time.Time }
type result struct {
	Name, Command, Directory, Digest, Error, ContextError string
	Chain []map[string]any
	IsCanceled, IsDeadline, IsENOENT, IsEACCES, IsENOEXEC bool
	PathError map[string]any `json:",omitempty"`
	ExitError map[string]any `json:",omitempty"`
	Started *started `json:",omitempty"`
	MarkerBeforeDeadline bool
	ChildGone, GOPATHRemoved bool
}

func must(err error) { if err != nil { panic(err) } }
func writeJSON(path string, v any) { b,e:=json.MarshalIndent(v,"","  "); must(e); must(os.WriteFile(path,append(b,'\n'),0600)) }
func child() {
	var c config; must(json.Unmarshal([]byte(os.Getenv("ADAPTER_BASE_PROBE_CONFIG")), &c))
	dir:=filepath.Join(c.Root,c.Name)
	cwd,e:=os.Getwd(); must(e)
	must(os.WriteFile(filepath.Join(dir,"child.stdout"),[]byte(c.Stdout),0600))
	must(os.WriteFile(filepath.Join(dir,"child.stderr"),[]byte(c.Stderr),0600))
	_,e=os.Stdout.WriteString(c.Stdout); must(e)
	_,e=os.Stderr.WriteString(c.Stderr); must(e)
	writeJSON(filepath.Join(dir,"started.json"),started{os.Getpid(),cwd,os.Getenv("GOPATH"),os.Args,time.Now()})
	switch c.Mode {
	case "sleep": time.Sleep(time.Hour)
	case "signal": must(syscall.Kill(os.Getpid(),syscall.SIGTERM)); time.Sleep(time.Hour)
	}
	os.Exit(c.Exit)
}
func marker(path string) (*started,error) {
	b,e:=os.ReadFile(path); if e!=nil{return nil,e}; var s started; e=json.Unmarshal(b,&s); return &s,e
}
func probe(c config, command, dir, contextMode string) result {
	caseDir:=filepath.Join(c.Root,c.Name); must(os.Mkdir(caseDir,0700))
	b,e:=json.Marshal(c); must(e); must(os.Setenv("ADAPTER_BASE_PROBE_CONFIG",string(b)))
	ctx,cancel:=context.WithTimeout(context.Background(),10*time.Second)
	defer cancel()
	var deadline time.Time
	switch contextMode {
	case "canceled_before": cancel()
	case "expired_before":
		cancel(); ctx,cancel=context.WithDeadline(context.Background(),time.Now().Add(-time.Second)); defer cancel()
	case "deadline_after":
		cancel(); deadline=time.Now().Add(2*time.Second); ctx,cancel=context.WithDeadline(context.Background(),deadline); defer cancel()
	}
	var observed *started
	done:=make(chan struct{})
	if contextMode=="canceled_after" || contextMode=="deadline_after" {
		go func(){
			defer close(done)
			ticker:=time.NewTicker(time.Millisecond); defer ticker.Stop()
			for {
				if s,err:=marker(filepath.Join(caseDir,"started.json"));err==nil {observed=s;if contextMode=="canceled_after" {cancel()};return}
				select {case <-ctx.Done():return;case <-ticker.C:}
			}
		}()
	} else {close(done)}
	digest,err:=target.AdapterPreparedSourceSetSHA256(ctx,command,dir,"example.test/adapter","darwin","arm64")
	<-done
	r:=result{Name:c.Name,Command:command,Directory:dir,Digest:digest,IsCanceled:errors.Is(err,context.Canceled),IsDeadline:errors.Is(err,context.DeadlineExceeded),IsENOENT:errors.Is(err,syscall.ENOENT),IsEACCES:errors.Is(err,syscall.EACCES),IsENOEXEC:errors.Is(err,syscall.ENOEXEC)}
	if err!=nil {r.Error=err.Error()}
	if ctx.Err()!=nil {r.ContextError=ctx.Err().Error()}
	for cause:=err;cause!=nil;cause=errors.Unwrap(cause) {r.Chain=append(r.Chain,map[string]any{"type":fmt.Sprintf("%T",cause),"text":cause.Error()})}
	var pe *os.PathError
	if errors.As(err,&pe) {r.PathError=map[string]any{"op":pe.Op,"path":pe.Path,"errno_type":fmt.Sprintf("%T",pe.Err),"errno":pe.Err.Error()};if n,ok:=pe.Err.(syscall.Errno);ok{r.PathError["errno_number"]=uint64(n)}}
	var ee *exec.ExitError
	if errors.As(err,&ee) {ps:=ee.ProcessState;r.ExitError=map[string]any{"exit_code":ps.ExitCode(),"pid":ps.Pid(),"exited":ps.Exited(),"success":ps.Success(),"state":ps.String(),"stderr":string(ee.Stderr)};if ws,ok:=ps.Sys().(syscall.WaitStatus);ok{r.ExitError["signaled"]=ws.Signaled();r.ExitError["signal"]=ws.Signal().String();r.ExitError["signal_number"]=int(ws.Signal());r.ExitError["wait_status"]=int(ws)}}
	r.Started,_=marker(filepath.Join(caseDir,"started.json"))
	if observed!=nil && !deadline.IsZero(){r.MarkerBeforeDeadline=observed.At.Before(deadline)}
	if r.Started!=nil {r.ChildGone=errors.Is(syscall.Kill(r.Started.PID,0),syscall.ESRCH);_,e=os.Stat(r.Started.GOPATH);r.GOPATHRemoved=errors.Is(e,os.ErrNotExist)}
	if contextMode=="canceled_after" || contextMode=="deadline_after" {if observed==nil {panic("after-start context case failed to acknowledge start")};if contextMode=="deadline_after"&&!r.MarkerBeforeDeadline{panic("marker was not before deadline")}}
	writeJSON(filepath.Join(caseDir,"result.json"),r)
	must(json.NewEncoder(os.Stdout).Encode(r))
	return r
}
func main() {
	if len(os.Args)>1 && os.Args[1]=="list" {child();return}
	if len(os.Args)!=2 {panic("usage: probe <root>")}
	root:=os.Args[1];self,e:=os.Executable();must(e);cwd,e:=os.Getwd();must(e)
	writeJSON(filepath.Join(root,"host.json"),map[string]any{"go_version":runtime.Version(),"goos":runtime.GOOS,"goarch":runtime.GOARCH,"cwd":cwd,"self":self})
	valid:=fmt.Sprintf(`{"Dir":%q,"Name":"adapter","ImportPath":"ignored/input"}`,root)
	base:=func(name string)config{return config{Root:root,Name:name,Stdout:valid,Stderr:" \tobserved diagnostic\n\n "}}
	run:=func(name,mode,out string,exit int,contextMode string){c:=base(name);c.Mode=mode;c.Stdout=out;c.Exit=exit;probe(c,self,root,contextMode)}
	run("success_valid_json","",valid,0,"")
	run("exit7_valid_json","",valid,7,"")
	run("exit7_malformed_json","","{broken",7,"")
	run("exit7_listed_error","",`{"Error":{"Err":"listed error"}}`,7,"")
	run("exit0_malformed_json","","{broken",0,"")
	run("exit0_trailing_json","",valid+" {}",0,"")
	run("exit0_listed_error","",`{"Error":{"Err":"listed error"}}`,0,"")
	run("exit0_missing_fields","",`{}`,0,"")
	run("ordinary_sigterm","signal",valid,0,"")
	for _,mode:=range []string{"canceled_before","expired_before","canceled_after","deadline_after"}{run(mode,"sleep",valid,0,mode)}
	probe(base("missing_executable"),filepath.Join(root,"absent"),root,"")
	nonexec:=filepath.Join(root,"nonexecutable-file");must(os.WriteFile(nonexec,[]byte("not executable\n"),0600));probe(base("nonexecutable"),nonexec,root,"")
	invalid:=filepath.Join(root,"invalid-format");must(os.WriteFile(invalid,[]byte("invalid executable\n"),0700));probe(base("invalid_format"),invalid,root,"")
	probe(base("invalid_directory"),self,filepath.Join(root,"missing-directory"),"")
	probe(base("empty_command"),"",root,"")
	probe(base("empty_directory"),self,"","")
	probe(base("dot_directory"),self,".","")
	for _,name:=range []string{"relative-cmd","bare-cmd"}{must(os.Symlink(self,filepath.Join(root,name)))}
	rel,e:=filepath.Rel(cwd,root);must(e)
	probe(base("relative_directory"),self,rel,"")
	probe(base("relative_command_from_child_dir"),"./relative-cmd",root,"")
	probe(base("relative_command_with_relative_dir"),"./relative-cmd",rel,"")
	probe(base("relative_command_resolves_child_not_parent"),filepath.Join(rel,"relative-cmd"),root,"")
	oldPath:=os.Getenv("PATH");must(os.Setenv("PATH",root+":"+oldPath));probe(base("bare_command_absolute_path_entry"),"bare-cmd",root,"");must(os.Setenv("PATH",oldPath))
	for _,mode:=range []string{"canceled_before","expired_before"}{probe(base("missing_executable_"+mode),filepath.Join(root,"absent"),root,mode);probe(base("bare_missing_"+mode),"fn109-nonexistent-command-"+strconv.Itoa(os.Getpid()),root,mode)}
}
