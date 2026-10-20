package wasi

import (
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

func TestHelperInputTapeAttestsNextExpected(t *testing.T) {
	t.Parallel()
	for _,name := range []string{"valid-next","valid-end","valid-record","count","digest","cursor","missing-next","unexpected-next","ordinal","kind","alternatives","selected","site","set-digest","selected-identity","rank-override","valid-rank"} {
		t.Run(name,func(t *testing.T) {
			t.Parallel()
			request := &backends.ChoiceRequest{Mode:choice.ModeReplay,TapeSHA256:[32]byte{1},Decisions:1}
			next := &choice.Decision{Ordinal:0,Kind:choice.KindRunnable,Alternatives:2,Selected:0,SelectedIdentity:[32]byte{2},AlternativeSetDigest:[32]byte{3}}
			model := &helperModelResult{TapeSHA256:"0100000000000000000000000000000000000000000000000000000000000000",TapeRecords:1,NextExpected:next}
			valid := false
			switch name {
			case "valid-next": valid=true
			case "valid-end": model.Cursor=1;model.NextExpected=nil;valid=true
			case "valid-record": request.Mode=choice.ModeRecord;model.TapeRecords=0;model.TapeSHA256="";model.NextExpected=nil;valid=true
			case "count": model.TapeRecords=2
			case "digest": model.TapeSHA256="0000000000000000000000000000000000000000000000000000000000000000"
			case "cursor": model.Cursor=2
			case "missing-next": model.NextExpected=nil
			case "unexpected-next": model.Cursor=1
			case "ordinal": next.Ordinal=1
			case "kind": next.Kind=choice.KindSelectResult
			case "alternatives": next.Alternatives=1
			case "selected": next.Selected=2
			case "site": next.SiteMissing=true;next.SiteOffset=42
			case "set-digest": next.AlternativeSetDigest=[32]byte{}
			case "selected-identity": next.SelectedIdentity=[32]byte{}
			case "rank-override": next.RankOverride=true;next.SelectedIdentity=[32]byte{}
			case "valid-rank": request.Mode=choice.ModePrefix;request.PrefixControl=&backends.ChoicePrefixControl{Cutoff:1,Override:&backends.ChoiceRankOverride{Ordinal:0,Rank:1}};next.RankOverride=true;next.Selected=1;next.SelectedIdentity=[32]byte{};valid=true
			}
			if err := validateHelperInputTape(model,request); (err == nil) != valid { t.Fatalf("terminal validation = %v, valid=%v",err,valid) }
		})
	}
}
