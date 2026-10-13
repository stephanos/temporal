// Package strictjson decodes owned typed JSON schemas with strict syntax.
package strictjson

import (
	json "github.com/go-json-experiment/json"
	jsonv1 "github.com/go-json-experiment/json/v1"
)

var (
	unmarshal                   = json.Unmarshal
	matchCaseInsensitiveNames   = json.MatchCaseInsensitiveNames
	matchCaseSensitiveDelimiter = jsonv1.MatchCaseSensitiveDelimiter
	rejectUnknownMembers        = json.RejectUnknownMembers
)

// Decode unmarshals exactly one JSON value into destination. It rejects
// malformed JSON, duplicate object names, invalid UTF-8, trailing data, and
// unknown members of typed structs. Map keys and values captured as raw JSON
// are not typed schema members; callers own byte bounds, framing, and semantic
// validation for those values.
func Decode(data []byte, destination any) error {
	//gomad:dynamic reviewed 2026-10-13 pinned JSON decoder operates only on caller memory
	return unmarshal(data, destination,
		//gomad:dynamic reviewed 2026-10-13 pinned JSON option constructor has no host effects
		matchCaseInsensitiveNames(true),
		//gomad:dynamic reviewed 2026-10-13 pinned JSON option constructor has no host effects
		matchCaseSensitiveDelimiter(true),
		//gomad:dynamic reviewed 2026-10-13 pinned JSON option constructor has no host effects
		rejectUnknownMembers(true),
	)
}
