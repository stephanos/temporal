package wasi

import (
	"bytes"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

// sameStorageEvidence compares the storage evidence of the Go-owner reference
// and the helper-owned volume, including the recovery image identity: both
// owners encode the same self-contained image v1 bytes.
func sameStorageEvidence(reference, local Result) string {
	if diff := sameEvidence(reference, local); diff != "" {
		return diff
	}
	switch {
	case reference.Message != local.Message:
		return "message"
	case reference.EnvironmentIdentity != local.EnvironmentIdentity:
		return "environment"
	case reference.Calls != local.Calls || reference.TimeAdvances != local.TimeAdvances:
		return "accounting"
	case !bytes.Equal(reference.StorageTranscript, local.StorageTranscript):
		return "storage transcript"
	case !reflect.DeepEqual(reference.StorageSnapshots, local.StorageSnapshots):
		return "snapshots"
	case !reflect.DeepEqual(reference.StorageOperations, local.StorageOperations):
		return "operations"
	case !reflect.DeepEqual(reference.StorageCrashEnumeration, local.StorageCrashEnumeration):
		return "enumerations"
	case reference.StorageImage.Identity() != local.StorageImage.Identity():
		return "image"
	case reference.RuntimeInterrupted != local.RuntimeInterrupted || !bytes.Equal(reference.InterruptedChoiceBytes, local.InterruptedChoiceBytes):
		return "interrupted choices"
	}
	return ""
}

// runStorageOwners runs one storage request through the Go-owner reference
// and the helper-owned volume and requires identical evidence.
func runStorageOwners(t *testing.T, label string, request Request, referenceImage ...*StorageImage) (Result, Result) {
	t.Helper()
	remote := request
	remote.remoteModel = true
	if len(referenceImage) != 0 {
		// The Go owner recovers only its own (Go volume) images.
		remote.StorageImage = referenceImage[0]
	}
	reference := runStorageGuest(t, remote)
	local := runStorageGuest(t, request)
	if local.GoFilesystemCalls == nil || len(local.GoFilesystemCalls) != 0 {
		t.Fatalf("%s: helper storage run called Go filesystem: %v", label, local.GoFilesystemCalls)
	}
	if reference.GoFilesystemCalls["fd_pwrite"] == 0 {
		t.Fatalf("%s: Go reference did not serve storage operations: %v", label, reference.GoFilesystemCalls)
	}
	if diff := sameStorageEvidence(reference, local); diff != "" {
		t.Fatalf("%s: helper storage evidence differs in %s: Go=%s %q Rust=%s %q", label, diff, reference.Termination, reference.Message, local.Termination, local.Message)
	}
	return reference, local
}

func TestStorageOwnerParity(t *testing.T) {
	base := storageSQLiteRequest(t)
	model := *base.Config.Storage
	model.SQLiteLockDatabases = []string{"/volume/db"}
	model.Faults = []StorageFault{{Name: "acknowledged-crash", Resource: "/volume/crash-trigger", Operation: "open", Occurrence: 1, Action: "crash-before"}}
	crashing := base
	crashing.Config.Storage = &model
	setupReference, setup := runStorageOwners(t, "setup", storageGuestOperation(crashing, "setup-live", "WAL", nil))
	requireStorageGuestExit(t, setup)
	crash := storageGuestOperation(crashing, "append-crash", "WAL", setup.StorageImage)
	crash.StorageCrashLimits = &volume.CrashEnumerationLimits{States: 32, Operations: 8, Depth: 8, Bytes: 4 << 20, Work: 1 << 16}
	crashReference, crashed := runStorageOwners(t, "crash-before", crash, setupReference.StorageImage)
	if crashed.Termination != "storage_crash" || !crashed.RuntimeInterrupted || string(crashed.Stdout) != "ACK 2\n" {
		t.Fatalf("helper-originated storage crash: %s %s %s", crashed.Termination, crashed.Message, crashed.Stdout)
	}
	// The Go reference recovers the Go image; the helper recovers the blob.
	recoverRequest := storageGuestOperation(crashing, "recover", "WAL", nil)
	referenceRecovery := recoverRequest
	referenceRecovery.remoteModel = true
	referenceRecovery.StorageImage = crashReference.StorageImage
	reopenedReference := runStorageGuest(t, referenceRecovery)
	localRecovery := recoverRequest
	localRecovery.StorageImage = crashed.StorageImage
	reopened := runStorageGuest(t, localRecovery)
	requireStorageGuestExit(t, reopened)
	if diff := sameStorageEvidence(reopenedReference, reopened); diff != "" || string(reopened.Stdout) != "INTEGRITY ok\nHISTORY 1:committed,2:committed\n" {
		t.Fatalf("reopen from helper image differs in %s: %s", diff, reopened.Stdout)
	}
	// A Go-owner image carries the same blob for a helper-owned recovery.
	crossRecovery := recoverRequest
	crossRecovery.StorageImage = crashReference.StorageImage
	if crossed := runStorageGuest(t, crossRecovery); sameStorageEvidence(reopenedReference, crossed) != "" {
		t.Fatal("helper recovery from the Go-owner image differs")
	}
	if setupReference.StorageImage.Identity() != setup.StorageImage.Identity() {
		t.Fatal("setup image identity differs")
	}
	for _, action := range []string{"io-error", "short-write", "sync-error"} {
		t.Run(action, func(t *testing.T) {
			operation := "write"
			if action == "sync-error" {
				operation = "sync"
			}
			fault := StorageFault{Name: "parity-" + action, Resource: "/volume/db-wal", Operation: operation, Occurrence: 1, Action: action}
			if action == "short-write" {
				fault.Bytes = 1
			}
			faulty := base
			faultModel := *base.Config.Storage
			faultModel.Faults = []StorageFault{fault}
			faulty.Config.Storage = &faultModel
			setup := storageGuestOperation(faulty, "setup", "WAL", nil)
			reference := setup
			reference.remoteModel = true
			referenceInitialized := runStorageGuest(t, reference)
			initialized := runStorageGuest(t, setup)
			requireStorageGuestExit(t, initialized)
			_, failed := runStorageOwners(t, action, storageGuestOperation(faulty, "append", "WAL", initialized.StorageImage), referenceInitialized.StorageImage)
			if !bytes.Contains(failed.StorageTranscript, []byte(`"fault":"parity-`+action+`"`)) {
				t.Fatalf("fault %s not realized", action)
			}
		})
	}
}

func TestStorageHelperRefusesInvalidRecoveryBeforeStart(t *testing.T) {
	base := storageSQLiteRequest(t)
	setup := runStorageGuest(t, storageGuestOperation(base, "setup", "WAL", nil))
	requireStorageGuestExit(t, setup)
	tampered := *setup.StorageImage
	tampered.fs = nil
	tampered.data = append([]byte(nil), setup.StorageImage.data...)
	tampered.data[len(tampered.data)-1] ^= 1
	request := storageGuestOperation(base, "recover", "WAL", &tampered)
	result, err := Run(t.Context(), request)
	if err != nil || result.Termination != "invalid" || len(result.Stdout) != 0 || result.Calls != 0 {
		t.Fatalf("tampered recovery image: %s %q calls=%d err=%v", result.Termination, result.Message, result.Calls, err)
	}
	other := base
	otherModel := *base.Config.Storage
	otherModel.Volumes = append([]volume.VolumeConfig(nil), otherModel.Volumes...)
	otherModel.Volumes[0].CapacityBytes--
	other.Config.Storage = &otherModel
	if _, err := Run(t.Context(), storageGuestOperation(other, "recover", "WAL", setup.StorageImage)); err == nil {
		t.Fatal("recovery image of another storage model admitted")
	}
}
