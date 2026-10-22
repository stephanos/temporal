package softassert

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad/internal/model/strictjson"
)

const Prefix = "failed assertion: "

type Observation struct {
	Messages []string
	Complete bool
}

func Observe(stdout, stderr []byte, complete bool) Observation {
	result := Observation{Complete: complete}
	for _, stream := range [][]byte{stdout, stderr} {
		if !utf8.Valid(stream) {
			result.Complete = false
			continue
		}
		for _, line := range bytes.Split(stream, []byte{'\n'}) {
			message, assertion, readable := observeLine(strings.TrimLeft(strings.TrimSuffix(string(line), "\r"), " \t"))
			if !readable {
				result.Complete = false
			}
			if assertion && readable {
				result.Messages = append(result.Messages, message)
			}
		}
	}
	slices.Sort(result.Messages)
	result.Messages = slices.Compact(result.Messages)
	return result
}

func observeLine(line string) (string, bool, bool) {
	at := jsonRecordStart(line)
	jsonRecord := at >= 0
	if !jsonRecord {
		if fields := strings.LastIndex(line, "\t{"); fields >= 0 {
			at = fields + 1
		}
	}
	if at >= 0 {
		var fields map[string]json.RawMessage
		if err := strictjson.Decode([]byte(line[at:]), &fields); err == nil {
			var tagged bool
			for _, key := range []string{"failed-assertion", "FailedAssertion"} {
				var value bool
				if data := fields[key]; data != nil {
					if err := strictjson.Decode(data, &value); err != nil {
						return "", false, false
					}
					tagged = tagged || value
				}
			}
			if jsonRecord {
				var message string
				for _, key := range []string{"msg", "message"} {
					if data := fields[key]; data != nil {
						if err := strictjson.Decode(data, &message); err != nil {
							return "", false, !tagged
						}
						break
					}
				}
				assertion := tagged || strings.HasPrefix(message, Prefix)
				return staticMessage(message, assertion)
			}
			line = strings.TrimSuffix(line[:at], "\t")
			if tagged && !strings.Contains(line, Prefix) {
				parts := strings.Split(line, "\t")
				return staticMessage(parts[len(parts)-1], true)
			}
		} else if strings.Contains(line, Prefix) || strings.Contains(line, "failed-assertion") || strings.Contains(line, "FailedAssertion") {
			return "", false, false
		}
	}
	if at := strings.Index(line, Prefix); at >= 0 {
		return staticMessage(line[at:], true)
	}
	return "", false, true
}

func jsonRecordStart(line string) int {
	at := strings.IndexByte(line, '{')
	if at <= 0 {
		return at
	}
	// testing.T logs prefix records with a source file and decimal line number.
	prefix := strings.TrimSpace(line[:at])
	if !strings.HasSuffix(prefix, ":") {
		return -1
	}
	prefix = strings.TrimSuffix(prefix, ":")
	separator := strings.LastIndexByte(prefix, ':')
	if separator <= 0 || separator == len(prefix)-1 {
		return -1
	}
	for _, digit := range prefix[separator+1:] {
		if digit < '0' || digit > '9' {
			return -1
		}
	}
	if strings.ContainsAny(prefix[:separator], " \t") || strings.Contains(prefix[:separator], Prefix) {
		return -1
	}
	return at
}

func staticMessage(message string, assertion bool) (string, bool, bool) {
	if !assertion {
		return "", false, true
	}
	message = strings.TrimPrefix(message, Prefix)
	if len(message) == 0 || len(message) > 4096 {
		return "", true, false
	}
	return message, true, true
}
