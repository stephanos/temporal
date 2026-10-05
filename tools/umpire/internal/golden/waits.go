package golden

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

// Since fn-118.4 a lowered Case's waits are derived from its realization's API behavior
// (.plans/API_BEHAVIOR_HINTS.md, tools/umpire/lower/waits.go), so a realization leaves to that
// behavior the waits it spelled out before: a poll's interval and a command's timeout. The delta lists
// each command whose wait it so leaves (Delta.DerivedWaits). Both harnesses compare such a command
// without its wait on both sides, and each Case lowered from its IR file without the wait the
// command's instruction carries; every other byte of the Model and of the Case, the whole Contract
// included, is compared.

// DerivedWait is one command of a script of a realization of an archived IR file whose wait the
// realization leaves to the API behavior: the baseline command writes a positive Poll.interval_ms or
// Command.timeout_ms, and the current one writes neither.
type DerivedWait struct {
	// Model is the IR file, by archive key: "ir/nexus-caller.json".
	Model string `json:"model"`
	// Script is the script, by id, which is the id of the entrypoint a Case runs it as: "handler".
	Script string `json:"script"`
	// Command is the command, by id, which is the id of the instruction a Case carries it as, or of
	// each, under its ordinal `<id>-<n>`, when the Case carries it more than once: "respond-async".
	Command string `json:"command"`
}

// Waits are the derived waits of one IR file.
type Waits []DerivedWait

// Waits gives the derived waits of the IR file of a key. An empty key names no IR file, so it gives
// none.
func (d Delta) Waits(key string) Waits {
	var out Waits
	for _, w := range d.DerivedWaits {
		if w.Model == key {
			out = append(out, w)
		}
	}
	return out
}

// checkWaits checks each derived wait: an archived IR file, one script's command, listed once.
func (d Delta) checkWaits(newFiles map[string]bool) error {
	seen := map[DerivedWait]bool{}
	for _, w := range d.DerivedWaits {
		switch {
		case !irKey(w.Model) || newFiles[w.Model]:
			return fmt.Errorf("derived wait %+v names no archived IR file", w)
		case w.Script == "" || w.Command == "":
			return fmt.Errorf("derived wait %+v is not one script's command", w)
		case seen[w]:
			return fmt.Errorf("derived wait %+v is listed twice", w)
		}
		seen[w] = true
	}
	return nil
}

// writesWait reports whether a command writes a wait of its own: a poll interval or a timeout.
func writesWait(c *umpirespb.Command) bool {
	return c.GetTimeoutMs() != 0 || c.GetPoll().GetIntervalMs() != 0
}

// commands visits each command of m the wait names: every command of that id, placed or performed,
// of every script of that id of every realization.
func (w DerivedWait) commands(m *umpirespb.Model, visit func(*umpirespb.Command)) {
	for _, r := range m.GetRealizations() {
		for _, s := range r.GetScripts() {
			if s.GetId() != w.Script {
				continue
			}
			for _, item := range s.GetItems() {
				commands := []*umpirespb.Command{item.GetCommand()}
				for _, p := range item.GetPerforms() {
					commands = append(commands, p.GetCommand())
				}
				for _, c := range commands {
					if c != nil && c.GetId() == w.Command {
						visit(c)
					}
				}
			}
		}
	}
}

// writtenIn checks that the baseline m has the wait's command, and that each command of it writes a
// positive poll interval or timeout: a wait the realization could leave to the API behavior.
func (w DerivedWait) writtenIn(m *umpirespb.Model) error {
	n := 0
	var errs []error
	w.commands(m, func(c *umpirespb.Command) {
		n++
		if c.GetTimeoutMs() <= 0 && c.GetPoll().GetIntervalMs() <= 0 {
			errs = append(errs, fmt.Errorf("derived wait %+v names a command the baseline writes no wait of", w))
		}
	})
	if n == 0 {
		errs = append(errs, fmt.Errorf("derived wait %+v names no command of the baseline", w))
	}
	return errors.Join(errs...)
}

// clear clears the poll interval and the timeout of each command of m the waits name.
func (ws Waits) clear(m *umpirespb.Model) {
	for _, w := range ws {
		w.commands(m, func(c *umpirespb.Command) {
			c.TimeoutMs = 0
			if poll := c.GetPoll(); poll != nil {
				poll.IntervalMs = 0
			}
		})
	}
}

// Baseline gives a baseline Model of the IR file as the comparison reads it: without the waits of the
// listed commands, each of which it must have, each writing a positive wait.
func (ws Waits) Baseline(baseline *umpirespb.Model) (*umpirespb.Model, error) {
	if len(ws) == 0 {
		return baseline, nil
	}
	var errs []error
	for _, w := range ws {
		errs = append(errs, w.writtenIn(baseline))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	m := proto.CloneOf(baseline)
	ws.clear(m)
	return m, nil
}

// Current gives a current Model of the IR file as the comparison reads it: without the waits of the
// listed commands, none of which may write one, since the realization leaves it to the API behavior.
// A listed command the current Model lacks stays lacking, so the comparison with the baseline, which
// has it, fails.
func (ws Waits) Current(current *umpirespb.Model) (*umpirespb.Model, error) {
	if len(ws) == 0 {
		return current, nil
	}
	var errs []error
	for _, w := range ws {
		w.commands(current, func(c *umpirespb.Command) {
			if writesWait(c) {
				errs = append(errs, fmt.Errorf("derived wait %+v names a command that still writes its wait", w))
			}
		})
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	m := proto.CloneOf(current)
	ws.clear(m)
	return m, nil
}

// ordinal is the suffix a command carried again in a Case takes after its id: `-<n>`, n from 2.
var ordinal = regexp.MustCompile(`^-[1-9][0-9]*$`)

// Names reports whether an instruction of an entrypoint is a listed command's: the command's id, or
// that id under its ordinal.
func (ws Waits) Names(entrypoint, instruction string) bool {
	for _, w := range ws {
		if w.Script != entrypoint {
			continue
		}
		if rest, ok := strings.CutPrefix(instruction, w.Command); ok && (rest == "" || ordinal.MatchString(rest)) {
			return true
		}
	}
	return false
}

// The members a derived wait leaves out of a listed instruction node, and of the read it carries:
// what the lowering derives from the API behavior.
var (
	derivedNodeMembers = []string{"limits", "waitHints"}
	derivedReadMembers = []string{"pollIntervalMilliseconds", "once"}
)

// Case gives the bytes of a Case lowered from the IR file as both harnesses compare them: each
// instruction node of a listed command, in the entrypoint of its script, without its limits, its
// wait hints, and its read's poll interval and once. Every other byte is kept, the Contract's and the
// provenance's among them. The Case must be compact JSON, as the lowering writes it, so that it is
// rewritten member by member; with no waits listed it is kept as it is.
func (ws Waits) Case(encoded []byte) ([]byte, error) {
	if len(ws) == 0 {
		return encoded, nil
	}
	return trailing(encoded, func(c []byte) ([]byte, error) {
		return editObject(c, func(members []member) ([]member, error) {
			return editMember(members, "program", ws.program)
		})
	})
}

// Program gives the bytes of a Case's Program as Case gives them within the Case.
func (ws Waits) Program(encoded []byte) ([]byte, error) {
	if len(ws) == 0 {
		return encoded, nil
	}
	return trailing(encoded, ws.program)
}

func (ws Waits) program(encoded []byte) ([]byte, error) {
	return editObject(encoded, func(program []member) ([]member, error) {
		return editMember(program, "entrypoints", func(entrypoints []byte) ([]byte, error) {
			return editArray(entrypoints, ws.entrypoint)
		})
	})
}

func (ws Waits) entrypoint(encoded []byte) ([]byte, error) {
	return editObject(encoded, func(entrypoint []member) ([]member, error) {
		var id string
		if err := stringMember(entrypoint, "entrypointId", &id); err != nil {
			return nil, err
		}
		return editMember(entrypoint, "instructions", func(instructions []byte) ([]byte, error) {
			return editArray(instructions, func(node []byte) ([]byte, error) { return ws.node(id, node) })
		})
	})
}

func (ws Waits) node(entrypoint string, encoded []byte) ([]byte, error) {
	return editObject(encoded, func(node []member) ([]member, error) {
		var id string
		if err := stringMember(node, "instructionId", &id); err != nil {
			return nil, err
		}
		if !ws.Names(entrypoint, id) {
			return node, nil
		}
		node = without(node, derivedNodeMembers...)
		return editMember(node, "instruction", func(instruction []byte) ([]byte, error) {
			return editObject(instruction, func(members []member) ([]member, error) {
				return editMember(members, "readEvidence", func(read []byte) ([]byte, error) {
					return editObject(read, func(members []member) ([]member, error) { return without(members, derivedReadMembers...), nil })
				})
			})
		})
	})
}

// member is one member of a JSON object, its value as encoded.
type member struct {
	key   string
	value json.RawMessage
}

// trailing edits encoded without the one newline that may end it, and ends the edit with it again.
func trailing(encoded []byte, edit func([]byte) ([]byte, error)) ([]byte, error) {
	body, newline := bytes.CutSuffix(encoded, []byte("\n"))
	edited, err := edit(body)
	if err != nil || !newline {
		return edited, err
	}
	return append(edited, '\n'), nil
}

// editObject gives a compact JSON object with its members edited. One that its members do not encode
// again byte for byte, such as an indented one, is refused, so an edit changes only what it edits.
func editObject(encoded []byte, edit func([]member) ([]member, error)) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return nil, fmt.Errorf("%.40q is not a JSON object", encoded)
	}
	var members []member
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		members = append(members, member{key: token.(string), value: value})
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	if !bytes.Equal(encodeObject(members), encoded) {
		return nil, fmt.Errorf("%.40q is not compact JSON", encoded)
	}
	edited, err := edit(members)
	if err != nil {
		return nil, err
	}
	return encodeObject(edited), nil
}

func encodeObject(members []member) []byte {
	var out bytes.Buffer
	out.WriteByte('{')
	for i, m := range members {
		if i > 0 {
			out.WriteByte(',')
		}
		// A string always encodes.
		key, _ := json.Marshal(m.key)
		out.Write(key)
		out.WriteByte(':')
		out.Write(m.value)
	}
	out.WriteByte('}')
	return out.Bytes()
}

// editArray gives a compact JSON array with each element edited, refused as editObject refuses one.
func editArray(encoded []byte, edit func([]byte) ([]byte, error)) ([]byte, error) {
	var elements []json.RawMessage
	if err := json.Unmarshal(encoded, &elements); err != nil {
		return nil, err
	}
	if !bytes.Equal(encodeArray(elements), encoded) {
		return nil, fmt.Errorf("%.40q is not a compact JSON array", encoded)
	}
	for i, element := range elements {
		edited, err := edit(element)
		if err != nil {
			return nil, err
		}
		elements[i] = edited
	}
	return encodeArray(elements), nil
}

func encodeArray(elements []json.RawMessage) []byte {
	var out bytes.Buffer
	out.WriteByte('[')
	for i, e := range elements {
		if i > 0 {
			out.WriteByte(',')
		}
		out.Write(e)
	}
	out.WriteByte(']')
	return out.Bytes()
}

// editMember edits the value of the member of a key, if the object has one.
func editMember(members []member, key string, edit func([]byte) ([]byte, error)) ([]member, error) {
	for i, m := range members {
		if m.key != key {
			continue
		}
		edited, err := edit(m.value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		members[i].value = edited
	}
	return members, nil
}

// stringMember reads the string member of a key, which the object must have.
func stringMember(members []member, key string, out *string) error {
	for _, m := range members {
		if m.key == key {
			return json.Unmarshal(m.value, out)
		}
	}
	return fmt.Errorf("no %s", key)
}

// without gives the members without those of the keys.
func without(members []member, keys ...string) []member {
	var out []member
	for _, m := range members {
		if !slices.Contains(keys, m.key) {
			out = append(out, m)
		}
	}
	return out
}
