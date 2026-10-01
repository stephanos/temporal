package sprig_test

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	sprig "github.com/Masterminds/sprig/v3"
)

func TestHostLookupRefusesService(t *testing.T) {
	defer func() {
		if got := recover(); got != "gomad: Sprig host lookup is unsupported" {
			t.Fatalf("host lookup panic = %v, want explicit unsupported service", got)
		}
	}()
	sprig.TxtFuncMap()["getHostByName"].(func(string) string)("localhost")
}

func TestTemplateLookupReportsRefusal(t *testing.T) {
	tmpl, err := template.New("lookup").Funcs(sprig.TxtFuncMap()).Parse(`{{getHostByName "localhost"}}`)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err = tmpl.Execute(&output, nil)
	if err == nil || !strings.Contains(err.Error(), "gomad: Sprig host lookup is unsupported") {
		t.Fatalf("template lookup error = %v, output = %q", err, output.String())
	}
}

func TestOrdinaryTemplateFunctions(t *testing.T) {
	tmpl, err := template.New("ordinary").Funcs(sprig.TxtFuncMap()).Parse(`{{" sample " | trim | upper}}:{{add 2 3}}:{{list "a" "b" | join ","}}:{{default "fallback" ""}}`)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, nil); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "SAMPLE:5:a,b:fallback" {
		t.Fatalf("ordinary template = %q", got)
	}
}
