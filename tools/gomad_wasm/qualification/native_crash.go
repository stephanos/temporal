package qualification

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"
	"sort"

	"go.temporal.io/server/tools/gomad3/volume"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3sim"
	"go.temporal.io/server/tools/gomad_wasm/simulation"
)

const nativeCrashCatalogueBytes = 8 << 20

// NativeCrashObservation identifies one actual execute/replay measurement.
type NativeCrashObservation struct {
	Phase     string
	Iteration uint64
}

// NativeCrashCatalogue retains actual native full states once per distinct
// incarnation/volume/digest. Different observations are preserved, not folded
// into a matched catalogue. It is projection in the existing Result, not an
// enumeration algorithm or another execution/report owner.
type NativeCrashCatalogue struct {
	Handle       gomad3sim.NodeHandle
	Volume       string
	SHA256       record.SHA256
	Enumeration  volume.CrashEnumeration
	Observations []NativeCrashObservation
}

// This conservative upper bound covers JSON escaping/base64 and scalar fields
// before cloning or serializing the potentially large native entry payloads.
func nativeEnumerationBound(enumeration volume.CrashEnumeration) (uint64, error) {
	if len(enumeration.States) > 32 {
		return 0, errors.New("native crash catalogue state capacity")
	}
	size, entries := uint64(512), 0
	if cursor := enumeration.ChoiceExploration; cursor != nil {
		if len(cursor.Cursor) > 65536 || len(cursor.Seen) > 1024 {
			return 0, errors.New("native crash cursor capacity")
		}
		size += uint64(512 + 6*(len(cursor.Volume)+len(cursor.PendingSHA256)+len(cursor.Identity)) + 4*((len(cursor.Cursor)+2)/3))
		for _, seen := range cursor.Seen {
			if len(seen) > 256 {
				return 0, errors.New("native crash cursor identity capacity")
			}
			size += uint64(6*len(seen) + 4)
		}
	}
	for _, state := range enumeration.States {
		if len(state.Volume) > 128 || len(state.Identity) > 256 || len(state.PendingSHA256) > 256 || len(state.SelectedOperations) > 4096 {
			return 0, errors.New("native crash state identity/selection capacity")
		}
		size += uint64(512 + 6*(len(state.Volume)+len(state.Identity)+len(state.PendingSHA256)) + 24*len(state.SelectedOperations))
		entries += len(state.Entries)
		if entries > 32768 {
			return 0, errors.New("native crash entry capacity")
		}
		for _, entry := range state.Entries {
			if len(entry.Path) > 4096 || len(entry.Data) > nativeCrashCatalogueBytes {
				return 0, errors.New("native crash entry payload capacity")
			}
			size += uint64(512 + 6*len(entry.Path) + 4*((len(entry.Data)+2)/3))
			if size > nativeCrashCatalogueBytes {
				return 0, errors.New("native crash catalogue byte capacity")
			}
		}
	}
	if size > nativeCrashCatalogueBytes {
		return 0, errors.New("native crash catalogue byte capacity")
	}
	return size, nil
}

func cloneNativeEnumeration(enumeration volume.CrashEnumeration) volume.CrashEnumeration {
	enumeration.States = slices.Clone(enumeration.States)
	for i := range enumeration.States {
		state := &enumeration.States[i]
		state.SelectedOperations = slices.Clone(state.SelectedOperations)
		state.Entries = slices.Clone(state.Entries)
		for j := range state.Entries {
			state.Entries[j].Data = bytes.Clone(state.Entries[j].Data)
		}
	}
	if enumeration.ChoiceExploration != nil {
		cursor := *enumeration.ChoiceExploration
		cursor.Cursor, cursor.Seen = bytes.Clone(cursor.Cursor), slices.Clone(cursor.Seen)
		enumeration.ChoiceExploration = &cursor
	}
	return enumeration
}

func captureNativeCrashCatalogues(result *Result, incarnations []simulation.IncarnationResult, phase string, iteration uint64) error {
	if result.Manifest.Workload != "sqlite" {
		return nil
	}
	for _, incarnation := range incarnations {
		volumes := make([]string, 0, len(incarnation.Result.StorageCrashEnumeration))
		if len(incarnation.Result.StorageCrashEnumeration) > 4 {
			return errors.New("native crash volume capacity")
		}
		for name := range incarnation.Result.StorageCrashEnumeration {
			volumes = append(volumes, name)
		}
		sort.Strings(volumes)
		for _, name := range volumes {
			enumeration := incarnation.Result.StorageCrashEnumeration[name]
			if _, err := nativeEnumerationBound(enumeration); err != nil {
				return err
			}
			encoded, err := json.Marshal(enumeration)
			if err != nil {
				return err
			}
			digest := record.HashBytes(encoded)
			found := -1
			for index, catalogue := range result.NativeCrashCatalogues {
				if catalogue.Handle == incarnation.Handle && catalogue.Volume == name && catalogue.SHA256 == digest {
					found = index
					break
				}
			}
			if found < 0 {
				if len(result.NativeCrashCatalogues) >= 32 {
					return errors.New("native crash catalogue count capacity")
				}
				var total uint64
				for _, catalogue := range result.NativeCrashCatalogues {
					bound, err := nativeEnumerationBound(catalogue.Enumeration)
					if err != nil {
						return err
					}
					total += bound
				}
				bound, err := nativeEnumerationBound(enumeration)
				if err != nil {
					return err
				}
				if total+bound > nativeCrashCatalogueBytes {
					return errors.New("aggregate native crash catalogue byte capacity")
				}
				result.NativeCrashCatalogues = append(result.NativeCrashCatalogues, NativeCrashCatalogue{Handle: incarnation.Handle, Volume: name, SHA256: digest, Enumeration: cloneNativeEnumeration(enumeration)})
				found = len(result.NativeCrashCatalogues) - 1
			}
			if len(result.NativeCrashCatalogues[found].Observations) >= 128 {
				return errors.New("native crash observation capacity")
			}
			result.NativeCrashCatalogues[found].Observations = append(result.NativeCrashCatalogues[found].Observations, NativeCrashObservation{Phase: phase, Iteration: iteration})
		}
	}
	return nil
}

func validateNativeCrashCatalogues(result Result) error {
	if result.NativeCrashSelection != nil {
		state := *result.NativeCrashSelection
		if result.Manifest.Workload != "sqlite" || state.Identity == "" || state.Volume == "" || state.PendingSHA256 == "" {
			return errors.New("invalid native SQLite selection identity")
		}
		if _, err := nativeEnumerationBound(volume.CrashEnumeration{States: []volume.CrashState{state}}); err != nil {
			return err
		}
		if _, err := record.ParseSHA256(string(result.NativeCrashSourceResultSHA256)); err != nil {
			return errors.New("native selected-state source Result binding required")
		}
	} else if result.NativeCrashSourceResultSHA256 != "" {
		return errors.New("native crash source reference without a selected state")
	}
	if len(result.NativeCrashCatalogues) == 0 {
		return nil
	}
	if result.Manifest.Workload != "sqlite" || len(result.NativeCrashCatalogues) > 32 {
		return errors.New("native SQLite crash catalogue admission")
	}
	var total uint64
	seen := map[struct {
		handle        gomad3sim.NodeHandle
		volume, phase string
		iteration     uint64
	}]bool{}
	for _, catalogue := range result.NativeCrashCatalogues {
		bound, err := nativeEnumerationBound(catalogue.Enumeration)
		if err != nil {
			return err
		}
		total += bound
		if total > nativeCrashCatalogueBytes || catalogue.Handle.Node == "" || catalogue.Handle.Incarnation == 0 || catalogue.Volume == "" || len(catalogue.Volume) > 128 || len(catalogue.Observations) == 0 || len(catalogue.Observations) > 128 {
			return errors.New("native crash catalogue bound/identity")
		}
		encoded, err := json.Marshal(catalogue.Enumeration)
		if err != nil {
			return err
		}
		if record.HashBytes(encoded) != catalogue.SHA256 {
			return errors.New("native crash catalogue digest changed")
		}
		for _, state := range catalogue.Enumeration.States {
			if state.Volume != catalogue.Volume || state.Identity == "" || state.PendingSHA256 == "" {
				return errors.New("native crash state volume/identity differs from catalogue")
			}
		}
		for _, observation := range catalogue.Observations {
			key := struct {
				handle        gomad3sim.NodeHandle
				volume, phase string
				iteration     uint64
			}{catalogue.Handle, catalogue.Volume, observation.Phase, observation.Iteration}
			if observation.Phase != "execute" && observation.Phase != "replay" || observation.Iteration == 0 || observation.Iteration > result.Manifest.Repeat || seen[key] {
				return errors.New("native crash observation phase/iteration repeated or invalid")
			}
			seen[key] = true
		}
	}
	return nil
}

// MatchedNativeCrashEnumeration requires actual execute/replay observations for
// every repetition and exact full native-state identity, including entry bytes.
func MatchedNativeCrashEnumeration(result Result, handle gomad3sim.NodeHandle, name string) (volume.CrashEnumeration, error) {
	if err := validateNativeCrashCatalogues(result); err != nil {
		return volume.CrashEnumeration{}, err
	}
	var matched *NativeCrashCatalogue
	for index := range result.NativeCrashCatalogues {
		catalogue := &result.NativeCrashCatalogues[index]
		if catalogue.Handle != handle || catalogue.Volume != name {
			continue
		}
		if matched != nil {
			return volume.CrashEnumeration{}, errors.New("native crash enumeration diverged across observations")
		}
		matched = catalogue
	}
	if matched == nil || !matched.Enumeration.Complete || matched.Enumeration.Capacity != "" || matched.Enumeration.ChoiceExploration != nil || len(matched.Observations) != int(2*result.Manifest.Repeat) || result.Manifest.Repeat == 0 || result.Manifest.Repeat > 32 {
		return volume.CrashEnumeration{}, errors.New("complete matched native crash observations required")
	}
	return cloneNativeEnumeration(matched.Enumeration), nil
}
