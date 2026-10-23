package runner

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
)

type commandResult struct {
	code           int
	stdout, stderr string
}

func command(t *testing.T, drivers map[string]NewDriver, args ...string) commandResult {
	t.Helper()
	var stdout, stderr strings.Builder
	code := Main(context.Background(), args, &stdout, &stderr, drivers)
	return commandResult{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func scripted(script map[byte][]step) map[string]NewDriver {
	return map[string]NewDriver{"fake": func(context.Context) (Driver, error) { return &fakeDriver{script: script}, nil }}
}

func TestUmpireCommandExitCodes(t *testing.T) {
	wasm := statefulGuest(t)
	directory := t.TempDir()
	model := filepath.Join(directory, "guest.wasm")
	world := filepath.Join(directory, "world.binpb")
	observation := filepath.Join(directory, "observation.binpb")
	for path, contents := range map[string][]byte{model: wasm, world: mustMarshal(t, control(0)), observation: mustMarshal(t, control(2))} {
		if err := os.WriteFile(path, contents, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	store := filepath.Join(directory, "store")
	if err := os.Mkdir(store, 0o755); err != nil {
		t.Fatal(err)
	}
	base := []string{"--model", model, "--world", world, "--store", store}
	published := regexp.MustCompile(`(?m)^published ([0-9a-f]{64}/[0-9]+/[0-9a-f]{64}\.binpb)$`)

	explained := command(t, scripted(good), append([]string{"explore", "--seed", "2"}, base...)...)
	if explained.code != ExitExplained || len(published.FindAllString(explained.stdout, -1)) != 2 || !strings.HasSuffix(explained.stdout, "explained: quiescent\n") {
		t.Fatalf("explore: %+v", explained)
	}
	if !strings.Contains(explained.stderr, "umpire: mode=explore iteration=1 seed=2 events=10 verdict=explained termination=quiescent") || strings.Contains(explained.stderr, directory) {
		t.Fatalf("progress: %s", explained.stderr)
	}
	recorded := filepath.Join(store, published.FindStringSubmatch(explained.stdout)[1])

	unexplained := command(t, scripted(buggy), append([]string{"explore", "--seed", "1"}, base...)...)
	if unexplained.code != ExitUnexplained || !strings.HasSuffix(unexplained.stdout, "unexplained at event 5\n") {
		t.Fatalf("unexplained: %+v", unexplained)
	}
	buggyRun := filepath.Join(store, published.FindStringSubmatch(unexplained.stdout)[1])

	faulted := command(t, scripted(map[byte][]step{0: {{kind: EventFault, fault: "unreachable"}}}), append([]string{"explore", "--seed", "1"}, base...)...)
	if faulted.code != ExitRefused || !strings.Contains(faulted.stdout, "refused at event 2: driver fault: unreachable") {
		t.Fatalf("fault: %+v", faulted)
	}

	repeated := command(t, scripted(good), append([]string{"repeat", "--seed", "1", "--count", "2"}, base...)...)
	if repeated.code != ExitExplained || !strings.Contains(repeated.stdout, "already-published ") {
		t.Fatalf("repeat: %+v", repeated)
	}

	reduced := command(t, scripted(buggy), "reduce", "--model", model, "--run", buggyRun, "--store", store)
	if reduced.code != ExitUnexplained || !strings.Contains(reduced.stdout, "already-published ") {
		t.Fatalf("reduce: %+v", reduced)
	}

	if replayed := command(t, nil, "replay", "--model", model, "--run", recorded); replayed.code != ExitExplained {
		t.Fatalf("replay: %+v", replayed)
	}
	if replayed := command(t, nil, "replay", "--model", model, "--run", buggyRun); replayed.code != ExitUnexplained {
		t.Fatalf("replay Unexplained: %+v", replayed)
	}
	if asked := command(t, nil, "question", "--model", model, "--world", world, "--seed", "3"); asked.code != ExitExplained || asked.stdout != "explained: question\n" {
		t.Fatalf("question: %+v", asked)
	}
	if asked := command(t, nil, "question", "--model", model, "--run", recorded, "--events", "4", "--observation", observation); asked.code != ExitUnexplained {
		t.Fatalf("question Judge: %+v", asked)
	}

	data, err := os.ReadFile(recorded)
	if err != nil {
		t.Fatal(err)
	}
	run := new(Run)
	if err := proto.Unmarshal(data, run); err != nil {
		t.Fatal(err)
	}
	run.GetEvents()[5].GetAnswer().GetResponse().Reply = control(41)
	tampered := filepath.Join(directory, "tampered.binpb")
	if err := os.WriteFile(tampered, mustMarshal(t, run), 0o644); err != nil {
		t.Fatal(err)
	}
	if replayed := command(t, nil, "replay", "--model", model, "--run", tampered); replayed.code != ExitRefused || !strings.Contains(replayed.stdout, "replay: event 5 differs") {
		t.Fatalf("tampered replay: %+v", replayed)
	}
	if replayed := command(t, nil, "replay", "--model", model, "--run", filepath.Join(directory, strings.Repeat("0", 64)+RunExtension)); replayed.code != ExitError {
		t.Fatalf("missing Run: %+v", replayed)
	}

	two := map[string]NewDriver{"a": scripted(good)["fake"], "b": scripted(good)["fake"]}
	for name, tc := range map[string]struct {
		drivers map[string]NewDriver
		args    []string
	}{
		"no mode":         {nil, nil},
		"unknown mode":    {nil, []string{"run", "--model", model}},
		"no model":        {scripted(good), []string{"explore", "--store", store}},
		"no store":        {scripted(good), []string{"explore", "--model", model, "--seed", "1"}},
		"no driver":       {nil, append([]string{"explore", "--seed", "1"}, base...)},
		"ambiguous":       {two, append([]string{"explore", "--seed", "1"}, base...)},
		"unknown driver":  {two, append([]string{"explore", "--seed", "1", "--driver", "c"}, base...)},
		"no seed":         {scripted(good), base},
		"positional":      {nil, []string{"replay", "--model", model, "extra"}},
		"no recorded Run": {nil, []string{"replay", "--model", model}},
		"bad world":       {scripted(good), append([]string{"explore", "--seed", "1", "--world", model}, base[:2]...)},
	} {
		if name == "no seed" {
			tc.args = append([]string{"explore"}, base...)
		}
		if result := command(t, tc.drivers, tc.args...); result.code != ExitError {
			t.Errorf("%s: %+v", name, result)
		}
	}
	if chosen := command(t, two, append([]string{"explore", "--seed", "1", "--driver", "b"}, base...)...); chosen.code != ExitExplained {
		t.Fatalf("chosen driver: %+v", chosen)
	}
}

func mustMarshal(t *testing.T, message proto.Message) []byte {
	t.Helper()
	data, err := proto.Marshal(message)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
