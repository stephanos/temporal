// Package volume is gomad's durable filesystem model. Application filesystem
// operations never access host files.
package volume

// Work charges every cursor visited, including invalid or duplicate selections.
// The model bounds enumeration without consulting ambient wall time.
type enumerationBudget struct {
	maximum uint64
	used    uint64
}

func (budget *enumerationBudget) Take() bool {
	if budget.used == budget.maximum {
		return false
	}
	budget.used++
	return true
}

// CloneDurable detaches the persisted volume image without advancing or revoking
// the original filesystem. Pending writes are intentionally absent from the clone.
func (fs *FS) CloneDurable() *FS {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.forkVolumesLocked()
}
