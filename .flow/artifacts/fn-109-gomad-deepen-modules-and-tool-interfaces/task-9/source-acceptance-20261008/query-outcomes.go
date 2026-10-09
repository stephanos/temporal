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
	"strings"
	"syscall"
	"time"

	"go.temporal.io/server/tools/gomad3/target"
)

type cause struct { Type, Text string; Children []cause }
func chain(err error) cause {
	if err == nil { return cause{} }
	v := cause{Type:fmt.Sprintf("%T",err),Text:err.Error()}
	if multi,ok:=err.(interface{Unwrap() []error});ok { for _,child:=range multi.Unwrap(){v.Children=append(v.Children,chain(child))} } else if child:=errors.Unwrap(err);child!=nil {v.Children=append(v.Children,chain(child))}
	return v
}

func main() {
	base,err:=os.MkdirTemp("","task9-query-contract-");if err!=nil{panic(err)}
	defer func(){if err:=os.RemoveAll(base);err!=nil{panic(err)}}()
	cache:=filepath.Join(base,"cache");if err:=os.Mkdir(cache,0o700);err!=nil{panic(err)}
	for _,tc:=range []struct{name,kind,mode string;exit int}{
		{"identity_nonzero","identity","",7},{"cache_nonzero","cache","",7},
		{"identity_stderr_capacity","identity","capacity",7},{"cache_stderr_capacity","cache","capacity",7},
		{"cache_before_cancel","cache","before",0},{"cache_cancel_after_ack","cache","cancel",0},{"cache_deadline_after_ack","cache","deadline",0},
	} {
		root:=filepath.Join(base,tc.name);key:=strings.Repeat("9",64)
		for _,dir:=range []string{filepath.Join(root,"bin"),filepath.Join(root,"builds",key,"bin")} {if err:=os.MkdirAll(dir,0o700);err!=nil{panic(err)}}
		if err:=os.WriteFile(filepath.Join(root,"build-key"),[]byte(key+"\n"),0o600);err!=nil{panic(err)}
		output:=cache+"\n";if tc.kind=="identity"{output="go1.27.1\n"+runtime.GOOS+"\n"+runtime.GOARCH+"\n0\n"}
		marker:=filepath.Join(root,"started")
		body:=fmt.Sprintf("#!/bin/sh\nprintf '%%s' '%s'\nprintf 'query diagnostic' >&2\n",output)
		if tc.mode=="capacity"{body=fmt.Sprintf("#!/bin/sh\nprintf '%%s' '%s'\nprintf '%%065536d' 0 >&2\n",output)}
		if tc.mode=="cancel"||tc.mode=="deadline"{body+=fmt.Sprintf("touch '%s'\nexec /bin/sleep 30\n",marker)}else{body+=fmt.Sprintf("exit %d\n",tc.exit)}
		for _,file:=range []string{filepath.Join(root,"bin/go"),filepath.Join(root,"builds",key,"bin/go")} {if err:=os.WriteFile(file,[]byte(body),0o700);err!=nil{panic(err)}}
		ctx,cancel:=context.WithCancel(context.Background());if tc.mode=="before"{cancel()};if tc.mode=="deadline"{cancel();ctx,cancel=context.WithTimeout(context.Background(),500*time.Millisecond)}
		result:=make(chan error,1)
		go func(){var err error;if tc.kind=="identity"{_,err=target.ReadToolchainIdentity(root)}else{_,err=target.ReadModuleCache(ctx,root)};result<-err}()
		ack:=false
		if tc.mode=="cancel"||tc.mode=="deadline" {
			ticker:=time.NewTicker(time.Millisecond);watchdog:=time.NewTimer(2*time.Second)
			for !ack{select{case<-ticker.C:_,statErr:=os.Stat(marker);ack=statErr==nil;if statErr!=nil&&!errors.Is(statErr,os.ErrNotExist){panic(statErr)};case<-watchdog.C:panic("child acknowledgement missing")}}
			ticker.Stop();watchdog.Stop();if tc.mode=="cancel"{cancel()}
		}
		err:=<-result;cancel();var exitErr *exec.ExitError;asExit:=errors.As(err,&exitErr);code:=0;signal,state:="","";var stderr []byte;var waitStatus uint32
		if asExit{code=exitErr.ExitCode();stderr=exitErr.Stderr;if exitErr.ProcessState!=nil{state=exitErr.ProcessState.String();if status,ok:=exitErr.Sys().(syscall.WaitStatus);ok{waitStatus=uint32(status);if status.Signaled(){signal=status.Signal().String()}}}}
		row:=struct{Name string;Cause cause;Canceled,Deadline,ExecExit bool;Exit int;Signal,ProcessState string;WaitStatus uint32;ExecStderr []byte;Acknowledged bool}{tc.name,chain(err),errors.Is(err,context.Canceled),errors.Is(err,context.DeadlineExceeded),asExit,code,signal,state,waitStatus,stderr,ack}
		if err:=json.NewEncoder(os.Stdout).Encode(row);err!=nil{panic(err)}
	}
}
