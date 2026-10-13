package softassert

import (
	"reflect"
	"testing"
)

func TestObserveStaticMessagesFromBothStreams(t *testing.T) {
	for _, test := range []struct {
		name, output string
		want         []string
	}{
		{"json", `{"msg":"failed assertion: invalid state","failed-assertion":true,"seed":7}`, []string{"invalid state"}},
		{"test logger json", `    fixture.go:42: {"msg":"failed assertion: invalid state","failed-assertion":true,"seed":7}`, []string{"invalid state"}},
		{"JSON nested tab whitespace", "{\"msg\":\"failed assertion: invalid {state} \",\"failed-assertion\":true,\"details\":\t{\"value\":1}}", []string{"invalid {state} "}},
		{"test logger JSON nested tab whitespace", "fixture.go:42: {\"msg\":\"failed assertion: invalid {state}\",\"details\":\t{\"value\":1}}", []string{"invalid {state}"}},
		{"JSON nested assertion tag is not root tag", "{\"msg\":\"ordinary error\",\"details\":\t{\"FailedAssertion\":true}}", nil},
		{"test logger JSON nested assertion tag is not root tag", "fixture.go:42: {\"msg\":\"ordinary error\",\"details\":\t{\"FailedAssertion\":true}}", nil},
		{"alternate shared tag", `{"msg":"invalid state","FailedAssertion":true,"seed":7}`, []string{"invalid state"}},
		{"console shared tag", "2000-01-01\terror\tinvalid state\t{\"failed-assertion\":true,\"seed\":7}", []string{"invalid state"}},
		{"tag", `{"message":"invalid state","failed-assertion":true,"seed":8}`, []string{"invalid state"}},
		{"console", "PASS\n2000-01-01\terror\tfailed assertion: invalid state\t{\"seed\":9}", []string{"invalid state"}},
		{"plain", "PASS\nfailed assertion: invalid state", []string{"invalid state"}},
		{"plain JSON literal", `failed assertion: invalid {"left":1}`, []string{`invalid {"left":1}`}},
		{"plain brace literal", `failed assertion: invalid {state}`, []string{`invalid {state}`}},
		{"plain JSON message key", `failed assertion: invalid {"msg":"ordinary error","failed-assertion":true}`, []string{`invalid {"msg":"ordinary error","failed-assertion":true}`}},
		{"console JSON literal", "2000-01-01\terror\tfailed assertion: invalid {\"left\":1}\t{\"failed-assertion\":true,\"seed\":9}", []string{`invalid {"left":1}`}},
		{"console brace literal", "2000-01-01\terror\tfailed assertion: invalid {state}\t{\"failed-assertion\":true,\"seed\":9}", []string{`invalid {state}`}},
		{"console tagged brace literal", "2000-01-01\terror\tinvalid {state}\t{\"failed-assertion\":true,\"seed\":9}", []string{`invalid {state}`}},
		{"console JSON message key", "2000-01-01\terror\tfailed assertion: invalid {\"msg\":\"ordinary error\"}\t{\"failed-assertion\":true}", []string{`invalid {"msg":"ordinary error"}`}},
		{"test logger console brace literal", "fixture.go:42: 2000-01-01\terror\tfailed assertion: invalid {state}\t{\"failed-assertion\":true}", []string{`invalid {state}`}},
		{"console trailing message space", "2000-01-01\terror\tfailed assertion: invalid state \t{\"failed-assertion\":true}", []string{"invalid state "}},
		{"unrelated", `{"msg":"ordinary error","failed-assertion":false}`, nil},
		{"dynamic tag", `{"msg":"ordinary error","details":"failed assertion: quoted text"}`, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Observe([]byte(test.output), nil, true)
			if !got.Complete || !reflect.DeepEqual(got.Messages, test.want) {
				t.Fatalf("observation = %+v, want %v", got, test.want)
			}
		})
	}
	got := Observe([]byte("failed assertion: z\nfailed assertion: a\n"), []byte("failed assertion: z\n"), true)
	if !reflect.DeepEqual(got.Messages, []string{"a", "z"}) {
		t.Fatalf("messages were not sorted and deduplicated: %+v", got)
	}
}

func TestObserveIncompleteEvidenceNeverPasses(t *testing.T) {
	for _, output := range [][]byte{
		{0xff},
		[]byte(`{"msg":"failed assertion: broken`),
		[]byte(`{"failed-assertion":true}`),
		[]byte(`{"msg":"failed assertion: broken","failed-assertion":"true"}`),
		[]byte("2000-01-01\terror\tfailed assertion: invalid {state}\t{\"failed-assertion\":true"),
		[]byte("failed assertion: "),
	} {
		if got := Observe(output, nil, true); got.Complete {
			t.Fatalf("unreadable assertion evidence passed: %q %+v", output, got)
		}
	}
	if got := Observe([]byte("PASS\n"), nil, false); got.Complete {
		t.Fatal("truncated evidence passed")
	}
}
