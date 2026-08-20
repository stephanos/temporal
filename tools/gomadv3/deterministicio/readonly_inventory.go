package deterministicio

import (
	"errors"
	"path"
	"sort"
)

func CaptureReadOnlyMountInventory(mappings []Mapping, limits Limits) (_ CapturedInputsManifest, retErr error) {
	captured, err := CaptureReadOnlyMountInputs(mappings, limits)
	if err != nil {
		return CapturedInputsManifest{}, err
	}
	return captured.Manifest, nil
}

func CaptureReadOnlyMountInputs(mappings []Mapping, limits Limits) (_ CapturedInputs, retErr error) {
	broker, err := Prepare(mappings, limits)
	if err != nil {
		return CapturedInputs{}, err
	}
	defer func() { retErr = errors.Join(retErr, broker.Close()) }()
	pending := make([]string, len(mappings))
	for index, mapping := range mappings {
		pending[index] = mapping.Target
	}
	sort.Strings(pending)
	for len(pending) != 0 {
		name := pending[0]
		pending = pending[1:]
		entry, err := broker.Lookup(name)
		if err != nil {
			return CapturedInputs{}, err
		}
		if entry.Kind != KindDirectory {
			continue
		}
		children := make([]string, len(entry.Children))
		for index, child := range entry.Children {
			children[index] = path.Join(name, child.Name)
		}
		pending = append(pending, children...)
	}
	captured, err := EncodeCapturedInputs(mappings, limits, broker.Captured())
	if err != nil {
		return CapturedInputs{}, err
	}
	return captured, nil
}
