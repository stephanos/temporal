package deterministicio

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

func TestPortableAdapterSourcePreservation(t *testing.T) {
	_, cache := portableAdapterGo(t)
	for _, adapter := range rewrittenModuleAdapters {
		t.Run(adapter.name, func(t *testing.T) {
			moduleRoot := portableAdapterModule(t, cache, adapter.module, adapter.version)
			identity := gomadversion.AdapterIdentity{Module: adapter.module, Version: adapter.version, Sum: adapter.sum}
			prepared, err := adapter.prepare(cache, t.TempDir(), identity)
			if err != nil {
				t.Fatal(err)
			}
			if prepared.evidence.PreparedPackage != adapter.preparedPackage || prepared.evidence.PreparedSourceSetSHA256 != adapter.preparedSourceSetSHA256 || prepared.evidence.SourceSHA256 != adapter.rewrites[0].sourceSHA256 || prepared.evidence.ReplacementSHA256 != adapter.rewrites[0].replacementSHA256 {
				t.Fatalf("prepared package evidence = %+v", prepared.evidence)
			}
			var rewritten []byte
			for _, rewrite := range adapter.rewrites {
				contents, err := os.ReadFile(filepath.Join(prepared.replacement, filepath.FromSlash(rewrite.path)))
				if err != nil {
					t.Fatal(err)
				}
				rewritten = append(rewritten, contents...)
				if adapter.cacheDir == "" {
					continue
				}
				t.Run(rewrite.path, func(t *testing.T) {
					source, err := readAdapterSource(adapter.module, moduleRoot, rewrite.path)
					if err != nil {
						t.Fatal(err)
					}
					if !adapter.rewriteDropsComments {
						comments := func(contents []byte) []string {
							file, err := parser.ParseFile(token.NewFileSet(), rewrite.path, contents, parser.ParseComments)
							if err != nil {
								t.Fatal(err)
							}
							var texts []string
							for _, group := range file.Comments {
								for _, comment := range group.List {
									texts = append(texts, comment.Text)
								}
							}
							return texts
						}
						if !reflect.DeepEqual(comments(source), comments(contents)) {
							t.Fatal("rewrite changed original comments")
						}
					}
					for _, test := range []struct {
						name, want string
						change     func(*sourceRewrite, *[]byte)
					}{
						{name: "source drift", want: "source identity mismatch", change: func(_ *sourceRewrite, source *[]byte) { *source = append(*source, '\n') }},
						{name: "missing anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) {
							rewrite.rewrites[0].anchor = []byte("absent Gomad adapter anchor")
						}},
						{name: "ambiguous anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite, _ *[]byte) { rewrite.rewrites[0].anchor = []byte(adapter.ambiguousAnchor) }},
						{name: "replacement drift", want: "replacement identity mismatch", change: func(changed *sourceRewrite, _ *[]byte) { changed.replacementSHA256 = rewrite.sourceSHA256 }},
					} {
						t.Run(test.name, func(t *testing.T) {
							changed, input := rewrite, slices.Clone(source)
							changed.rewrites = slices.Clone(rewrite.rewrites)
							test.change(&changed, &input)
							if _, err := rewriteAdapterSource(adapter.module, changed, input); err == nil || !strings.Contains(err.Error(), test.want) {
								t.Fatalf("rewrite error = %v, want %s", err, test.want)
							}
						})
					}
				})
			}
			for _, removed := range adapter.removed {
				if bytes.Contains(rewritten, []byte(removed)) {
					t.Fatalf("rewritten source retains %q", removed)
				}
			}
			for _, retained := range adapter.retained {
				if !bytes.Contains(rewritten, []byte(retained)) {
					t.Fatalf("rewritten source omits %q", retained)
				}
			}
			if adapter.unrewritten != "" {
				original, err := readAdapterSource(adapter.module, moduleRoot, adapter.unrewritten)
				if err != nil {
					t.Fatal(err)
				}
				copied, err := readAdapterSource(adapter.module, prepared.replacement, adapter.unrewritten)
				if err != nil || !bytes.Equal(original, copied) {
					t.Fatalf("unrewritten %s changed: %v", adapter.unrewritten, err)
				}
				privateCache := t.TempDir()
				privateModule := filepath.Join(privateCache, filepath.FromSlash(adapter.cacheDir+"@"+adapter.version))
				copyFixtureTree(t, moduleRoot, privateModule)
				editFixtureFile(t, filepath.Join(privateModule, filepath.FromSlash(adapter.unrewritten)), func(contents []byte) []byte { return append(contents, '\n') })
				if _, err := adapter.prepare(privateCache, t.TempDir(), identity); err == nil || !strings.Contains(err.Error(), "source inventory identity mismatch") {
					t.Fatalf("changed unrewritten inventory = %v", err)
				}
			}
			identities := []gomadversion.AdapterIdentity{{Module: adapter.module, Version: adapter.version, Sum: "h1:changed"}}
			if adapter.otherModule != "" {
				identities = append(identities, gomadversion.AdapterIdentity{Module: adapter.otherModule, Version: adapter.version, Sum: adapter.sum}, gomadversion.AdapterIdentity{Module: adapter.module, Version: adapter.otherVersion, Sum: adapter.sum})
			}
			for _, cache := range []string{t.TempDir(), cache} {
				for _, changed := range identities {
					if _, err := adapter.prepare(cache, t.TempDir(), changed); err == nil || !strings.Contains(err.Error(), "identity mismatch") {
						t.Fatalf("changed identity %+v = %v", changed, err)
					}
				}
			}
		})
	}
}
