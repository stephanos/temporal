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

func TestDebugDecisionStopsBeforeApplyingEveryDecisionKind(t *testing.T) {
	wanted := map[string]bool{"gomadChoiceDecision": true, "gomadChoiceAppendRecord": true, "gomadDebugDecision": true, "gomadDebugWitnessOnly": true, "gomadDebugCaptureHeap": true, "gomadDebugHeader": true, "gomadDebugPut32": true, "gomadDebugRead32": true}
	var functions bytes.Buffer
	for _, source := range [][]byte{runtimeGlue, runtimeInspectGlue} {
		file, err := parser.ParseFile(token.NewFileSet(), "runtime_wasm.go.txt", source, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || !wanted[function.Name.Name] {
				continue
			}
			delete(wanted, function.Name.Name)
			if err := format.Node(&functions, token.NewFileSet(), function); err != nil {
				t.Fatal(err)
			}
			functions.WriteByte('\n')
		}
	}
	if len(wanted) != 0 {
		t.Fatalf("missing runtime predecision control: %v", wanted)
	}
	source := `package boundary
import ("bytes"; "encoding/binary"; "sync/atomic"; "testing")
const gomadChoiceMaximumAlternatives=256
const gomadDiagnosticRecordBytes=96
const gomadDebugWitnessBytes=128
const gomadChoiceFlagSiteMissing=4
const gomadChoiceKindRunnable=1
const gomadChoiceDivergenceAlternativeCapacity=1
const gomadChoiceDivergenceSelected=2
var gomadChoiceRecords atomic.Uint64
var gomadDebugDecisions uint64
var gomadDiagnosticEnabled bool
var gomadDebugInspecting bool
var gomadChoiceSchedulerSites [256]uint64
var gomadChoiceSchedulerSiteFlags [256]uint8
var gomadWasmDecisionBuffer [32+48*256+96]byte
const gomadChoiceRecordBytes=96
type gomadChoiceRecordValue struct{ordinal uint64}
var gomadWasmObservationBuffer [96+96]byte
func gomadChoiceEncodeRecord(b []byte,value gomadChoiceRecordValue){binary.BigEndian.PutUint64(b,value.ordinal)}
func gomadWasmObservation(b *byte,n uint32)uint32{return 0}
var witnessCalls int
func gomadDebugCall(op,length uint32){if op!=10 || length!=128 {panic("wrong witness exchange")};witnessCalls++}
var gomadDebugBuffer [16+32+32*256+128]byte
var gomadDebugPending struct{length uint32; decision [32+32*256]byte}
var stop bool
var canonicalCalls int
type MemProfileRecord struct{AllocObjects int64}
var gomadDebugHeapRecords [4096]MemProfileRecord
var gomadDebugHeapCount int
var gomadDebugHeapOK bool
var gomadDebugHeapRate int
var MemProfileRate=1234
var profileAllocations int64=4
func MemProfile(records []MemProfileRecord,all bool)(int,bool){records[0].AllocObjects=profileAllocations;return 1,true}
func gomadChoiceDivergeCurrent(uint8){panic("divergence")}
func gomadChoicePut32(b []byte,n uint32){binary.BigEndian.PutUint32(b,n)}
func gomadChoicePut64(b []byte,n uint64){binary.BigEndian.PutUint64(b,n)}
func gomadChoiceRead32(b []byte)uint32{return binary.BigEndian.Uint32(b)}
func gomadWasmDiagnostic(b []byte){clear(b)}
func gomadDebugWitness(b []byte){for i:=range b {b[i]=byte(i)}}
func gomadDebugHandoff(){panic("inspection")}
func gomadWasmDecision(b *byte,n uint32)uint32{
canonicalCalls++;gomadChoicePut32(gomadWasmDecisionBuffer[24:28],1);return 0
}

func gomadDebugExchange(b *byte,n uint32)uint32{
 if gomadDebugRead32(gomadDebugBuffer[:4])!=1 || gomadDebugRead32(gomadDebugBuffer[4:8])!=1 {panic("bad debug header")}
 size:=int(gomadDebugRead32(gomadDebugBuffer[12:16]))
 if int(n)!=16+size || size!=32+64+128 {panic("missing decision or compulsory witness")}
 if !bytes.Equal(gomadDebugBuffer[16:16+96],gomadWasmDecisionBuffer[:96]) {panic("changed pending decision")}
 if stop {gomadDebugPut32(gomadDebugBuffer[8:12],1);gomadChoicePut32(gomadDebugBuffer[40:44],1)}
 return 0
}
func TestBoundary(t *testing.T){
 for _,kind:=range []uint8{1,2} { for _,diagnostics:=range []bool{false,true} {
 gomadDiagnosticEnabled=diagnostics;gomadChoiceRecords.Store(9);gomadDebugDecisions=4;canonicalCalls=0
 alternatives:=[][32]byte{{1},{2}};var ordered [256][32]byte
 stop=false
 if got:=gomadChoiceDecision(kind,4,123,alternatives,&ordered,0,7);got!=1 {t.Fatalf("rank=%d",got)}
 expectedRecords:=uint64(10);if kind==1 {expectedRecords=12}
 if canonicalCalls!=1 || gomadChoiceRecords.Load()!=expectedRecords || gomadDebugDecisions!=5 {t.Fatal("continued decision counters wrong")}
 stop=true;canonicalCalls=0
 func(){defer func(){if recover()!="inspection"{t.Fatal("did not hand off")}}();gomadChoiceDecision(kind,4,123,alternatives,&ordered,0,7)}()
 if canonicalCalls!=0 || gomadChoiceRecords.Load()!=expectedRecords || gomadDebugDecisions!=5 {t.Fatal("stop consumed decision")}
 expected:=append([]byte(nil),gomadWasmDecisionBuffer[:96]...);gomadChoicePut32(expected[24:28],1)
 if gomadDebugPending.length!=96 || !bytes.Equal(gomadDebugPending.decision[:96],expected) {t.Fatal("validated pending decision lost")}
 }}
}
func TestObservationsDoNotShiftDecisionOrdinals(t *testing.T){
 gomadChoiceRecords.Store(11);gomadDebugDecisions=7;witnessCalls=0
 gomadChoiceAppendRecord(gomadChoiceRecordValue{})
 if gomadChoiceRecords.Load()!=12 || gomadDebugDecisions!=7 || witnessCalls!=1 {t.Fatal("observation shifted tape ordinal or omitted sealed witness")}
}
func TestHeapSnapshotPrecedesInspectionAllocations(t *testing.T){
 gomadDebugCaptureHeap();profileAllocations=99;MemProfileRate=1
 if !gomadDebugHeapOK || gomadDebugHeapCount!=1 || gomadDebugHeapRecords[0].AllocObjects!=4 || gomadDebugHeapRate!=1234 {t.Fatal("heap snapshot changed with inspection allocations")}
}
` + functions.String()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "boundary_test.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(filepath.Join(stockRoot(t), "bin/go"), "test", "-count=1", "boundary_test.go")
	command.Dir = directory
	command.Env = append(os.Environ(), "GOROOT=", "GOTOOLCHAIN=local", "GOWORK=off", "GOENV=off", "GOFLAGS=")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("runtime boundary: %v\n%s", err, output)
	}
}

func TestDebugNetworkHandoffSealsWitness(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "net_stream_wasm.go.txt", netStreamGlue, 0)
	if err != nil {
		t.Fatal(err)
	}
	var function bytes.Buffer
	for _, declaration := range file.Decls {
		if fn, ok := declaration.(*ast.FuncDecl); ok && fn.Name.Name == "gomadStreamInvoke" {
			if err := format.Node(&function, token.NewFileSet(), fn); err != nil {
				t.Fatal(err)
			}
		}
	}
	source := `package handoff
import("encoding/binary";"testing")
const gomadStreamHeader=64
var byteorder=struct{BEUint32 func([]byte)uint32}{binary.BigEndian.Uint32}
var witnessed bool
func gomadStreamWitness(){witnessed=true}
func gomadWasmNetStream(*byte,uint32)uint32{if !witnessed {panic("network yielded without current runtime witness")};return 0}
func TestHandoff(t *testing.T){gomadStreamInvoke(make([]byte,64))}
` + function.String()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "handoff_test.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(filepath.Join(stockRoot(t), "bin/go"), "test", "-count=1", "handoff_test.go")
	command.Dir = directory
	command.Env = append(os.Environ(), "GOROOT=", "GOTOOLCHAIN=local", "GOWORK=off", "GOENV=off", "GOFLAGS=")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("network handoff witness: %v\n%s", err, output)
	}
}
