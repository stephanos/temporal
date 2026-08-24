package execution

import (
	"errors"
	"fmt"
)

const (
	EvidenceProfilePublicGRPC        = "public-grpc"
	EvidenceProfilePublicGRPCHistory = "public-grpc-history"
	EvidenceProfileDualHistory       = "public-grpc-history+internal-history"
	EvidenceProfileTelemetry         = "telemetry"
	EvidenceProfileInProcessHooks    = "in-process-hooks"
)

func (p EnvironmentIdentity) Validate() error {
	if p.Name == "" || p.BuildID == "" || p.ConfigurationIdentity == "" || p.IsolationIdentity == "" {
		return errors.New("profile identity, build, configuration, and isolation are required")
	}
	switch p.EvidenceProfile {
	case EvidenceProfilePublicGRPC, EvidenceProfilePublicGRPCHistory, EvidenceProfileDualHistory,
		EvidenceProfileTelemetry, EvidenceProfileInProcessHooks:
	default:
		return fmt.Errorf("unknown evidence profile %q", p.EvidenceProfile)
	}
	if p.DrivingAuthority == "" || p.ObservationAuthority == "" || p.FaultAuthority == "" || p.RetentionClass == "" {
		return errors.New("profile driving, observation, fault, and retention authority are required")
	}
	return nil
}
