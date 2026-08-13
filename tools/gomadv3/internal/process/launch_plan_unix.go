//go:build unix

package process

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"syscall"
)

type launchStage uint8

const (
	supervisorStage launchStage = iota
	bootstrapStage
	targetStage
)

type launchCapabilities struct {
	ioTranscript  bool
	readOnlyMount bool
	choiceTrace   bool
}

type resourceName string

const (
	controlResource           resourceName = "supervisor-control"
	reportResource            resourceName = "supervisor-report"
	stdoutResource            resourceName = "target-stdout"
	stderrResource            resourceName = "target-stderr"
	supervisorRequestResource resourceName = "supervisor-request"
	bootstrapRequestResource  resourceName = "bootstrap-request"
	activationResource        resourceName = "bootstrap-activation"
	readinessResource         resourceName = "bootstrap-readiness"
	worldConfigResource       resourceName = "world-config"
	worldRecordResource       resourceName = "world-record"
	identityResource          resourceName = "target-identity"
	ioConfigResource          resourceName = "io-config"
	ioTranscriptResource      resourceName = "io-transcript"
	ioTerminalResource        resourceName = "io-terminal"
	ioExpectedResource        resourceName = "io-expected"
	ioROMountRequestResource  resourceName = "io-ro-mount-request"
	ioROMountResponseResource resourceName = "io-ro-mount-response"
	choiceTraceResource       resourceName = "choice-trace"
	choiceTerminalResource    resourceName = "choice-terminal"
)

const (
	controlFD                    = 3
	reportFD                     = 4
	stdoutFD                     = 5
	stderrFD                     = 6
	requestFD                    = 7
	worldRecordFD                = 8
	targetIdentityFD             = 9
	ioTranscriptFD               = 10
	ioTerminalFD                 = 11
	ioExpectedFD                 = 12
	ioROMountRequestFD           = 13
	ioROMountResponseFD          = 14
	bootstrapRequestFD           = 3
	bootstrapActivationFD        = 4
	bootstrapReadinessFD         = 5
	bootstrapWorldConfigFD       = 6
	bootstrapWorldRecordFD       = 7
	bootstrapIdentityFD          = 8
	bootstrapIOTranscriptFD      = 9
	bootstrapIOTerminalFD        = 10
	bootstrapIOExpectedFD        = 11
	bootstrapIOROMountRequestFD  = 12
	bootstrapIOROMountResponseFD = 13
	targetWorldConfigFD          = 3
	targetWorldRecordFD          = 4
	targetIOConfigFD             = 5
	targetIOTranscriptFD         = 6
	targetIOTerminalFD           = 7
	targetIOExpectedFD           = 8
	targetIOROMountRequestFD     = 9
	targetIOROMountResponseFD    = 10
)

type descriptorSpec struct {
	resource               resourceName
	supervisorFD           int
	bootstrapFD            int
	targetFD               int
	ioTranscript           bool
	readOnlyMount          bool
	choiceTrace            bool
	closeOnSupervisorStart bool
	closeOnBootstrapStart  bool
}

type descriptorBinding struct {
	resource resourceName
	fd       int
}

var launchDescriptorSpecs = []descriptorSpec{
	{resource: controlResource, supervisorFD: controlFD, closeOnSupervisorStart: true},
	{resource: reportResource, supervisorFD: reportFD, closeOnSupervisorStart: true},
	{resource: stdoutResource, supervisorFD: stdoutFD, closeOnSupervisorStart: true},
	{resource: stderrResource, supervisorFD: stderrFD, closeOnSupervisorStart: true},
	{resource: supervisorRequestResource, supervisorFD: requestFD, closeOnSupervisorStart: true},
	{resource: bootstrapRequestResource, bootstrapFD: bootstrapRequestFD, closeOnBootstrapStart: true},
	{resource: activationResource, bootstrapFD: bootstrapActivationFD, closeOnBootstrapStart: true},
	{resource: readinessResource, bootstrapFD: bootstrapReadinessFD, closeOnBootstrapStart: true},
	{resource: worldConfigResource, bootstrapFD: bootstrapWorldConfigFD, targetFD: targetWorldConfigFD, closeOnBootstrapStart: true},
	{resource: worldRecordResource, supervisorFD: worldRecordFD, bootstrapFD: bootstrapWorldRecordFD, targetFD: targetWorldRecordFD, closeOnSupervisorStart: true, closeOnBootstrapStart: true},
	{resource: identityResource, supervisorFD: targetIdentityFD, bootstrapFD: bootstrapIdentityFD, closeOnSupervisorStart: true, closeOnBootstrapStart: true},
	{resource: ioConfigResource, targetFD: targetIOConfigFD},
	{resource: ioTranscriptResource, supervisorFD: ioTranscriptFD, bootstrapFD: bootstrapIOTranscriptFD, targetFD: targetIOTranscriptFD, ioTranscript: true, closeOnBootstrapStart: true},
	{resource: ioTerminalResource, supervisorFD: ioTerminalFD, bootstrapFD: bootstrapIOTerminalFD, targetFD: targetIOTerminalFD, ioTranscript: true, closeOnSupervisorStart: true, closeOnBootstrapStart: true},
	{resource: ioExpectedResource, supervisorFD: ioExpectedFD, bootstrapFD: bootstrapIOExpectedFD, targetFD: targetIOExpectedFD, ioTranscript: true, closeOnBootstrapStart: true},
	{resource: ioROMountRequestResource, supervisorFD: ioROMountRequestFD, bootstrapFD: bootstrapIOROMountRequestFD, targetFD: targetIOROMountRequestFD, readOnlyMount: true, closeOnSupervisorStart: true, closeOnBootstrapStart: true},
	{resource: ioROMountResponseResource, supervisorFD: ioROMountResponseFD, bootstrapFD: bootstrapIOROMountResponseFD, targetFD: targetIOROMountResponseFD, readOnlyMount: true, closeOnSupervisorStart: true, closeOnBootstrapStart: true},
	{resource: choiceTraceResource, choiceTrace: true, closeOnBootstrapStart: true},
	{resource: choiceTerminalResource, choiceTrace: true, closeOnSupervisorStart: true, closeOnBootstrapStart: true},
}

func descriptorLayout(stage launchStage, capabilities launchCapabilities) []descriptorBinding {
	if capabilities.readOnlyMount {
		capabilities.ioTranscript = true
	}
	bindings := make([]descriptorBinding, 0, len(launchDescriptorSpecs))
	for _, spec := range launchDescriptorSpecs {
		if spec.choiceTrace {
			continue
		}
		if spec.ioTranscript && !capabilities.ioTranscript || spec.readOnlyMount && !capabilities.readOnlyMount || spec.choiceTrace && !capabilities.choiceTrace {
			continue
		}
		var descriptor int
		switch stage {
		case supervisorStage:
			descriptor = spec.supervisorFD
		case bootstrapStage:
			descriptor = spec.bootstrapFD
		case targetStage:
			descriptor = spec.targetFD
		}
		if descriptor != 0 {
			bindings = append(bindings, descriptorBinding{resource: spec.resource, fd: descriptor})
		}
	}
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].fd < bindings[j].fd })
	if capabilities.choiceTrace {
		next := 3
		if len(bindings) != 0 {
			next = bindings[len(bindings)-1].fd + 1
		}
		bindings = append(bindings, descriptorBinding{resource: choiceTraceResource, fd: next}, descriptorBinding{resource: choiceTerminalResource, fd: next + 1})
	}
	return bindings
}

type resourceFiles map[resourceName]**os.File

type inheritedPipeEnd uint8

const (
	inheritRead inheritedPipeEnd = iota
	inheritWrite
)

type launchResources struct {
	capabilities launchCapabilities
	files        resourceFiles
	owned        []**os.File
}

func newLaunchResources(capabilities launchCapabilities) *launchResources {
	return &launchResources{capabilities: capabilities, files: make(resourceFiles)}
}

func (resources *launchResources) createPipe(resource resourceName, inherited inheritedPipeEnd, description string) (*os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("create %s pipe: %w", description, err)
	}
	resources.owned = append(resources.owned, &read, &write)
	if inherited == inheritRead {
		resources.files[resource] = &read
		return write, nil
	}
	resources.files[resource] = &write
	return read, nil
}

func (resources *launchResources) bind(resource resourceName, file **os.File) {
	resources.files[resource] = file
}

func (resources *launchResources) extraFiles(stage launchStage) ([]*os.File, error) {
	return filesForStage(stage, resources.capabilities, resources.files)
}

func (resources *launchResources) closeInherited(stage launchStage) error {
	return closeInheritedStage(stage, resources.capabilities, resources.files)
}

func (resources *launchResources) close() error {
	var result error
	for _, file := range resources.owned {
		if err := closeFile(file); err != nil && !errors.Is(err, os.ErrClosed) {
			result = errors.Join(result, err)
		}
	}
	return result
}

func filesForStage(stage launchStage, capabilities launchCapabilities, resources resourceFiles) ([]*os.File, error) {
	layout := descriptorLayout(stage, capabilities)
	files := make([]*os.File, 0, len(layout))
	for _, binding := range layout {
		if binding.fd != 3+len(files) {
			return nil, fmt.Errorf("descriptor plan for %q has a gap before fd %d", binding.resource, binding.fd)
		}
		file := resources[binding.resource]
		if file == nil || *file == nil {
			return nil, fmt.Errorf("descriptor resource %q is unavailable", binding.resource)
		}
		files = append(files, *file)
	}
	return files, nil
}

func closeInheritedStage(stage launchStage, capabilities launchCapabilities, resources resourceFiles) error {
	var result error
	for _, binding := range descriptorLayout(stage, capabilities) {
		spec := descriptorSpecFor(binding.resource)
		closeAfterStart := stage == supervisorStage && spec.closeOnSupervisorStart || stage == bootstrapStage && spec.closeOnBootstrapStart
		if closeAfterStart {
			result = errors.Join(result, closeFile(resources[binding.resource]))
		}
	}
	return result
}

func descriptorSpecFor(resource resourceName) descriptorSpec {
	for _, spec := range launchDescriptorSpecs {
		if spec.resource == resource {
			return spec
		}
	}
	panic(fmt.Sprintf("unknown descriptor resource %q", resource))
}

func installTargetStage(capabilities launchCapabilities) error {
	var installed []int
	for _, binding := range descriptorLayout(targetStage, capabilities) {
		if binding.resource == ioConfigResource {
			continue
		}
		source := descriptorFor(bootstrapStage, capabilities, binding.resource)
		if source == 0 {
			return errors.Join(fmt.Errorf("descriptor resource %q has no bootstrap source", binding.resource), closeDescriptors(installed...))
		}
		if err := syscall.Dup2(source, binding.fd); err != nil {
			return errors.Join(fmt.Errorf("install target descriptor %q: %w", binding.resource, err), closeDescriptors(installed...))
		}
		installed = append(installed, binding.fd)
		if source != binding.fd {
			if err := syscall.Close(source); err != nil {
				return errors.Join(fmt.Errorf("close target bootstrap descriptor %q: %w", binding.resource, err), closeDescriptors(installed...))
			}
		}
	}
	return nil
}

func descriptorFor(stage launchStage, capabilities launchCapabilities, resource resourceName) int {
	for _, binding := range descriptorLayout(stage, capabilities) {
		if binding.resource == resource {
			return binding.fd
		}
	}
	return 0
}
