package gomadv3sim

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

func (cluster *inProcessCluster) startProcessNode(node *clusterNode, handle NodeHandle) error {
	bootstrap := processNodeBootstrap{
		Schema: processNodeBootstrapSchema, SpecSHA256: cluster.specSHA256, Boot: node.spec.Boot,
		Context: NodeContext{NodeHandle: handle, Address: node.spec.Address, Config: append([]byte(nil), node.spec.Config...)},
	}
	encoded, err := encodeProcessValue(bootstrap)
	if err != nil {
		return err
	}
	_, err = exchangeProcessFrame(processFrameStart, handle, encoded)
	return err
}

func activateProcessNode(handle NodeHandle) error {
	_, err := exchangeProcessFrame(processFrameActivate, handle, nil)
	return err
}

func waitProcessNode(handle NodeHandle) (processNodeTerminal, error) {
	response, err := exchangeProcessFrame(processFrameWait, handle, nil)
	if err != nil {
		return processNodeTerminal{}, err
	}
	return decodeProcessTerminal(response.Payload, handle)
}

func stopProcessNode(handle NodeHandle) (processNodeTerminal, error) {
	response, err := exchangeProcessFrame(processFrameStop, handle, nil)
	if err != nil {
		return processNodeTerminal{}, err
	}
	return decodeProcessTerminal(response.Payload, handle)
}

func crashProcessNode(handle NodeHandle) error {
	_, err := exchangeProcessFrame(processFrameCrash, handle, nil)
	return err
}

func waitCrashedProcessNode(handle NodeHandle) error {
	response, err := exchangeProcessFrame(processFrameWait, handle, nil)
	if err != nil {
		return err
	}
	if len(response.Payload) != 0 {
		return errors.New("crashed process simulation node returned a terminal payload")
	}
	return nil
}

func decodeProcessTerminal(encoded []byte, handle NodeHandle) (processNodeTerminal, error) {
	var terminal processNodeTerminal
	if err := decodeProcessValue(encoded, &terminal); err != nil {
		return processNodeTerminal{}, err
	}
	if terminal.Schema != processNodeTerminalSchema {
		return processNodeTerminal{}, fmt.Errorf("process simulation terminal schema = %q", terminal.Schema)
	}
	for _, output := range terminal.Outputs {
		if output.Handle != handle {
			return processNodeTerminal{}, errors.New("process simulation output identity changed")
		}
	}
	if err := validateOutputs(terminal.Outputs, MaximumObservationBytes); err != nil {
		return processNodeTerminal{}, err
	}
	return terminal, nil
}

func runPrivateProcessNodeIfPresent(ctx context.Context, spec Spec) bool {
	if spec.Backend != BackendProcess || !processBackendAvailable() || processBackendRole() != processRoleNode {
		return false
	}
	if err := runPrivateProcessNode(ctx, spec); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
	return true
}

func runPrivateProcessNode(ctx context.Context, spec Spec) error {
	bootstrapBytes, err := processBackendBootstrap(maximumProcessFrameBytes)
	if err != nil {
		return err
	}
	var bootstrap processNodeBootstrap
	if err := decodeProcessValue(bootstrapBytes, &bootstrap); err != nil {
		return err
	}
	if bootstrap.Schema != processNodeBootstrapSchema || bootstrap.Context.Node == "" || bootstrap.Context.Incarnation == 0 {
		return errors.New("process simulation node bootstrap is incomplete")
	}
	specSHA256, err := hashSpec(spec)
	if err != nil {
		return err
	}
	if bootstrap.SpecSHA256 != specSHA256 {
		return errors.New("process simulation node bootstrap specification changed")
	}
	var selected *NodeSpec
	for index := range spec.Nodes {
		if spec.Nodes[index].ID == bootstrap.Context.Node {
			selected = &spec.Nodes[index]
			break
		}
	}
	if selected == nil || selected.Boot != bootstrap.Boot || selected.Address != bootstrap.Context.Address || !bytesEqual(selected.Config, bootstrap.Context.Config) {
		return errors.New("process simulation node bootstrap identity changed")
	}
	boot, ok := RegisteredBoot(bootstrap.Boot)
	if !ok {
		return errors.New("process simulation node boot is unregistered")
	}

	runtimeRun, err := runtimeDomainBegin(spec.Limits.ObservationBytes, spec.Limits.ScenarioActions)
	if err != nil {
		return err
	}
	networkConfig, err := encodeRuntimeNetworkConfig(spec)
	if err != nil {
		_, finishErr := runtimeDomainFinish(runtimeRun)
		return errors.Join(err, finishErr)
	}
	if err := runtimeNetworkBegin(runtimeRun, networkConfig); err != nil {
		_, finishErr := runtimeDomainFinish(runtimeRun)
		return errors.Join(err, finishErr)
	}
	volumeConfig, err := encodeRuntimeVolumeConfig(spec)
	if err != nil {
		_, networkErr := runtimeNetworkFinish(runtimeRun)
		_, finishErr := runtimeDomainFinish(runtimeRun)
		return errors.Join(err, networkErr, finishErr)
	}
	if err := runtimeVolumeBegin(runtimeRun, volumeConfig); err != nil {
		_, networkErr := runtimeNetworkFinish(runtimeRun)
		_, finishErr := runtimeDomainFinish(runtimeRun)
		return errors.Join(err, networkErr, finishErr)
	}
	domain, err := runtimeDomainRegister(runtimeRun, bootstrap.Context.Node, bootstrap.Context.Address, bootstrap.Context.Incarnation)
	if err != nil {
		return finishPrivateProcessModels(runtimeRun, 0, err)
	}
	if err := runtimeVolumeRegister(domain); err != nil {
		return finishPrivateProcessModels(runtimeRun, domain, err)
	}
	previous, err := runtimeDomainEnter(domain)
	if err != nil {
		return finishPrivateProcessModels(runtimeRun, domain, err)
	}
	if _, err := exchangeProcessFrame(processFrameReady, bootstrap.Context.NodeHandle, nil); err != nil {
		runtimeDomainLeave(previous)
		return finishPrivateProcessModels(runtimeRun, domain, err)
	}
	activation, err := exchangeProcessFrame(processFrameActivated, bootstrap.Context.NodeHandle, nil)
	if err != nil {
		runtimeDomainLeave(previous)
		return finishPrivateProcessModels(runtimeRun, domain, err)
	}
	current, err := decodeProcessActivationTime(activation.Payload)
	if err != nil {
		runtimeDomainLeave(previous)
		return finishPrivateProcessModels(runtimeRun, domain, err)
	}
	if err := runtimeProcessTimeAdvance(current); err != nil {
		runtimeDomainLeave(previous)
		return finishPrivateProcessModels(runtimeRun, domain, err)
	}
	bootCtx, stop := context.WithCancel(ctx)
	go func() {
		if processBackendWaitStop() == nil {
			stop()
		}
	}()
	bootErr := boot(bootCtx, bootstrap.Context)
	stopped := bootCtx.Err() != nil
	stop()
	runtimeDomainLeave(previous)
	if stopped && errors.Is(bootErr, context.Canceled) {
		bootErr = nil
	}
	cleanupErr := errors.Join(runtimeVolumeRevoke(domain, true, false), runtimeNetworkRevoke(domain, true), runtimeDomainRevoke(domain))
	_, volumeErr := runtimeVolumeFinish(runtimeRun)
	_, networkErr := runtimeNetworkFinish(runtimeRun)
	outputs, runtimeErr := runtimeDomainFinish(runtimeRun)
	terminalErr := errors.Join(bootErr, cleanupErr, volumeErr, networkErr, runtimeErr)
	terminal := processNodeTerminal{Schema: processNodeTerminalSchema, Outputs: outputs}
	if terminalErr != nil {
		terminal.Error = boundedTerminalText(terminalErr.Error())
	}
	terminalBytes, err := encodeProcessValue(terminal)
	if err != nil {
		return err
	}
	_, err = exchangeProcessFrame(processFrameTerminal, bootstrap.Context.NodeHandle, terminalBytes)
	return err
}

func decodeProcessActivationTime(encoded []byte) (int64, error) {
	if len(encoded) != 8 {
		return 0, errors.New("process simulation activation time is invalid")
	}
	current := int64(binary.BigEndian.Uint64(encoded))
	if current < 946684800000000000 {
		return 0, errors.New("process simulation activation time is invalid")
	}
	return current, nil
}

func finishPrivateProcessModels(runtimeRun, domain uint64, source error) error {
	var revokeErr error
	if domain != 0 {
		revokeErr = runtimeDomainRevoke(domain)
	}
	_, volumeErr := runtimeVolumeFinish(runtimeRun)
	_, networkErr := runtimeNetworkFinish(runtimeRun)
	_, runtimeErr := runtimeDomainFinish(runtimeRun)
	return errors.Join(source, revokeErr, volumeErr, networkErr, runtimeErr)
}

func bytesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
