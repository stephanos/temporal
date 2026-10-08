package toolchain

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceAcceptanceMaterializedPreservation(t *testing.T) {
	root, archive, descriptor := pinnedArchive(t)
	candidate := materializePinnedSource(t, root, archive, "")
	if err := rejectOverlayCollisions(filepath.Join(root, "toolchain/runtime/overlay"), candidate); err != nil { t.Fatal(err) }
	contents, err := os.ReadFile("/Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-2/gfield-compact-20261005/preservation.json")
	if err != nil { t.Fatal(err) }
	var receipt struct { Files []struct { Path string; FinalSHA256 string `json:"final_sha256"` } }
	if err := json.Unmarshal(contents, &receipt); err != nil { t.Fatal(err) }
	if len(receipt.Files) != 21 || len(descriptor.PatchAllowlist) != 20 || len(descriptor.OverlayAllowlist) != 79 { t.Fatal("source closure inventory drift") }
	for _, file := range receipt.Files {
		name := filepath.Join(candidate, file.Path)
		if strings.HasPrefix(file.Path, "tools/") { name = filepath.Join(filepath.Dir(filepath.Dir(root)), file.Path) }
		data, err := os.ReadFile(name)
		if err != nil { t.Fatal(err) }
		if fmt.Sprintf("%x", sha256.Sum256(data)) != file.FinalSHA256 { t.Fatalf("preserved source drift: %s", file.Path) }
	}
	t.Log("real archive collision check: 79 overlay paths; exact retained materialized preservation: 20 patched files + runtime overlay")
}

func TestSourceAcceptanceSchedulerExtraction(t *testing.T) {
	root, archive, _ := pinnedArchive(t)
	repo := filepath.Dir(filepath.Dir(root))
	baseline := "5df49456e649d4d3d2c6d8ba6518a2a05ca41e26"
	baselineRoot := t.TempDir()
	for _, name := range []string{"toolchain/version/version.json", "toolchain/runtime/go1.27.1.patch"} {
		command := exec.Command("git", "show", baseline+":tools/gomad3/"+name)
		command.Dir = repo
		data, err := command.Output()
		if err != nil { t.Fatal(err) }
		destination := filepath.Join(baselineRoot, name)
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil { t.Fatal(err) }
		if err := os.WriteFile(destination, data, 0644); err != nil { t.Fatal(err) }
	}
	extracted := filepath.Join(t.TempDir(), "source")
	if err := ExtractSource(context.Background(), archive, extracted); err != nil { t.Fatal(err) }
	beforeRoot := filepath.Join(extracted, "go")
	if err := MaterializePatch(context.Background(), PatchSpec{Root: baselineRoot, SourceRoot: beforeRoot}); err != nil { t.Fatal(err) }
	afterRoot := materializePinnedSource(t, root, archive, "")
	read := func(name string) string { data, err := os.ReadFile(name); if err != nil { t.Fatal(err) }; return string(data) }
	before := read(filepath.Join(beforeRoot, "src/runtime/proc.go"))
	after := read(filepath.Join(afterRoot, "src/runtime/proc.go"))
	overlay := read(filepath.Join(root, "toolchain/runtime/overlay/src/runtime/gomad.go"))
	function := func(source, name string) string {
		start := strings.Index(source, "func "+name+"(")
		if start < 0 { t.Fatalf("missing function %s", name) }
		rest := source[start:]
		end := strings.Index(rest, "\n}\n")
		if end < 0 { t.Fatalf("missing function end %s", name) }
		return rest[:end+2]
	}
	alpha := func(source string) string { return strings.ReplaceAll(source, ".gomadSimulationTransport", ".gomadSimIO") }
	if alpha(function(before, "gomadSimulationTimeQuiescenceChanged")) != function(overlay, "gomadSimulationTimeQuiescenceChanged") { t.Fatal("quiescence function differs from pre-extraction source beyond admitted field rename") }
	oldResume := function(before, "exitsyscallNoP")
	start := strings.Index(oldResume, "\t\tif gomadEnabled {\n\t\t\t// The syscall returned")
	if start < 0 { t.Fatal("baseline resume extraction missing") }
	end := strings.Index(oldResume[start:], "\n\t\t}\n")
	if start < 0 || end < 0 { t.Fatal("baseline resume extraction missing") }
	body := oldResume[start+len("\t\tif gomadEnabled {\n"):start+end]
	body = strings.ReplaceAll("\n"+body, "\n\t\t", "\n")
	if strings.Count(body, "\tlocked = gp.lockedm != 0") != 1 { t.Fatal("baseline locked-state assignment drift") }
	body = strings.Replace(body, "\tlocked = gp.lockedm != 0", "\tlocked := gp.lockedm != 0", 1)
	if "func gomadResumeSyscall(gp *g, pp *p) {"+body+"\n}" != function(overlay, "gomadResumeSyscall") { t.Fatalf("resume helper differs from original block\nBEFORE:\n%s\nAFTER:\n%s", body, function(overlay, "gomadResumeSyscall")) }
	oldCheck := function(before, "checkdead")
	start = strings.Index(oldCheck, "\t\tif gomadSimulationTimeEnabled && gomadSimulationExternalRequests.Load() != 0 {")
	if start < 0 { t.Fatal("baseline clock extraction missing") }
	end = strings.Index(oldCheck[start:], "\n\t\tif wakeTimer {")
	if start < 0 || end < 0 { t.Fatal("baseline clock extraction missing") }
	body = oldCheck[start:start+end]
	body = strings.ReplaceAll("\n"+body, "\n\t", "\n")
	body = strings.ReplaceAll(body, "return\n", "return false, true\n")
	if "func gomadCheckDeadTime() (bool, bool) {"+body+"\n\treturn wakeTimer, false\n}" != function(overlay, "gomadCheckDeadTime") { t.Fatalf("clock helper differs from original block beyond private return adaptation\nBEFORE:\n%s\nAFTER:\n%s", body, function(overlay, "gomadCheckDeadTime")) }
	for _, name := range []string{"gomadResumeSyscall(gp, pp)", "wakeTimer, waiting := gomadCheckDeadTime()", "gomadSimulationTransportSyscalls.Load()", "gomadArrivals.pushBack(gp)", "globrunqput(gp)", "pidleget(0)", "mget()"} { if !strings.Contains(after, name) { t.Fatalf("lost upstream seam: %s", name) } }
	if strings.Contains(after, "func gomadSimulationTimeQuiescenceChanged(") { t.Fatal("legacy implementation remains in proc.go") }
	if read(filepath.Join(beforeRoot, "src/runtime/sizeof_test.go")) != read(filepath.Join(afterRoot, "src/runtime/sizeof_test.go")) { t.Fatal("size assertion drift since pre-extraction source") }
	if function(read(filepath.Join(beforeRoot, "src/runtime/time.go")), "timeSleepUntil") != function(read(filepath.Join(afterRoot, "src/runtime/time.go")), "timeSleepUntil") { t.Fatal("timer-presence result drift") }
	t.Log("three extraction bodies/comments preserved against verified pre-extraction 5df49456e6 source; exact return adaptation, admitted field rename, timer-presence result and sizeof assertion")
}
