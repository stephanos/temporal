package toolchain

import (
 "bytes"
 "go/ast"
 "go/format"
 "go/parser"
 "go/token"
 "os"
 "os/exec"
 "path/filepath"
 "testing"
)

func TestStreamPendingReadinessBatches(t *testing.T) {
 file, err := parser.ParseFile(token.NewFileSet(), "net_stream_wasm.go.txt", netStreamGlue, 0)
 if err != nil { t.Fatal(err) }
 wanted := map[string]bool{"gomadStreamInvoke":true,"gomadStreamFrame":true,"gomadStreamRemove":true,"gomadStreamPollPending":true}
 var declarations bytes.Buffer
 for _, declaration := range file.Decls {
  include := false
  switch declaration := declaration.(type) {
  case *ast.GenDecl:
   include = declaration.Tok == token.CONST
   for _, spec := range declaration.Specs {
    if spec, ok := spec.(*ast.TypeSpec); ok && spec.Name.Name == "gomadStreamPending" {include = true}
   }
  case *ast.FuncDecl:
   include = wanted[declaration.Name.Name]
  }
  if include {
   if err := format.Node(&declarations, token.NewFileSet(), declaration); err != nil {t.Fatal(err)}
   declarations.WriteByte('\n')
  }
 }
 source := `package pending
import ("encoding/binary";"reflect";"testing";"unsafe")
var byteorder=struct{
 BEUint32 func([]byte)uint32
 BEUint64 func([]byte)uint64
 BEPutUint32 func([]byte,uint32)
 BEPutUint64 func([]byte,uint64)
}{binary.BigEndian.Uint32,binary.BigEndian.Uint64,binary.BigEndian.PutUint32,binary.BigEndian.PutUint64}
var gomadStreamNextRequest uint64
var gomadStreamPendingHead,gomadStreamPendingTail *gomadStreamPending
var ready=map[uint64]bool{}
var polls []uint64
var probes [][]uint64
var witnesses int
var malformed string
var cascade bool
func gomadStreamWitness(){witnesses++}
func gomadWasmNetStream(p *byte,n uint32)uint32{
 b:=unsafe.Slice(p,int(n))
 switch op:=byteorder.BEUint32(b[4:]);op {
 case 14:
  count:=int(byteorder.BEUint32(b[36:]))
  if count<1 || count>128 || len(b)!=64+8*count {panic("unbounded readiness probe")}
  expected:=make([]byte,len(b));byteorder.BEPutUint32(expected,2);byteorder.BEPutUint32(expected[4:],14);byteorder.BEPutUint32(expected[36:],uint32(count));copy(expected[64:],b[64:])
  if !reflect.DeepEqual(b,expected){panic("noncanonical readiness probe")}
  ids:=make([]uint64,count)
  for i:=range ids {id:=byteorder.BEUint64(b[64+8*i:]);ids[i]=id;if !ready[id]{byteorder.BEPutUint64(b[64+8*i:],0)}}
  probes=append(probes,ids)
  switch malformed {case "id":byteorder.BEPutUint64(b[64:],9999);case "count":byteorder.BEPutUint32(b[36:],0);case "status":byteorder.BEPutUint32(b[40:],1)}
 case gomadStreamPoll:
  id:=byteorder.BEUint64(b[8:]);polls=append(polls,id)
  if ready[id]{delete(ready,id);byteorder.BEPutUint32(b[40:],0);byteorder.BEPutUint32(b[36:],1);b[64]=byte(id)}else{byteorder.BEPutUint32(b[40:],1)}
 default:panic("unexpected operation")
 }
 if cascade && byteorder.BEUint32(b[4:])==gomadStreamPoll && byteorder.BEUint64(b[8:])==1 {ready[2]=true}
 return 0
}
func setup(n int)[]*gomadStreamPending{
 gomadStreamPendingHead=nil;gomadStreamPendingTail=nil;gomadStreamNextRequest=0;ready=map[uint64]bool{};polls=nil;probes=nil;witnesses=0;malformed="";cascade=false
 pending:=make([]*gomadStreamPending,n)
 for i:=range pending{
  p:=&gomadStreamPending{buffer:gomadStreamFrame(gomadStreamRead,1,0,0,1,0),done:make(chan struct{}),active:true};pending[i]=p
  if gomadStreamPendingTail==nil{gomadStreamPendingHead=p}else{gomadStreamPendingTail.next=p};gomadStreamPendingTail=p
 }
 return pending
}
func TestRepeatedScansAndReadyOrder(t *testing.T){
 pending:=setup(259)
 for scan:=0;scan<2;scan++{
  if woken,waiting:=gomadStreamPollPending();woken || !waiting {t.Fatal("pending scan changed state")}
 }
 expected:=[][]uint64{}
 for scan:=0;scan<2;scan++ {for start:=1;start<=259;start+=128 {ids:=[]uint64{};for id:=start;id<=259 && id<start+128;id++ {ids=append(ids,uint64(id))};expected=append(expected,ids)}}
 if !reflect.DeepEqual(probes,expected)||len(polls)!=0 {t.Fatalf("pending scans: probes=%v polls=%v",probes,polls)}
 for _,id:=range []uint64{2,128,129,259}{ready[id]=true}
 if woken,waiting:=gomadStreamPollPending();!woken || !waiting {t.Fatal("ready scan did not wake")}
 if !reflect.DeepEqual(polls,[]uint64{2,128,129,259}) {t.Fatalf("ready wake order=%v",polls)}
 for i,p:=range pending{
  completed:=i==1||i==127||i==128||i==258
  select {case <-p.done:if !completed{t.Fatal("pending request woken")};default:if completed{t.Fatal("completion not woken")}}
  if p.active==completed {t.Fatal("pending removal changed")}
  if completed && (p.buffer[64]!=byte(i+1)||byteorder.BEUint32(p.buffer[36:])!=1){t.Fatal("completion bytes changed")}
 }
 if gomadStreamPendingTail!=pending[257] {t.Fatal("tail not removed")}
 if gomadStreamNextRequest!=259 {t.Fatal("probe consumed request identities")}
 if witnesses!=len(probes)+len(polls) {t.Fatal("missing boundary witness")}
 for p:=gomadStreamPendingHead;p!=nil;p=p.next {ready[byteorder.BEUint64(p.buffer[8:])]=true}
 if woken,waiting:=gomadStreamPollPending();!woken || waiting || gomadStreamPendingTail!=nil {t.Fatal("drain left pending requests")}
 calls:=len(probes)+len(polls)
 if woken,waiting:=gomadStreamPollPending();woken || waiting || calls!=len(probes)+len(polls){t.Fatal("empty scan crossed host boundary")}
}
func TestPollCompletionCascade(t *testing.T){
 pending:=setup(3);ready[1]=true;cascade=true
 if woken,waiting:=gomadStreamPollPending();!woken || !waiting {t.Fatal("cascade scan state changed")}
 if !reflect.DeepEqual(polls,[]uint64{1,2}) || pending[0].active || pending[1].active || !pending[2].active {t.Fatalf("same-scan completion cascade lost: %v",polls)}
}
func TestMalformedReadinessResponse(t *testing.T){
 for _,kind:=range []string{"id","count","status"}{t.Run(kind,func(t *testing.T){
  setup(1);malformed=kind
  defer func(){if recover()==nil {t.Fatal("malformed readiness response accepted")};if len(polls)!=0 {t.Fatal("malformed response consumed completion")}}()
  gomadStreamPollPending()
 })}
}
`+declarations.String()
 directory:=t.TempDir()
 if err:=os.WriteFile(filepath.Join(directory,"pending_test.go"),[]byte(source),0600);err!=nil {t.Fatal(err)}
 command:=exec.Command(filepath.Join(stockRoot(t),"bin/go"),"test","-count=1","pending_test.go")
 command.Dir=directory
 command.Env=append(os.Environ(),"GOROOT=","GOTOOLCHAIN=local","GOWORK=off","GOENV=off","GOFLAGS=")
 if output,err:=command.CombinedOutput();err!=nil {t.Fatalf("guest pending readiness: %v\n%s",err,output)}
}
