package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.temporal.io/server/tools/gomad3/target"
)

type cause struct { Type, Text string; Children []cause }
func chain(err error) cause {if err==nil{return cause{}};v:=cause{Type:fmt.Sprintf("%T",err),Text:err.Error()};if multi,ok:=err.(interface{Unwrap() []error});ok{for _,child:=range multi.Unwrap(){v.Children=append(v.Children,chain(child))}}else if child:=errors.Unwrap(err);child!=nil{v.Children=append(v.Children,chain(child))};return v}
func lockAvailable(file string)(bool,error){f,err:=os.OpenFile(file,os.O_RDWR,0);if err!=nil{return false,err};err=syscall.Flock(int(f.Fd()),syscall.LOCK_EX|syscall.LOCK_NB);if err!=nil{return false,errors.Join(err,f.Close())};return true,errors.Join(syscall.Flock(int(f.Fd()),syscall.LOCK_UN),f.Close())}

func main(){
	var config struct{Scratch,Root,Module,Key string;Modes []string}
	bytes,err:=os.ReadFile(os.Getenv("TASK9_FAILURE_CONTROL"));if err!=nil{panic(err)};if err:=json.Unmarshal(bytes,&config);err!=nil{panic(err)}
	base,err:=os.MkdirTemp("","task9-build-contract-");if err!=nil{panic(err)};defer func(){if err:=os.RemoveAll(base);err!=nil{panic(err)}}()
	for _,mode:=range config.Modes{
		marker:=filepath.Join(base,mode+".started");if err:=os.Setenv("TASK9_FAILURE_MODE",mode);err!=nil{panic(err)};if err:=os.Setenv("TASK9_BUILD_MARKER",marker);err!=nil{panic(err)}
		ctx,cancel:=context.WithTimeout(context.Background(),15*time.Second);if mode=="before-cancel"{cancel()}else if mode=="deadline"{cancel();ctx,cancel=context.WithTimeout(context.Background(),3*time.Second)}
		type outcome struct{prepared target.Prepared;err error};done:=make(chan outcome,1)
		go func(){p,err:=target.Prepare(ctx,target.Spec{Kind:target.KindGoRun,Source:".",WorkingDir:config.Module,ToolchainRoot:config.Root,PreparationRoot:filepath.Join(base,mode)});done<-outcome{p,err}}()
		cacheLock:=filepath.Join(config.Root,"builds",config.Key,"target-cache/gomad-cache.lock");held:=false
		if mode=="cancel"||mode=="deadline"{
			ticker:=time.NewTicker(time.Millisecond);started:=false
			for !started{select{case<-ticker.C:_,statErr:=os.Stat(marker);started=statErr==nil;if statErr!=nil&&!errors.Is(statErr,os.ErrNotExist){panic(statErr)};case<-ctx.Done():panic("build startup acknowledgement missing")}}
			ticker.Stop();available,lockErr:=lockAvailable(cacheLock);held=!available&&errors.Is(lockErr,syscall.EWOULDBLOCK);if !held{panic(fmt.Sprintf("shared build lock absent: %v",lockErr))};if mode=="cancel"{cancel()}
		}
		got:=<-done;cancel();markerBytes,markerErr:=os.ReadFile(marker);started:=markerErr==nil;if markerErr!=nil&&!errors.Is(markerErr,os.ErrNotExist){panic(markerErr)}
		pid:=0;gone:=false;if started{pid,err=strconv.Atoi(strings.TrimSpace(string(markerBytes)));if err!=nil{panic(err)};gone=errors.Is(syscall.Kill(pid,0),syscall.ESRCH)}
		available,lockErr:=lockAvailable(cacheLock);lockMissing:=errors.Is(lockErr,os.ErrNotExist);if lockErr!=nil&&!lockMissing{panic(lockErr)}
		var exitErr *exec.ExitError;asExit:=errors.As(got.err,&exitErr);code:=0;state,signal:="","";var stderr []byte;var waitStatus uint32
		if asExit{code=exitErr.ExitCode();stderr=exitErr.Stderr;if exitErr.ProcessState!=nil{state=exitErr.ProcessState.String();if status,ok:=exitErr.Sys().(syscall.WaitStatus);ok{waitStatus=uint32(status);if status.Signaled(){signal=status.Signal().String()}}}}
		row:=struct{Name string;Cause cause;Canceled,Deadline,ExecExit bool;Exit int;Signal,ProcessState string;WaitStatus uint32;ExecStderr []byte;Prepared target.Prepared;ZeroPrepared,BuildStarted,ChildGone,SharedLockObserved,ExclusiveLockReacquired,CacheLockMissing bool;PID int}{mode,chain(got.err),errors.Is(got.err,context.Canceled),errors.Is(got.err,context.DeadlineExceeded),asExit,code,signal,state,waitStatus,stderr,got.prepared,reflect.DeepEqual(got.prepared,target.Prepared{}),started,gone,held,available,lockMissing,pid}
		if err:=json.NewEncoder(os.Stdout).Encode(row);err!=nil{panic(err)}
	}
}
