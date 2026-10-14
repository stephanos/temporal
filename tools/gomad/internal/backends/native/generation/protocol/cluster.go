package protocol

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

var clusterWireNames = map[string]string{
	"encodeRuntimeNetworkConfig": "EncodeNetworkConfig", "decodeRuntimeNetworkFinish": "DecodeNetworkFinish",
	"decodeRuntimeNetworkErrorWire": "DecodeNetworkError", "encodeRuntimeNetworkTransitions": "EncodeNetworkTransitions",
	"encodeRuntimeNetworkSnapshotIdentity": "EncodeNetworkSnapshotIdentity", "validNetworkTransition": "ValidNetworkTransition",
	"validNetworkEndpoint": "ValidNetworkEndpoint", "networkTransitionLane": "NetworkTransitionLane", "networkEndpointKey": "NetworkEndpointKey",
	"encodeRuntimeVolumeConfig": "EncodeVolumeConfig", "decodeRuntimeVolumeFinish": "DecodeVolumeFinish",
	"decodeRuntimeVolumeErrorWire": "DecodeVolumeError", "encodeRuntimeVolumeFrontier": "EncodeVolumeFrontier",
	"decodeRuntimeVolumeEnumeration": "DecodeVolumeEnumeration", "encodeRuntimeVolumeTransitions": "EncodeVolumeTransitions",
	"maximumProcessFrameBytes": "MaximumProcessFrameBytes", "processProtocol": "ProcessProtocol",
	"processNodeBootstrapSchema": "ProcessNodeBootstrapSchema", "processNodeTerminalSchema": "ProcessNodeTerminalSchema",
	"processFrameKind": "ProcessFrameKind", "processFrame": "ProcessFrame",
	"processNodeBootstrap": "ProcessNodeBootstrap", "processNodeTerminal": "ProcessNodeTerminal",
	"encodeProcessValue": "EncodeProcessValue", "decodeProcessValue": "DecodeProcessValue", "validateProcessFrame": "ValidateProcessFrame",
}

var clusterWireTypes = map[string]string{
	"BootID": "BootID", "NodeID": "NodeID", "VolumeID": "VolumeID", "NodeHandle": "NodeHandle", "NodeContext": "NodeContext", "OutputObservation": "OutputObservation",
	"NetworkTransitionKind": "NetworkTransitionKind", "NetworkOutcome": "NetworkOutcome", "NetworkEndpoint": "NetworkEndpoint", "NetworkTransition": "NetworkTransition",
	"NetworkNodeSnapshot": "NetworkNodeSnapshot", "NetworkLinkSnapshot": "NetworkLinkSnapshot", "NetworkListenerSnapshot": "NetworkListenerSnapshot", "NetworkConnectionSnapshot": "NetworkConnectionSnapshot",
	"NetworkSnapshot": "NetworkSnapshot", "NetworkRecord": "NetworkRecord",
	"VolumeTransitionKind": "VolumeTransitionKind", "VolumeOutcome": "VolumeOutcome", "VolumeCrashCapacity": "VolumeCrashCapacity", "VolumeTransition": "VolumeTransition",
	"VolumeEntrySnapshot": "VolumeEntrySnapshot", "VolumeStateSnapshot": "VolumeStateSnapshot", "VolumeSnapshot": "VolumeSnapshot", "VolumeRecord": "VolumeRecord",
	"VolumeCrashFrontier": "VolumeCrashFrontier", "VolumeCrashEntry": "VolumeCrashEntry", "VolumeCrashState": "VolumeCrashState", "VolumeCrashEnumeration": "VolumeCrashEnumeration",
	"runtimeNetworkError": "RuntimeNetworkError", "runtimeVolumeError": "RuntimeVolumeError", "Spec": "RuntimeConfig",
}

func clusterWireSource(source []byte) ([]byte, error) {
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "cluster.go", source, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	usesHistory := false
	ast.Inspect(file, func(node ast.Node) bool {
		if field, ok := node.(*ast.Field); ok {
			if ident, ok := field.Type.(*ast.Ident); ok && ident.Name == "Limits" {
				field.Type = &ast.SelectorExpr{X: ast.NewIdent("history"), Sel: ast.NewIdent("RuntimeLimits")}
				usesHistory = true
			}
		}
		ident, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		if name, ok := clusterWireNames[ident.Name]; ok {
			ident.Name = name
		} else if strings.HasPrefix(ident.Name, "processFrame") && len(ident.Name) > len("processFrame") {
			ident.Name = "P" + ident.Name[1:]
		} else if name, ok := clusterWireTypes[ident.Name]; ok {
			ident.Name = "history." + name
			usesHistory = true
		}
		return true
	})
	if usesHistory {
		for _, declaration := range file.Decls {
			if group, ok := declaration.(*ast.GenDecl); ok && group.Tok == token.IMPORT {
				group.Specs = append(group.Specs, &ast.ImportSpec{Path: &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote("go.temporal.io/server/tools/gomad/internal/model/cluster/internal/history")}})
				break
			}
		}
	}
	var output bytes.Buffer
	if err := format.Node(&output, files, file); err != nil {
		return nil, err
	}
	return format.Source(output.Bytes())
}
