package temporal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.temporal.io/server/tests/umpire3/execution/participant"
)

func SDKActivity(ctx context.Context, operation participant.Operation) error {
	if operation.SDKOperation == participant.SDKRetry && activity.GetInfo(ctx).Attempt == 1 {
		return temporal.NewApplicationError("umpire3 retry probe", "umpire3-retryable")
	}
	return nil
}

func SDKChildWorkflow(ctx workflow.Context, operation participant.Operation) error {
	if operation.SDKOperation == participant.SDKCancel {
		return workflow.Await(ctx, func() bool { return false })
	}
	return nil
}

func SDKContinueAsNewWorkflow(ctx workflow.Context, continued bool) error {
	if continued {
		return nil
	}
	return workflow.NewContinueAsNewError(ctx, SDKContinueAsNewWorkflow, true)
}

func SDKImmediateWorkflow(workflow.Context) error { return nil }

func responseStatus(mode participant.ResponseMode) string {
	switch mode {
	case participant.ResponseAsynchronous:
		return "accepted"
	case participant.ResponseDeferred:
		return "deferred"
	default:
		return "completed"
	}
}

func detachedResponse(mode participant.ResponseMode) bool {
	return mode == participant.ResponseAsynchronous || mode == participant.ResponseDeferred
}

func cloneResults(source map[string]participant.Result) map[string]participant.Result {
	result := make(map[string]participant.Result, len(source))
	for identifier, value := range source {
		value.Lineage = append([]string(nil), value.Lineage...)
		result[identifier] = value
	}
	return result
}

func updateID(programID, commandID string) string {
	return "umpire3-" + safeSDKName(programID+"-"+commandID)
}

func safeSDKName(value string) string {
	var builder strings.Builder
	for _, character := range value {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' || character == '-' {
			builder.WriteRune(character)
		} else {
			builder.WriteByte('-')
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "participant"
	}
	if len(result) > 120 {
		digest := sha256.Sum256([]byte(result))
		return result[:80] + "-" + hex.EncodeToString(digest[:8])
	}
	return result
}
