package record

import (
	"reflect"
	"strings"
	"testing"

	modelworld "go.temporal.io/server/tools/gomad3/world"
)

func TestOwnedWorldComposedManifestPreservesIdentity(t *testing.T) {
	core,err:=modelworld.New(modelworld.Config{Seed:1,Limits:modelworld.Limits{MaxRequests:8,MaxEvents:8,MaxQueuedEvents:8,MaxPayloadBytes:1<<20,MaxStringBytes:1<<16,MaxTransitions:64}})
	if err!=nil{t.Fatal(err)}
	recorder,err:=core.StartRecording(1<<20);if err!=nil{t.Fatal(err)}
	recording,err:=recorder.FinishError(&modelworld.CapacityError{Dimension:"雪\"\n"+strings.Repeat("x",8192),Limit:7,Used:6,Delta:2});if err!=nil{t.Fatal(err)}
	initial,err:=modelworld.EncodeSnapshot(recording.Initial);if err!=nil{t.Fatal(err)}
	final,err:=modelworld.EncodeSnapshot(recording.Final);if err!=nil{t.Fatal(err)}
	transitions,err:=modelworld.EncodeTransitions(recording.Final.Transitions);if err!=nil{t.Fatal(err)}
	input:=manifestFixture()
	input.World=World{
		Initial:WorldPayload{Schema:"gomad3.world.snapshot/v1",File:"world/snapshot.json",RawSHA256:HashBytes(initial),SemanticDigest:SHA256("sha256:"+string(recording.Initial.StateDigest))},
		Transitions:WorldTransitions{Schema:"gomad3.world.transitions/v1",File:"world/transitions.jsonl",RawSHA256:HashBytes(transitions),Count:Uint64String(len(recording.Final.Transitions)),TranscriptDigest:SHA256("sha256:"+string(recording.Final.TranscriptDigest))},
		Final:WorldPayload{Schema:"gomad3.world.snapshot/v1",File:"world/final-snapshot.json",RawSHA256:HashBytes(final),SemanticDigest:SHA256("sha256:"+string(recording.Final.StateDigest))},
		Adapters:[]WorldAdapter{},Terminal:WorldTerminal{Kind:string(recording.Terminal.Kind),Detail:recording.Terminal.Detail},
	}
	for i,file:=range input.Files{var data []byte;switch file.Path{case input.World.Initial.File:data=initial;case input.World.Transitions.File:data=transitions;case input.World.Final.File:data=final;default:continue};input.Files[i].Size=Uint64String(len(data));input.Files[i].SHA256=HashBytes(data)}
	value,encoded:=finalizedManifest(t,input)
	t.Logf("record=%s failure=%s bytes=%s",value.RecordHash,value.Outcome.FailureSignature,HashBytes(encoded))
	decoded,err:=DecodeExecutionRecord(encoded);if err!=nil{t.Fatal(err)}
	if !reflect.DeepEqual(decoded.World,input.World){t.Fatal("composed World terminal graph changed")}
}
