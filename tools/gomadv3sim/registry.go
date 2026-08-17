package gomadv3sim

import (
	"fmt"
	"slices"
	"sync"
)

var bootRegistry = struct {
	sync.RWMutex
	boots map[BootID]BootFunc
}{boots: make(map[BootID]BootFunc)}

func RegisterBoot(id BootID, boot BootFunc) error {
	if err := validateID("boot ID", string(id)); err != nil {
		return err
	}
	if boot == nil {
		return fmt.Errorf("boot %q function is nil", id)
	}
	bootRegistry.Lock()
	defer bootRegistry.Unlock()
	if _, exists := bootRegistry.boots[id]; exists {
		return fmt.Errorf("boot %q is already registered", id)
	}
	bootRegistry.boots[id] = boot
	return nil
}

func RegisteredBoot(id BootID) (BootFunc, bool) {
	bootRegistry.RLock()
	defer bootRegistry.RUnlock()
	boot, ok := bootRegistry.boots[id]
	return boot, ok
}

func RegisteredBootIDs() []BootID {
	bootRegistry.RLock()
	defer bootRegistry.RUnlock()
	ids := make([]BootID, 0, len(bootRegistry.boots))
	for id := range bootRegistry.boots {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}
