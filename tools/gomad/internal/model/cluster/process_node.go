package cluster

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"

	"go.temporal.io/server/tools/gomad/internal/model/cluster/internal/wire"
)

func (cluster *inProcessCluster) startProcessNode(node *clusterNode, handle NodeHandle) error {
	bootstrap := wire.ProcessNodeBootstrap{
		Schema: wire.ProcessNodeBootstrapSchema, SpecSHA256: cluster.specSHA256, Boot: node.spec.Boot,
		Context: NodeContext{NodeHandle: handle, Address: node.spec.Address, Config: append([]byte(nil), node.spec.Config...)},
	}
	encoded, err := wire.EncodeProcessValue(bootstrap)
	if err != nil {
		return err
	}
	_, err = exchangeProcessFrame(wire.ProcessFrameStart, handle, encoded)
	return err
}

func activateProcessNode(handle NodeHandle) error {
	_, err := exchangeProcessFrame(wire.ProcessFrameActivate, handle, nil)
	return err
}

func waitProcessNode(handle NodeHandle) (wire.ProcessNodeTerminal, error) {
	response, err := exchangeProcessFrame(wire.ProcessFrameWait, handle, nil)
	if err != nil {
		return wire.ProcessNodeTerminal{}, err
	}
	return decodeProcessTerminal(response.Payload, handle)
}

func stopProcessNode(handle NodeHandle) (wire.ProcessNodeTerminal, error) {
	response, err := exchangeProcessFrame(wire.ProcessFrameStop, handle, nil)
	if err != nil {
		return wire.ProcessNodeTerminal{}, err
	}
	return decodeProcessTerminal(response.Payload, handle)
}

func crashProcessNode(handle NodeHandle) error {
	_, err := exchangeProcessFrame(wire.ProcessFrameCrash, handle, nil)
	return err
}

func waitCrashedProcessNode(handle NodeHandle) error {
	response, err := exchangeProcessFrame(wire.ProcessFrameWait, handle, nil)
	if err != nil {
		return err
	}
	if len(response.Payload) != 0 {
		return errors.New("crashed process simulation node returned a terminal payload")
	}
	return nil
}

func decodeProcessTerminal(encoded []byte, handle NodeHandle) (wire.ProcessNodeTerminal, error) {
	var terminal wire.ProcessNodeTerminal
	if err := wire.DecodeProcessValue(encoded, &terminal); err != nil {
		return wire.ProcessNodeTerminal{}, err
	}
	if terminal.Schema != wire.ProcessNodeTerminalSchema {
		return wire.ProcessNodeTerminal{}, fmt.Errorf("process simulation terminal schema = %q", terminal.Schema)
	}
	for _, output := range terminal.Outputs {
		if output.Handle != handle {
			return wire.ProcessNodeTerminal{}, errors.New("process simulation output identity changed")
		}
	}
	if err := validateOutputs(terminal.Outputs, MaximumObservationBytes); err != nil {
		return wire.ProcessNodeTerminal{}, err
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
	bootstrapBytes, err := processBackendBootstrap(wire.MaximumProcessFrameBytes)
	if err != nil {
		return err
	}
	var bootstrap wire.ProcessNodeBootstrap
	if err := wire.DecodeProcessValue(bootstrapBytes, &bootstrap); err != nil {
		return err
	}
	if bootstrap.Schema != wire.ProcessNodeBootstrapSchema || bootstrap.Context.Node == "" || bootstrap.Context.Incarnation == 0 {
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
	networkConfig, err := wire.EncodeNetworkConfig(runtimeConfig(spec))
	if err != nil {
		_, finishErr := runtimeDomainFinish(runtimeRun)
		return errors.Join(err, finishErr)
	}
	if err := runtimeNetworkBegin(runtimeRun, networkConfig); err != nil {
		_, finishErr := runtimeDomainFinish(runtimeRun)
		return errors.Join(err, finishErr)
	}
	volumeConfig, err := wire.EncodeVolumeConfig(runtimeConfig(spec))
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
	if _, err := exchangeProcessFrame(wire.ProcessFrameReady, bootstrap.Context.NodeHandle, nil); err != nil {
		runtimeDomainLeave(previous)
		return finishPrivateProcessModels(runtimeRun, domain, err)
	}
	activationCurrent, err := encodeProcessActivationTime(runtimeProcessTimeCurrent())
	if err != nil {
		runtimeDomainLeave(previous)
		return finishPrivateProcessModels(runtimeRun, domain, err)
	}
	activation, err := exchangeProcessFrame(wire.ProcessFrameActivated, bootstrap.Context.NodeHandle, activationCurrent)
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
	terminal := wire.ProcessNodeTerminal{Schema: wire.ProcessNodeTerminalSchema, Outputs: outputs}
	if terminalErr != nil {
		terminal.Error = boundedTerminalText(terminalErr.Error())
	}
	terminalBytes, err := wire.EncodeProcessValue(terminal)
	if err != nil {
		return err
	}
	_, err = exchangeProcessFrame(wire.ProcessFrameTerminal, bootstrap.Context.NodeHandle, terminalBytes)
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

func encodeProcessActivationTime(current int64) ([]byte, error) {
	if current < 946684800000000000 {
		return nil, errors.New("process simulation activation time is invalid")
	}
	encoded := make([]byte, 8)
	binary.BigEndian.PutUint64(encoded, uint64(current))
	return encoded, nil
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
