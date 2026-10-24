package kettle

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "umpire.local/kettle/gen/kettle/v1"
)

func validateKettleID(id string) error {
	if id == "" || len(id) > MaxKettleIDBytes {
		return status.Error(codes.InvalidArgument, "kettle_id must contain 1..32 bytes")
	}
	return nil
}

func validateWaterKind(o *pb.FillRequest_FillOptions) error {
	if o.GetWaterKind() != pb.FillRequest_FillOptions_WATER_KIND_TAP && o.GetWaterKind() != pb.FillRequest_FillOptions_WATER_KIND_FILTERED {
		return status.Error(codes.InvalidArgument, "water_kind must be TAP or FILTERED")
	}
	return nil
}

func validateWaterSource(o *pb.FillRequest_FillOptions) error {
	switch source := o.GetSource().(type) {
	case *pb.FillRequest_FillOptions_Tap:
		if o.GetWaterKind() == pb.FillRequest_FillOptions_WATER_KIND_TAP && strings.TrimSpace(source.Tap) != "" {
			return nil
		}
	case *pb.FillRequest_FillOptions_Cartridge:
		if o.GetWaterKind() == pb.FillRequest_FillOptions_WATER_KIND_FILTERED && len(source.Cartridge) > 0 {
			return nil
		}
	default:
	}
	return status.Error(codes.InvalidArgument, "water source must match water_kind: TAP needs tap, FILTERED needs cartridge")
}

func validateLabels(o *pb.FillRequest_FillOptions) error {
	for key, value := range o.GetLabels() {
		if key == "" || value == "" {
			return status.Error(codes.InvalidArgument, "labels require nonempty keys and values")
		}
	}
	return nil
}

func validateTags(id string, o *pb.FillRequest_FillOptions) error {
	if len(o.GetTags()) > maxTags(id) {
		return status.Error(codes.InvalidArgument, "tags exceed per-kettle scoped limit")
	}
	return nil
}

func validateTimeout(o *pb.FillRequest_FillOptions) error {
	if timeout := o.GetTimeout(); timeout != nil {
		if timeout.CheckValid() != nil || timeout.AsDuration() <= 0 || timeout.GetSeconds() > 60 || timeout.GetSeconds() == 60 && timeout.GetNanos() != 0 {
			return status.Error(codes.InvalidArgument, "timeout must be a valid positive duration at most 60 seconds")
		}
	}
	return nil
}

func validateFill(id string, o *pb.FillRequest_FillOptions, options Options) error {
	if o == nil {
		return nil
	}
	for _, validate := range []func(*pb.FillRequest_FillOptions) error{validateWaterKind, validateWaterSource} {
		if err := validate(o); err != nil {
			return err
		}
	}
	if options.Mutant != "removed-validator" {
		if err := validateLabels(o); err != nil {
			return err
		}
	}
	if err := validateTimeout(o); err != nil {
		return err
	}
	return validateTags(id, o)
}

func validateHeatingAction(action pb.HeatingAction) error {
	if action != pb.HeatingAction_HEATING_ACTION_HEAT && action != pb.HeatingAction_HEATING_ACTION_FAILOVER {
		return status.Error(codes.InvalidArgument, "heating action must be HEAT or FAILOVER")
	}
	return nil
}
