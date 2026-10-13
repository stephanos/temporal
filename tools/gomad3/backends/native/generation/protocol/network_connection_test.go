package protocol

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNetworkConnectionPortableEquivalence(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	overlay := filepath.Join(root, "backends/native/toolchain/runtime/overlay/src")
	directory := t.TempDir()
	write := func(name string, source []byte) {
		t.Helper()
		path := filepath.Join(directory, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, source, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", []byte("module connection.test\n\ngo 1.27.1\n"))
	for _, name := range []string{"network.go", "network_connection.go", "simulation_network.go", "simulation_handles.go", "simulation_wire.go", "descriptor_backend.go"} {
		source, err := os.ReadFile(filepath.Join(overlay, "internal/gomadio", name))
		if os.IsNotExist(err) && name == "network_connection.go" {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		for _, owner := range []string{"gomadsim", "gomadvfd", "poll"} {
			source = bytes.ReplaceAll(source, []byte(`"internal/`+owner+`"`), []byte(`"connection.test/`+owner+`"`))
		}
		lines := strings.Split(string(source), "\n")
		for index, line := range lines {
			if strings.HasPrefix(line, "//go:linkname") {
				lines[index] = ""
			}
		}
		write("gomadio/"+name, []byte(strings.Join(lines, "\n")))
	}
	write("poll/poll.go", []byte("package poll\nimport \"errors\"\nvar ErrNetClosing = errors.New(\"use of closed network connection\")\n"))
	write("gomadsim/domain.go", []byte(connectionDomainFixture))
	write("gomadvfd/vfd.go", []byte("package gomadvfd\n"+networkTimeoutDeclarations(t, filepath.Join(overlay, "internal/gomadvfd/descriptor.go"), "Status OK Address Notice")+connectionNotifierFixture))
	write("gomadio/effects.go", []byte(connectionEffectFixture))
	write("gomadio/equivalence_test.go", []byte(connectionEquivalenceTests))
	goExecutable := os.Getenv("GOMAD3_STOCK_GO")
	if goExecutable == "" {
		var err error
		goExecutable, err = exec.LookPath("go")
		if err != nil {
			t.Fatal(err)
		}
	}
	command := exec.CommandContext(t.Context(), goExecutable, "test", "-tags", "test_dep", "-count=1", "-timeout=30s", "./...")
	command.Dir = directory
	command.Env = append(os.Environ(), "GOROOT=", "GOWORK=off", "GOENV=off", "GOFLAGS=", "GOTOOLCHAIN=local", "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("portable actual-source connections: %v\n%s", err, output)
	}
}

const connectionDomainFixture = `package gomadsim
type NetworkDomain struct { Token, Run, Incarnation uint64; Node, Address string }
var Current NetworkDomain
var Domains = map[uint64]NetworkDomain{}
func ProcessRole() int { return 0 }
func CurrentNetworkDomain() (NetworkDomain,error,bool) { return Current,nil,Current.Run!=0 }
func DescribeNetworkDomain(token uint64) (NetworkDomain,bool) { d,ok:=Domains[token]; return d,ok }
`

const connectionNotifierFixture = `
var Notices []Notice
func Notify(notices ...Notice) { Notices=append(Notices,notices...) }
func RegisterBackend(any) Status { return OK }
func Refuse(string) Status { return Unsupported }
`

const connectionEffectFixture = `package gomadio
import("context";"time")
func record(string,[]byte,[]byte,uint64,uint64,uint64,uint64) {}
func resultClass(error) uint64 { return 0 }
func networkArguments(string,...int) []byte { return nil }
func processNetworkListen(string,string,int) (*Listener,error) { return nil,ErrUnsupported }
func processNetworkDial(context.Context,string,string,int) (*Conn,error) { return nil,ErrUnsupported }
func revokeProcessNetworkResources(uint64) {}
var _ = time.Time{}
`

const connectionEquivalenceTests = `package gomadio
import("bytes";"context";"errors";"io";"os";"reflect";"testing";"time";"connection.test/gomadsim";"connection.test/gomadvfd")

func publicPair(t *testing.T, simulated bool) (*Conn,*Conn,func(bool)) {
 t.Helper()
 gomadsim.Current=gomadsim.NetworkDomain{}
 set:=func(bool) {}
 host:="127.0.0.1"
 if simulated {
  config:=simulationNetworkConfig{Nodes:[]simulationNodeConfig{{Node:"a",Address:"10.0.0.1"},{Node:"b",Address:"10.0.0.2"}},Links:[]simulationLink{{From:"a",To:"b",Enabled:true},{From:"b",To:"a",Enabled:true}},Limits:simulationLimits{Listeners:8,Connections:8,Bytes:16<<20,ConnectionBytes:4<<20,StallNanos:10_000_000_000,Transitions:16384}}
  n,err:=newSimulationNetwork(1,config);if err!=nil{t.Fatal(err)}
  simulationNetworkState.runs[1]=n
  set=func(client bool){if client {gomadsim.Current=gomadsim.NetworkDomain{Run:1,Node:"a",Address:"10.0.0.1",Incarnation:1}} else {gomadsim.Current=gomadsim.NetworkDomain{Run:1,Node:"b",Address:"10.0.0.2",Incarnation:1}}}
  set(false);host="10.0.0.2"
 }
 l,err:=ListenTCP("tcp4",host,0);if err!=nil{t.Fatal(err)}
 t.Cleanup(func(){set(false);_ = l.Close();gomadsim.Current=gomadsim.NetworkDomain{}})
 set(true);c,err:=DialTCP(context.Background(),"tcp4",host,l.Address().Port);if err!=nil{t.Fatal(err)}
 set(false);s,err:=l.Accept();if err!=nil{t.Fatal(err)}
 return c,s,set
}

func TestNetworkHandleOperationParity(t *testing.T) {
 for _,simulated:=range []bool{false,true} {t.Run(map[bool]string{false:"standalone",true:"simulation"}[simulated],func(t *testing.T){
  c,s,set:=publicPair(t,simulated)
  var got []string
  if c.LocalAddress()!=s.RemoteAddress()||c.RemoteAddress()!=s.LocalAddress(){t.Fatal("address pair")}
  set(true);if n,err:=c.Write([]byte("abcdef"));n!=6||err!=nil{t.Fatalf("write=%d,%v",n,err)}
  if err:=c.CloseWrite();err!=nil{t.Fatal(err)}
  set(false);if err:=s.SetReadDeadline(time.Now().Add(-time.Second));err!=nil{t.Fatal(err)}
  for _,size:=range []int{2,3,4} {p:=make([]byte,size);n,err:=s.Read(p);if err!=nil{t.Fatal(err)};got=append(got,string(p[:n]))}
  if !reflect.DeepEqual(got,[]string{"ab","cde","f"}){t.Fatalf("short reads=%v",got)}
  if n,err:=s.Read(make([]byte,1));n!=0||err!=io.EOF{t.Fatalf("EOF precedes deadline=%d,%v",n,err)}
  if n,err:=s.Read(nil);n!=0||err!=nil{t.Fatalf("empty=%d,%v",n,err)}
  set(true);_ = c.SetReadDeadline(time.Now().Add(-time.Second))
  if n,err:=c.Read(make([]byte,1));n!=0||!errors.Is(err,os.ErrDeadlineExceeded){t.Fatalf("deadline=%d,%v",n,err)}
  _ = c.SetReadDeadline(time.Time{})
  set(false);if n,err:=s.Write([]byte("reply"));n!=5||err!=nil{t.Fatalf("reverse=%d,%v",n,err)}
  set(true);p:=make([]byte,5);if n,err:=c.Read(p);n!=5||err!=nil||string(p)!="reply"{t.Fatalf("recover=%d,%v,%q",n,err,p)}
 })}
}

func TestNetworkHandleByteCapacityAndUnreadTail(t *testing.T) {
 for _,simulated:=range []bool{false,true}{t.Run(map[bool]string{false:"standalone",true:"simulation"}[simulated],func(t *testing.T){
  c,s,set:=publicPair(t,simulated);set(true)
  _ = c.SetWriteDeadline(time.Now().Add(-time.Second))
  for index:=0;index<65;index++{if n,err:=c.Write([]byte{'x'});n!=1||err!=nil{t.Fatalf("tiny write %d=%d,%v",index,n,err)}}
  payload:=bytes.Repeat([]byte{'y'},(4<<20)-65+1)
  if n,err:=c.Write(payload);n!=(4<<20)-65||!errors.Is(err,os.ErrDeadlineExceeded){t.Fatalf("4MiB prefix=%d,%v",n,err)}
  set(false);p:=make([]byte,1);if n,err:=s.Read(p);n!=1||err!=nil{t.Fatalf("consume=%d,%v",n,err)}
  set(true);if n,err:=c.Write([]byte("ab"));n!=1||!errors.Is(err,os.ErrDeadlineExceeded){t.Fatalf("tail stays charged=%d,%v",n,err)}
 })}
}

func TestNetworkHandleDescriptorPrimitiveParity(t *testing.T) {
 gomadsim.Current=gomadsim.NetworkDomain{}
 b:=descriptorBackend;l,status:=b.Socket();if status!=gomadvfd.OK{t.Fatal(status)}
 b.Attach(l,7,19)
 if status:=b.Bind(l,gomadvfd.Address{IP:[4]byte{127,0,0,1}});status!=gomadvfd.OK{t.Fatal(status)}
 if status:=b.Bind(l,gomadvfd.Address{});status!=gomadvfd.Invalid{t.Fatalf("duplicate bind=%v",status)}
 if status:=b.Listen(l,4);status!=gomadvfd.OK{t.Fatal(status)}
 address,status:=b.Local(l);if status!=gomadvfd.OK{t.Fatal(status)}
 c,status:=b.Socket();if status!=gomadvfd.OK{t.Fatal(status)};b.Attach(c,8,20)
 if status:=b.Connect(c,address);status!=gomadvfd.OK{t.Fatal(status)}
 s,_,status:=b.Accept(l);if status!=gomadvfd.OK{t.Fatal(status)};b.Attach(s,9,21)
 if n,status:=b.Read(s,make([]byte,1));n!=0||status!=gomadvfd.WouldBlock{t.Fatalf("empty readiness=%d,%v",n,status)}
 if n,status:=b.Write(c,[]byte("abc"));n!=3||status!=gomadvfd.OK{t.Fatalf("write=%d,%v",n,status)}
 if status:=b.Shutdown(c,1);status!=gomadvfd.OK{t.Fatal(status)}
 p:=make([]byte,2);if n,status:=b.Read(s,p);n!=2||status!=gomadvfd.OK||string(p)!="ab"{t.Fatalf("short=%d,%v,%q",n,status,p)}
 if n,status:=b.Read(s,p);n!=1||status!=gomadvfd.OK||p[0]!='c'{t.Fatalf("tail=%d,%v",n,status)}
 if n,status:=b.Read(s,p);n!=0||status!=gomadvfd.EndOfStream{t.Fatalf("EOF=%d,%v",n,status)}
 found:=false;for _,notice:=range gomadvfd.Notices{if notice.FD==9&&notice.Generation==21&&notice.Mode=='r'{found=true}};if !found{t.Fatal("generation-bound read notice missing")}
 _ = b.Close(c);_ = b.Close(s);_ = b.Close(l)
}

func TestNetworkHandleCommittedTerminalTimeout(t *testing.T) {
 for _,simulated:=range []bool{false,true}{t.Run(map[bool]string{false:"standalone",true:"simulation"}[simulated],func(t *testing.T){
  c,s,set:=publicPair(t,simulated)
  var state *connState
  switch implementation:=c.implementation.(type){case *standaloneConn:state=implementation.state;case *simulationConn:state=implementation.state;default:t.Fatal("nonlocal owner")}
  state.terminal=ErrStallTimeout
  for index,connection:=range []*Conn{c,s}{set(index==0);_ = connection.SetDeadline(time.Time{})
   for _,op:=range []func()(int,error){func()(int,error){return connection.Read(make([]byte,1))},func()(int,error){return connection.Write([]byte("x"))}}{
    n,err:=op();timeout,ok:=err.(interface{Timeout()bool});if n!=0||!errors.Is(err,ErrStallTimeout)||errors.Is(err,os.ErrDeadlineExceeded)||!ok||!timeout.Timeout(){t.Fatalf("terminal=%d,%v",n,err)}
   }
  }
 })}
}

func TestNetworkHandleDescriptorByteCapacityAndUnreadTail(t *testing.T) {
 gomadsim.Current=gomadsim.NetworkDomain{}
 b:=descriptorBackend;l,status:=b.Socket();if status!=gomadvfd.OK{t.Fatal(status)}
 if status:=b.Bind(l,gomadvfd.Address{IP:[4]byte{127,0,0,1}});status!=gomadvfd.OK{t.Fatal(status)}
 if status:=b.Listen(l,4);status!=gomadvfd.OK{t.Fatal(status)}
 address,status:=b.Local(l);if status!=gomadvfd.OK{t.Fatal(status)}
 c,status:=b.Socket();if status!=gomadvfd.OK{t.Fatal(status)}
 if status:=b.Connect(c,address);status!=gomadvfd.OK{t.Fatal(status)}
 s,_,status:=b.Accept(l);if status!=gomadvfd.OK{t.Fatal(status)}
 defer b.Close(c);defer b.Close(s);defer b.Close(l)
 for index:=0;index<65;index++{if n,status:=b.Write(c,[]byte{'x'});n!=1||status!=gomadvfd.OK{t.Fatalf("tiny descriptor write %d=%d,%v",index,n,status)}}
 for remaining:=(4<<20)-65;remaining>0;{n,status:=b.Write(c,make([]byte,min(remaining,maximumChunkBytes)));if n==0||status!=gomadvfd.OK{t.Fatalf("fill=%d,%v",n,status)};remaining-=n}
 if n,status:=b.Write(c,[]byte{'z'});n!=0||status!=gomadvfd.WouldBlock{t.Fatalf("byte-full=%d,%v",n,status)}
 if n,status:=b.Read(s,make([]byte,1));n!=1||status!=gomadvfd.OK{t.Fatalf("short read=%d,%v",n,status)}
 if n,status:=b.Write(c,[]byte("ab"));n!=1||status!=gomadvfd.OK{t.Fatalf("unread tail charge=%d,%v",n,status)}
 if n,status:=b.Write(c,[]byte{'z'});n!=0||status!=gomadvfd.WouldBlock{t.Fatalf("tail-byte-full=%d,%v",n,status)}
}

func TestNetworkHandleAutonomousStallSourceTimer(t *testing.T) {
 c,s,set:=publicPair(t,true)
 local:=c.implementation.(*simulationConn)
 n:=local.network;n.limits.StallNanos=1_000_000;n.limits.Transitions=6
 if err:=n.changeLink("a","b","disconnect",false,0);err!=nil{t.Fatal(err)}
 set(true);if count,err:=c.Write([]byte("held"));count!=4||err!=nil{t.Fatalf("held=%d,%v",count,err)}
 n.Lock();local.state.shared.Lock();changed:=local.state.shared.changed;expired:=local.state.terminal!=nil;local.state.shared.Unlock();n.Unlock()
 timer:=time.NewTimer(time.Second);defer timer.Stop()
 if !expired { select{case <-changed:case <-timer.C:t.Fatal("returned writer did not expire autonomously")} }
 for index,connection:=range []*Conn{c,s}{set(index==0);_,err:=connection.Read(make([]byte,1));if !errors.Is(err,ErrStallTimeout){t.Fatalf("autonomous terminal=%v",err)}}
 if n.pendingBytes!=0||n.historyReserved!=0||len(n.transitions)!=6{t.Fatal("terminal reservation or bytes leaked")}
}

func TestNetworkHandlePartitionedDialSourceTimer(t *testing.T) {
 c,s,set:=publicPair(t,true);n:=c.implementation.(*simulationConn).network;n.limits.StallNanos=1_000_000
 if err:=n.changeLink("a","b","disconnect",false,0);err!=nil{t.Fatal(err)}
 set(true);ctx,cancel:=context.WithTimeout(context.Background(),time.Second);defer cancel()
 if connection,err:=DialTCP(ctx,"tcp4",s.LocalAddress().IP,s.LocalAddress().Port);connection!=nil||!errors.Is(err,ErrStallTimeout){t.Fatalf("dial stall=%v,%v",connection,err)}
 if len(n.pendingDials)!=0||n.historyReserved!=0||n.nextConnection!=1{t.Fatal("pending dial leaked work or connection identity")}
 cancelled,stop:=context.WithCancel(context.Background());stop()
 if connection,err:=DialTCP(cancelled,"tcp4",c.LocalAddress().IP,c.LocalAddress().Port);connection!=nil||!errors.Is(err,context.Canceled){t.Fatalf("completed context=%v,%v",connection,err)}
}

func TestNetworkHandlePartialWriteSourceTimer(t *testing.T) {
 c,_,set:=publicPair(t,true);n:=c.implementation.(*simulationConn).network
 n.limits.ConnectionBytes=2;n.limits.StallNanos=1_000_000;n.limits.Transitions=6
 if err:=n.changeLink("a","b","disconnect",false,0);err!=nil{t.Fatal(err)}
 set(true);if err:=c.SetWriteDeadline(time.Now().Add(time.Second));err!=nil{t.Fatal(err)}
 count,err:=c.Write([]byte("prefix"));timeout,ok:=err.(interface{Timeout()bool})
 if count!=2||!errors.Is(err,ErrStallTimeout)||!ok||!timeout.Timeout(){t.Fatalf("partial stall=%d,%v",count,err)}
 if n.pendingBytes!=0||n.historyReserved!=0||len(n.transitions)!=6{t.Fatal("partial timeout leaked work")}
}
`
