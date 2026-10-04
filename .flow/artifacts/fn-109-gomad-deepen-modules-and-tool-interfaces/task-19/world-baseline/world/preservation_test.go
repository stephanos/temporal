package world
import("bytes";"crypto/sha256";"reflect";"strings";"testing")
func testLimits()Limits{return Limits{MaxRequests:100,MaxEvents:100,MaxQueuedEvents:100,MaxTransitions:500,MaxPayloadBytes:1<<20,MaxStringBytes:1024}}
func newTestWorld(t *testing.T,seed Seed)*Model{t.Helper();w,err:=New(Config{Seed:seed,Limits:testLimits()});if err!=nil{t.Fatal(err)};return w}
func TestOwnedTerminalPreservesAllCategoriesAndRecordingBytes(t *testing.T){
	for _,test:=range []struct{input error;kind TerminalKind;detail string}{
		{ErrInvalidConfig,TerminalInvalidInput,"invalid World config"},{ErrInvalidRequest,TerminalInvalidInput,"invalid World request"},{ErrUnknownRequest,TerminalInvalidInput,"unknown World request"},{ErrRequestState,TerminalInvalidInput,"invalid World request state"},{ErrTimeRegression,TerminalInvalidInput,"World time regression"},{ErrInvalidSnapshot,TerminalInvalidInput,"invalid World snapshot"},{ErrCapacity,TerminalCapacity,"World capacity exhausted"},{ErrReplayDivergence,TerminalReplayDivergence,"World replay divergence"},
		{&CapacityError{Dimension:"雪\"\n"+strings.Repeat("x",8192),Limit:7,Used:6,Delta:2},TerminalCapacity,"World capacity exhausted: 雪\"\n"+strings.Repeat("x",8192)+" limit=7 used=6 delta=2"},
		{&ReplayDivergenceError{Index:5,ExpectedKind:"ready",ActualKind:"cancel",Field:"digest",ExpectedDigest:Digest(strings.Repeat("a",64)),ActualDigest:Digest(strings.Repeat("b",64))},TerminalReplayDivergence,"World replay divergence: transition=5 field=digest expected-kind=ready actual-kind=cancel expected-digest="+strings.Repeat("a",64)+" actual-digest="+strings.Repeat("b",64)},
	}{t.Run(test.detail[:min(len(test.detail),50)],func(t *testing.T){
		w:=newTestWorld(t,1);recorder,err:=w.StartRecording(1<<20);if err!=nil{t.Fatal(err)};before:=w.Snapshot()
		actual,err:=recorder.FinishError(test.input);if err!=nil{t.Fatal(err)}
		want:=Recording{Initial:before,Final:before,Terminal:Terminal{Kind:test.kind,Detail:test.detail}}
		if !reflect.DeepEqual(actual,want){t.Fatalf("recording changed: terminal=%+v",actual.Terminal)}
		actualBytes,err:=EncodeRecording(actual);if err!=nil{t.Fatal(err)};wantBytes,err:=EncodeRecording(want);if err!=nil{t.Fatal(err)};if !bytes.Equal(actualBytes,wantBytes){t.Fatal("complete recording bytes changed")};t.Logf("terminal=%s detail-bytes=%d recording-sha256=%x",test.kind,len(test.detail),sha256.Sum256(actualBytes))
		decoded,err:=DecodeRecording(actualBytes);if err!=nil{t.Fatal(err)};if !reflect.DeepEqual(decoded,want){t.Fatal("decoded recording changed")}
	})}
}
