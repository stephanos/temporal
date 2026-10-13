// Package strictjson decodes owned typed JSON schemas with strict syntax.
package strictjson

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	json "github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	jsonv1 "github.com/go-json-experiment/json/v1"
)

var (
	ErrDuplicateName = jsontext.ErrDuplicateName
	ErrNestingLimit  = errors.New("strict JSON nesting limit")
	ErrTrailingData  = errors.New("strict JSON trailing data")
)

var (
	unmarshal                       = json.Unmarshal
	formatByteArrayAsArray          = jsonv1.FormatByteArrayAsArray
	matchCaseInsensitiveNames       = json.MatchCaseInsensitiveNames
	matchCaseSensitiveDelimiter     = jsonv1.MatchCaseSensitiveDelimiter
	reportErrorsWithLegacySemantics = jsonv1.ReportErrorsWithLegacySemantics
	rejectUnknownMembers            = json.RejectUnknownMembers
	newTextDecoder                  = jsontext.NewDecoder
	readToken                       = (*jsontext.Decoder).ReadToken
	stackDepth                      = (*jsontext.Decoder).StackDepth
	tokenKind                       = jsontext.Token.Kind
)

// Decode unmarshals exactly one JSON value into destination. It rejects
// malformed JSON, duplicate object names, invalid UTF-8, trailing data, and
// unknown members of typed structs. Map keys and values captured as raw JSON
// are not typed schema members; callers own byte bounds, framing, and semantic
// validation for those values.
func Decode(data []byte, destination any) error {
	//gomad:dynamic reviewed 2026-10-13 pinned JSON decoder operates only on caller memory
	return unmarshal(data, destination,
		//gomad:dynamic reviewed 2026-10-13 pinned JSON option preserves encoding/json byte-array input
		formatByteArrayAsArray(true),
		//gomad:dynamic reviewed 2026-10-13 pinned JSON option constructor has no host effects
		matchCaseInsensitiveNames(true),
		//gomad:dynamic reviewed 2026-10-13 pinned JSON option constructor has no host effects
		matchCaseSensitiveDelimiter(true),
		//gomad:dynamic reviewed 2026-10-13 pinned JSON option constructor has no host effects
		reportErrorsWithLegacySemantics(true),
		//gomad:dynamic reviewed 2026-10-13 pinned JSON option constructor has no host effects
		rejectUnknownMembers(true),
	)
}

// CheckSyntax validates one JSON value while enforcing a caller-owned nesting
// limit whose root value is depth zero. A non-positive maximumDepth retains
// the pinned library's own limit.
func CheckSyntax(data []byte, maximumDepth int) error {
	//gomad:dynamic reviewed 2026-10-13 pinned JSON tokenizer operates only on caller memory
	decoder := newTextDecoder(bytes.NewReader(data))
	//gomad:dynamic reviewed 2026-10-13 pinned JSON tokenizer operates only on caller memory
	token, err := readToken(decoder)
	if err != nil {
		return err
	}
	if err := checkTokenDepth(decoder, token, maximumDepth); err != nil {
		return err
	}
	//gomad:dynamic reviewed 2026-10-13 pinned JSON tokenizer reports only its in-memory stack
	for stackDepth(decoder) != 0 {
		//gomad:dynamic reviewed 2026-10-13 pinned JSON tokenizer operates only on caller memory
		token, err := readToken(decoder)
		if err != nil {
			return err
		}
		if err := checkTokenDepth(decoder, token, maximumDepth); err != nil {
			return err
		}
	}
	//gomad:dynamic reviewed 2026-10-13 pinned JSON tokenizer operates only on caller memory
	_, err = readToken(decoder)
	//gomad:dynamic reviewed 2026-10-13 reviewed tokenizer errors expose memory-only identities
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		//gomad:dynamic reviewed 2026-10-13 reviewed tokenizer errors expose memory-only messages
		return fmt.Errorf("%w: %w", ErrTrailingData, err)
	}
	return ErrTrailingData
}

func checkTokenDepth(decoder *jsontext.Decoder, token jsontext.Token, maximumDepth int) error {
	if maximumDepth <= 0 {
		return nil
	}
	//gomad:dynamic reviewed 2026-10-13 pinned JSON tokenizer reports only its in-memory stack
	depth := stackDepth(decoder)
	//gomad:dynamic reviewed 2026-10-13 pinned JSON token reports only its parsed kind
	switch tokenKind(token) {
	case jsontext.KindBeginObject, jsontext.KindBeginArray:
		depth--
	case jsontext.KindEndObject, jsontext.KindEndArray:
		return nil
	case jsontext.KindInvalid, jsontext.KindNull, jsontext.KindFalse, jsontext.KindTrue, jsontext.KindString, jsontext.KindNumber:
	}
	if depth > maximumDepth {
		return ErrNestingLimit
	}
	return nil
}
