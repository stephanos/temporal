package export

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// rawRecord indexes a record already admitted by indexQuintDump, without retaining its generic
// values. Duplicate fields select their last span, as the original map decoder does.
func rawRecord(input []byte) (map[string]dumpSpan, bool, error) {
	if len(input) == 0 {
		return nil, false, nil
	}
	w := dumpWalker{data: input, decoder: json.NewDecoder(bytes.NewReader(input))}
	w.decoder.UseNumber()
	token, _, err := w.token()
	if err != nil || token != json.Delim('{') {
		return nil, false, err
	}
	fields := map[string]dumpSpan{}
	for w.decoder.More() {
		key, _, err := w.token()
		if err != nil {
			return nil, true, err
		}
		span, err := w.value(false)
		if err != nil {
			return nil, true, err
		}
		fields[key.(string)] = span
	}
	_, _, err = w.token()
	return fields, true, err
}

func rawField(input []byte, fields map[string]dumpSpan, name string) []byte {
	span, ok := fields[name]
	if !ok {
		return nil
	}
	return input[span.start:span.end]
}

func rawValue(input []byte) (any, error) {
	if len(input) == 0 {
		return nil, nil
	}
	var value any
	err := json.Unmarshal(input, &value)
	return value, err
}

// rawList leaves a standard decoder at the first element. Only one #set wrapper is unwrapped,
// exactly as list does; malformed containers use list itself for the original refusal.
func rawList(input []byte) (*json.Decoder, error) {
	original := input
	fields, record, err := rawRecord(input)
	if err != nil {
		return nil, err
	}
	if record {
		input = rawField(input, fields, "#set")
	}
	if len(input) > 0 {
		d := json.NewDecoder(bytes.NewReader(input))
		token, err := d.Token()
		if err != nil {
			return nil, err
		}
		if token == json.Delim('[') {
			return d, nil
		}
	}
	value, err := rawValue(original)
	if err != nil {
		return nil, err
	}
	_, err = list(value)
	return nil, err
}

func rawEach[T any](input []byte, read func(any) (T, error)) ([]T, error) {
	d, err := rawList(input)
	if err != nil {
		return nil, err
	}
	out := make([]T, 0)
	for d.More() {
		var raw any
		if err := d.Decode(&raw); err != nil {
			return nil, err
		}
		value, err := read(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	_, err = d.Token()
	return out, err
}

// rawBySource delegates each complete generic row to the unchanged native traversal. Replacing
// the one returned source entry preserves duplicate-source resets and all prior callback failures.
func rawBySource[T any](input []byte, source, class func(any) (string, error), step func(any) (T, error)) (map[string]map[string][]T, error) {
	d, err := rawList(input)
	if err != nil {
		return nil, err
	}
	out := map[string]map[string][]T{}
	for d.More() {
		var raw any
		if err := d.Decode(&raw); err != nil {
			return nil, err
		}
		row, err := bySource([]any{raw}, source, class, step)
		if err != nil {
			return nil, err
		}
		for from, steps := range row {
			out[from] = steps
		}
	}
	_, err = d.Token()
	return out, err
}

func (r reader) rawMachine(input []byte) (*machineView, bool, error) {
	fields, present, err := rawRecord(input)
	if err != nil || !present {
		return nil, present, err
	}
	v := &machineView{}
	if v.Starts, err = rawEach(rawField(input, fields, "starts"), r.state); err != nil {
		return nil, true, fmt.Errorf("starts: %w", err)
	}
	if v.Reach, err = rawEach(rawField(input, fields, "reach"), r.state); err != nil {
		return nil, true, fmt.Errorf("reach: %w", err)
	}
	if v.Ends, err = rawEach(rawField(input, fields, "ends"), r.state); err != nil {
		return nil, true, fmt.Errorf("ends: %w", err)
	}
	if v.Classes, err = rawEach(rawField(input, fields, "classes"), r.class); err != nil {
		return nil, true, fmt.Errorf("classes: %w", err)
	}
	slices.Sort(v.Reach)
	slices.Sort(v.Ends)
	slices.Sort(v.Classes)
	closed, err := rawValue(rawField(input, fields, "closed"))
	if err != nil {
		return nil, true, err
	}
	if v.Closed, err = boolean(closed, "closed"); err != nil {
		return nil, true, err
	}
	if v.Rows, err = rawBySource(rawField(input, fields, "rows"), r.state, r.class, r.result); err != nil {
		return nil, true, fmt.Errorf("rows: %w", err)
	}
	if _, ok := fields["claims"]; ok {
		if v.Claims, err = rawBySource(rawField(input, fields, "claims"), r.state, r.class, r.claims); err != nil {
			return nil, true, fmt.Errorf("claims: %w", err)
		}
	}
	if _, ok := fields["product"]; ok {
		if v.Product, err = r.rawProduct(rawField(input, fields, "product")); err != nil {
			return nil, true, fmt.Errorf("product: %w", err)
		}
	}
	return v, true, nil
}

func (r reader) rawProduct(input []byte) (*productView, error) {
	fields, present, err := rawRecord(input)
	if err != nil {
		return nil, err
	}
	if !present {
		return nil, errors.New("the product is no record")
	}
	p := &productView{States: map[string]productState{}}
	if p.Starts, err = rawEach(rawField(input, fields, "starts"), r.productState); err != nil {
		return nil, err
	}
	closed, err := rawValue(rawField(input, fields, "closed"))
	if err != nil {
		return nil, err
	}
	if p.Closed, err = boolean(closed, "closed"); err != nil {
		return nil, err
	}
	source := func(raw any) (string, error) {
		ps, err := r.productState(raw)
		p.States[ps.key()] = ps
		return ps.key(), err
	}
	p.Steps, err = rawBySource(rawField(input, fields, "edges"), source, r.class, r.productStep)
	return p, err
}
