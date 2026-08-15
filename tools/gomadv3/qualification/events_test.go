package qualification

import (
	"bytes"
	"strings"
	"testing"
)

func TestQualificationEventStreamReturnsRetainedResult(t *testing.T) {
	report := QualificationReport{Failure: &QualificationFailure{Classification: "unsupported_target"}}
	var stream bytes.Buffer
	if err := WriteProgressEvent(&stream, 1, 2); err != nil {
		t.Fatal(err)
	}
	if err := WriteResultEvent(&stream, report, "/tmp/qualification.json"); err != nil {
		t.Fatal(err)
	}
	result, err := DecodeResultEvent(stream.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if result.Classification != "unsupported_target" || result.ReportPath != "/tmp/qualification.json" {
		t.Fatalf("result event = %#v", result)
	}
	if ExitStatus(result.Classification) != 2 {
		t.Fatalf("ExitStatus() = %d", ExitStatus(result.Classification))
	}
}

func TestDecodeResultEventRejectsInvalidStreams(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		message  string
	}{
		{name: "missing result", contents: `{"schema":"gomadv3.qualify-event/v1","type":"progress"}` + "\n", message: "has no retained result"},
		{name: "unretained error", contents: `{"schema":"gomadv3.qualify-event/v1","type":"error","classification":"invalid_input","message":"bad input"}` + "\n", message: "unretained qualification error invalid_input: bad input"},
		{name: "unsupported schema", contents: `{"schema":"gomadv3.qualify-event/v2","type":"result"}` + "\n", message: "unsupported qualification event schema"},
		{name: "duplicate result", contents: `{"schema":"gomadv3.qualify-event/v1","type":"result","classification":"qualified","report_path":"a"}` + "\n" + `{"schema":"gomadv3.qualify-event/v1","type":"result","classification":"qualified","report_path":"b"}` + "\n", message: "invalid or duplicated"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeResultEvent([]byte(test.contents))
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("DecodeResultEvent() error = %v", err)
			}
		})
	}
}
