//go:build umpire3_tla_experiment

package tla

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"go.temporal.io/server/tests/umpire3/protocol"
)

type Generated struct {
	Module string
	TLA    []byte
	Config []byte
}

func (g Generated) Digest() string {
	digest := sha256.New()
	_, _ = fmt.Fprintf(digest, "%d:%s:%d:", len(g.Module), g.Module, len(g.TLA))
	_, _ = digest.Write(g.TLA)
	_, _ = fmt.Fprintf(digest, ":%d:", len(g.Config))
	_, _ = digest.Write(g.Config)
	return fmt.Sprintf("sha256:%x", digest.Sum(nil))
}

var tlaIdentifierPart = regexp.MustCompile(`[^A-Za-z0-9_]`)

func Generate(view protocol.TemporalView) (Generated, error) {
	if err := view.Validate(); err != nil {
		return Generated{}, fmt.Errorf("validate temporal view: %w", err)
	}
	module := "Umpire3" + identifier(string(view.Target)) + identifier(view.Variant)
	if module == "" {
		return Generated{}, errors.New("temporal view does not produce a TLA+ module name")
	}
	actionNames := make(map[protocol.ActionKind]string, len(view.Actions))
	for _, action := range view.Actions {
		actionNames[action] = identifier(string(action))
	}

	var source strings.Builder
	fmt.Fprintf(&source, "---- MODULE %s ----\n", module)
	source.WriteString("EXTENDS TLC\n\n")
	fmt.Fprintf(&source, "\\* target: %s\n", view.Target)
	fmt.Fprintf(&source, "\\* property: %s\n", view.Property)
	fmt.Fprintf(&source, "\\* semantic-hash: %s\n", view.SemanticHash)
	fmt.Fprintf(&source, "\\* canonical-model: %s\n\n", view.CanonicalModel)
	source.WriteString("VARIABLES\n    \\* @type: Str;\n    phase\n\n")
	source.WriteString("\\* @type: <<Str>>;\nvars == <<phase>>\n\n")
	fmt.Fprintf(&source, "States == %s\n\n", tlaSet(view.States))
	fmt.Fprintf(&source, "Init == phase = %q\n\n", view.Initial)
	for _, action := range view.Actions {
		name := actionNames[action]
		var transitions []protocol.TemporalTransition
		for _, transition := range view.Transitions {
			if transition.Action == action {
				transitions = append(transitions, transition)
			}
		}
		fmt.Fprintf(&source, "%s ==\n", name)
		if len(transitions) == 0 {
			source.WriteString("    FALSE\n\n")
			continue
		}
		for index, transition := range transitions {
			prefix := "    "
			if len(transitions) > 1 {
				if index == 0 {
					prefix += "\\/ "
				} else {
					prefix += "\\/ "
				}
			}
			fmt.Fprintf(&source, "%s/\\ phase = %q\n", prefix, transition.FromState)
			fmt.Fprintf(&source, "       /\\ phase' = %q\n", transition.ToState)
		}
		source.WriteString("\n")
	}
	source.WriteString("Next ==\n")
	for index, transition := range view.Transitions {
		prefix := "    \\/ "
		if index == 0 {
			prefix = "    \\/ "
		}
		fmt.Fprintf(&source, "%s%s\n", prefix, actionNames[transition.Action])
	}
	source.WriteString("\nTypeOK == phase \\in States\n\n")
	fairnessNames := make([]string, len(view.Fairness))
	for index, fairness := range view.Fairness {
		name := "Responsive" + actionNames[fairness.Action]
		fairnessNames[index] = name
		fmt.Fprintf(&source, "%s == [](phase \\in %s => <> (phase \\notin %s))\n\n",
			name, tlaSet(fairness.EnabledStates), tlaSet(fairness.EnabledStates))
	}
	source.WriteString("Spec == Init /\\ [][Next]_vars")
	for _, fairness := range fairnessNames {
		fmt.Fprintf(&source, " /\\ %s", fairness)
	}
	source.WriteString("\n\n")
	fmt.Fprintf(&source, "Progress == [](phase \\in %s => <> (phase \\in %s))\n\n",
		tlaSet(view.Progress.TriggerStates), tlaSet(view.Progress.GoalStates))
	source.WriteString("====\n")

	config := "SPECIFICATION Spec\nINVARIANT TypeOK\nPROPERTY Progress\nCHECK_DEADLOCK FALSE\n"
	return Generated{Module: module, TLA: []byte(source.String()), Config: []byte(config)}, nil
}

func tlaSet(values []string) string {
	ordered := append([]string(nil), values...)
	slices.Sort(ordered)
	quoted := make([]string, len(ordered))
	for index, value := range ordered {
		quoted[index] = fmt.Sprintf("%q", value)
	}
	return "{" + strings.Join(quoted, ", ") + "}"
}

func identifier(value string) string {
	value = tlaIdentifierPart.ReplaceAllString(value, " ")
	parts := strings.Fields(value)
	for index, part := range parts {
		runes := []rune(part)
		if len(runes) != 0 {
			runes[0] = unicode.ToUpper(runes[0])
		}
		parts[index] = string(runes)
	}
	return strings.Join(parts, "")
}
