package finite

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/umpire3/internal/subprocess"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

type CertificateCheckMeasurement struct {
	DurationNanos   int64
	PeakMemoryBytes int64
}

func CheckCertificate(
	ctx context.Context,
	command []string,
	view protocolchecker.FirstOrderView,
	certificate Certificate,
) (Receipt, error) {
	receipt, _, err := MeasureCertificateCheck(ctx, command, view, certificate)
	return receipt, err
}

func MeasureCertificateCheck(
	ctx context.Context,
	command []string,
	view protocolchecker.FirstOrderView,
	certificate Certificate,
) (Receipt, CertificateCheckMeasurement, error) {
	if len(command) == 0 {
		return Receipt{}, CertificateCheckMeasurement{},
			errors.New("canonical Lean native certificate checker command is required")
	}
	if err := certificate.Validate(view); err != nil {
		return Receipt{}, CertificateCheckMeasurement{}, err
	}
	arguments, err := certificateArguments(certificate)
	if err != nil {
		return Receipt{}, CertificateCheckMeasurement{}, err
	}
	result, err := subprocess.Run(ctx, subprocess.Request{
		Command:        append(append([]string(nil), command...), arguments...),
		Timeout:        30 * time.Second,
		MaxOutputBytes: protocolexperiment.DefaultDecodeLimit,
		Limits: subprocess.Limits{
			CPUSeconds:  30,
			MemoryBytes: 1 << 30,
		},
	})
	if err != nil {
		return Receipt{}, CertificateCheckMeasurement{},
			fmt.Errorf("run canonical Lean native certificate checker: %w", err)
	}
	receipt, err := DecodeReceipt(strings.NewReader(string(result.Output)),
		protocolexperiment.DefaultDecodeLimit, certificate)
	if err != nil {
		return Receipt{}, CertificateCheckMeasurement{}, err
	}
	return receipt, CertificateCheckMeasurement{
		DurationNanos: result.DurationNanos, PeakMemoryBytes: result.PeakMemoryBytes,
	}, nil
}

func certificateArguments(certificate Certificate) ([]string, error) {
	arguments := []string{
		certificate.Digest,
		certificate.ViewDigest,
		string(certificate.Target),
		string(certificate.Property),
		certificate.World,
		certificate.Variant,
		certificate.SemanticHash,
		strconv.Itoa(certificate.Symmetry.Replicas),
		strconv.Itoa(certificate.Statistics.ExpandedStates),
		strconv.Itoa(certificate.Statistics.RepresentativeStates),
		strconv.Itoa(len(certificate.Nodes)),
	}
	for _, node := range certificate.Nodes {
		parent := "root"
		action := "root"
		if node.Parent >= 0 {
			parent = strconv.Itoa(node.Parent)
			action = string(node.Action)
		}
		values, err := nativeStateValues(node.State)
		if err != nil {
			return nil, err
		}
		arguments = append(arguments, parent, action)
		arguments = append(arguments, values...)
	}
	return arguments, nil
}

func nativeStateValues(state protocolchecker.FirstOrderState) ([]string, error) {
	values := make(map[string]string, len(state.Fields))
	for _, binding := range state.Fields {
		values[binding.Field] = binding.Value
	}
	required := []string{"lifecycle", "task", "owner-epoch", "worker-epoch", "completion-epoch"}
	result := make([]string, len(required))
	for index, field := range required {
		value, found := values[field]
		if !found {
			return nil, fmt.Errorf("native certificate state is missing %q", field)
		}
		if field == "owner-epoch" || field == "worker-epoch" || field == "completion-epoch" {
			var err error
			value, err = leanEpoch(value, field != "owner-epoch")
			if err != nil {
				return nil, err
			}
		}
		result[index] = value
	}
	return result, nil
}

func leanEpoch(value string, optional bool) (string, error) {
	if value == "none" {
		if optional {
			return value, nil
		}
		return "", errors.New("native owner epoch cannot be none")
	}
	index, found := strings.CutPrefix(value, "epoch-")
	if !found {
		return "", fmt.Errorf("native epoch %q has no canonical Lean encoding", value)
	}
	if _, err := strconv.Atoi(index); err != nil {
		return "", fmt.Errorf("native epoch %q has no canonical Lean encoding", value)
	}
	return index, nil
}
