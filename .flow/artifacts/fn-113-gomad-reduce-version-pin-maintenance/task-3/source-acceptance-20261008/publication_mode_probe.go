package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func main() {
	for _, filesystem := range []string{os.TempDir(), "/dev/shm"} {
		var capacity syscall.Statfs_t
		if err := syscall.Statfs(filesystem, &capacity); err != nil { panic(err) }
		if capacity.Bavail*uint64(capacity.Bsize)<1<<20 { panic("insufficient probe capacity") }
		parent, err := os.MkdirTemp(filesystem, "fn1133-publication-mode-probe-")
		if err != nil { panic(err) }
		fmt.Printf("probe %s uid %d filesystem type %x available %d\n", parent, os.Getuid(), capacity.Type, capacity.Bavail*uint64(capacity.Bsize))
		for iteration:=0;iteration<32;iteration++ {
			directory:=filepath.Join(parent,fmt.Sprintf("case-%02d",iteration))
			prior:=[]byte("{\"prior\":\"complete dossier\"}\n")
			file:=filepath.Join(directory,"upgrade-dossier.json")
			if err:=os.Mkdir(directory,0o700);err!=nil { panic(err) }
			if err:=os.WriteFile(file,prior,0o644);err!=nil { panic(err) }
			observations:=map[string]any{"iteration":iteration,"directory":directory}
			observe:=func(stage string) {
				contents,readErr:=os.ReadFile(file)
				info,statErr:=os.Stat(file)
				names,listErr:=os.ReadDir(directory)
				entries:=[]string{}
				for _,entry:=range names { entries=append(entries,entry.Name()) }
				mode:="unavailable"
				if info!=nil { mode=fmt.Sprintf("%o",info.Mode().Perm()) }
				parentInfo,parentErr:=os.Stat(directory)
				parentMode:="unavailable"
				if parentInfo!=nil {parentMode=fmt.Sprintf("%o",parentInfo.Mode().Perm())}
				observations[stage]=map[string]any{"read_error":fmt.Sprint(readErr),"stat_error":fmt.Sprint(statErr),"list_error":fmt.Sprint(listErr),"entries":entries,"mode":mode,"parent_mode":parentMode,"parent_stat_error":fmt.Sprint(parentErr),"sha256":fmt.Sprintf("%x",sha256.Sum256(contents)),"prior_preserved":string(contents)==string(prior)}
			}
			observe("before chmod")
			if err:=os.Chmod(directory,0o500);err!=nil {panic(err)}
			observe("after chmod")
			mkdirErr:=os.MkdirAll(directory,0o755)
			observations["mkdir_all_error"]=fmt.Sprint(mkdirErr)
			temporary,createErr:=os.CreateTemp(directory,".safefile-*")
			observations["create_error"]=fmt.Sprint(createErr)
			observations["create_is_eacces"]=errors.Is(createErr,syscall.EACCES)
			if temporary!=nil {if err:=temporary.Close();err!=nil {panic(err)}}
			observe("after create refusal")
			encoded,err:=json.Marshal(observations)
			if err!=nil {panic(err)}
			fmt.Println(string(encoded))
			if err:=os.Chmod(directory,0o700);err!=nil {panic(err)}
			if err:=os.RemoveAll(directory);err!=nil {panic(err)}
			if _,err:=os.Lstat(directory);!errors.Is(err,os.ErrNotExist) {panic(fmt.Sprintf("probe cleanup did not remove %s: %v",directory,err))}
		}
		if names,err:=os.ReadDir(parent);err!=nil||len(names)!=0 {panic(fmt.Sprintf("probe parent not empty %v %v",names,err))}
		if err:=os.Remove(parent);err!=nil {panic(err)}
		fmt.Println("checked restoration and cleanup",parent)
	}
}
