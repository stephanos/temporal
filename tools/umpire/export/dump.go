package export

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type dumpSpan struct{ start, end int }

type quintDumpIndex struct {
	wanted              map[string]bool
	parts               map[string]dumpSpan
	base, states        int
	firstMap, outRecord bool
}

func indexQuintDump(itf []byte, wanted []string) (*quintDumpIndex, error) {
	d := &quintDumpIndex{wanted: map[string]bool{}}
	for _, key := range wanted {
		d.wanted[key] = true
	}
	for d.base < len(itf) && dumpWhitespace(itf[d.base]) {
		d.base++
	}
	if err := json.Unmarshal(itf, d); err != nil {
		return nil, fmt.Errorf("the Quint dump is no ITF trace: %w", err)
	}
	if d.states == 0 {
		return nil, errors.New("the Quint dump holds no state")
	}
	if !d.firstMap || !d.outRecord {
		return nil, errors.New("the Quint dump's first state has no variable out")
	}
	return d, nil
}

func (d *quintDumpIndex) part(itf []byte, key string) (any, error) {
	span, ok := d.parts[key]
	if !ok {
		return nil, nil
	}
	var value any
	err := json.Unmarshal(itf[span.start:span.end], &value)
	return value, err
}

// UnmarshalJSON is entered only after encoding/json validates the entire document. Admission keeps
// byte positions, not borrowed callback bytes or decoded owner graphs; every state value is still
// checked for the float64 errors the original []map[string]any decoder reported.
func (d *quintDumpIndex) UnmarshalJSON(data []byte) error {
	w := dumpWalker{data: data, decoder: json.NewDecoder(bytes.NewReader(data)), index: d}
	w.decoder.UseNumber()
	token, span, err := w.token()
	if err != nil || token == nil {
		return err
	}
	if token != json.Delim('{') {
		return w.shapeError("top", token, span)
	}
	for w.decoder.More() {
		key, _, err := w.token()
		if err != nil {
			return err
		}
		if strings.EqualFold(key.(string), "states") {
			w.path = append(w.path, dumpPath{field: key.(string)})
			err = w.states()
			w.path = w.path[:len(w.path)-1]
		} else {
			_, err = w.value(false)
		}
		if err != nil {
			return err
		}
	}
	_, _, err = w.token()
	return err
}

type dumpWalker struct {
	data    []byte
	decoder *json.Decoder
	index   *quintDumpIndex
	path    []dumpPath
}

type dumpPath struct {
	field string
	index int
	array bool
}

func (w *dumpWalker) field() string {
	parts := make([]string, len(w.path))
	for i, p := range w.path {
		if p.array {
			parts[i] = strconv.Itoa(p.index)
		} else {
			parts[i] = strings.ReplaceAll(strings.ReplaceAll(p.field, "~", "~0"), "/", "~1")
		}
	}
	return strings.Join(parts, ".")
}

func dumpWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func (w *dumpWalker) token() (json.Token, dumpSpan, error) {
	start := int(w.decoder.InputOffset())
	token, err := w.decoder.Token()
	for start < len(w.data) && (dumpWhitespace(w.data[start]) || w.data[start] == ':' || w.data[start] == ',') {
		start++
	}
	return token, dumpSpan{start, int(w.decoder.InputOffset())}, err
}

func (w *dumpWalker) value(numbers bool) (dumpSpan, error) {
	token, span, err := w.token()
	if err != nil {
		return span, err
	}
	return w.entered(token, span, numbers)
}

func (w *dumpWalker) entered(token json.Token, span dumpSpan, numbers bool) (dumpSpan, error) {
	switch token {
	case json.Delim('{'), json.Delim('['):
		index := 0
		for w.decoder.More() {
			if token == json.Delim('{') {
				key, _, err := w.token()
				if err != nil {
					return span, err
				}
				w.path = append(w.path, dumpPath{field: key.(string)})
			} else {
				w.path = append(w.path, dumpPath{index: index, array: true})
			}
			_, err := w.value(numbers)
			w.path = w.path[:len(w.path)-1]
			if err != nil {
				return span, err
			}
			index++
		}
		_, end, err := w.token()
		span.end = end.end
		return span, err
	}
	if number, ok := token.(json.Number); ok && numbers {
		if _, err := number.Float64(); err != nil {
			const prefix = `{"states":[{"x":`
			var trace struct {
				States []map[string]any `json:"states"`
			}
			err := json.Unmarshal([]byte(prefix+number.String()+`}]}`), &trace)
			var typed *json.UnmarshalTypeError
			if errors.As(err, &typed) {
				typed.Offset += int64(w.index.base + span.start - len(prefix))
				typed.Field = w.field()
			}
			return span, err
		}
	}
	return span, nil
}

func (w *dumpWalker) resetFirst() {
	w.index.firstMap, w.index.outRecord, w.index.parts = false, false, nil
}

func (w *dumpWalker) states() error {
	token, span, err := w.token()
	if err != nil {
		return err
	}
	if token == nil {
		w.index.states = 0
		w.resetFirst()
		return nil
	}
	if token != json.Delim('[') {
		return w.shapeError("states", token, span)
	}
	count := 0
	for w.decoder.More() {
		w.path = append(w.path, dumpPath{index: count, array: true})
		err := w.state(count == 0)
		w.path = w.path[:len(w.path)-1]
		if err != nil {
			return err
		}
		count++
	}
	w.index.states = count
	if count == 0 {
		w.resetFirst()
	}
	_, _, err = w.token()
	return err
}

func (w *dumpWalker) state(first bool) error {
	token, span, err := w.token()
	if err != nil {
		return err
	}
	if token == nil {
		if first {
			w.resetFirst()
		}
		return nil
	}
	if token != json.Delim('{') {
		return w.shapeError("state", token, span)
	}
	if first {
		// A duplicate nonempty states array reuses its first map; omitted out retains its old value.
		w.index.firstMap = true
	}
	for w.decoder.More() {
		key, _, err := w.token()
		if err != nil {
			return err
		}
		w.path = append(w.path, dumpPath{field: key.(string)})
		if first && key == "out" {
			err = w.out()
		} else {
			_, err = w.value(true)
		}
		w.path = w.path[:len(w.path)-1]
		if err != nil {
			return err
		}
	}
	_, _, err = w.token()
	return err
}

func (w *dumpWalker) out() error {
	token, span, err := w.token()
	if err != nil {
		return err
	}
	w.index.outRecord, w.index.parts = token == json.Delim('{'), nil
	if !w.index.outRecord {
		_, err = w.entered(token, span, true)
		return err
	}
	w.index.parts = map[string]dumpSpan{}
	for w.decoder.More() {
		key, _, err := w.token()
		if err != nil {
			return err
		}
		w.path = append(w.path, dumpPath{field: key.(string)})
		span, err := w.value(true)
		w.path = w.path[:len(w.path)-1]
		if err != nil {
			return err
		}
		if w.index.wanted[key.(string)] {
			span.start, span.end = span.start+w.index.base, span.end+w.index.base
			w.index.parts[key.(string)] = span
		}
	}
	_, _, err = w.token()
	return err
}

func (w *dumpWalker) shapeError(where string, token json.Token, span dumpSpan) error {
	probe := "0"
	switch token.(type) {
	case json.Delim:
		if token == json.Delim('{') {
			probe = "{}"
		} else {
			probe = "[]"
		}
	case string:
		probe = `""`
	case bool:
		probe = "true"
	default:
	}
	switch where {
	case "states":
		probe = `{"states":` + probe + `}`
	case "state":
		probe = `{"states":[` + probe + `]}`
	default:
	}
	var trace struct {
		States []map[string]any `json:"states"`
	}
	err := json.Unmarshal([]byte(probe), &trace)
	var typed *json.UnmarshalTypeError
	if errors.As(err, &typed) {
		typed.Offset = int64(w.index.base + span.end)
		typed.Field = w.field()
	}
	return err
}
