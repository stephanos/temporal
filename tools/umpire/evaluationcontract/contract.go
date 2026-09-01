package evaluationcontract

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"unicode/utf8"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	SupportedMajorVersion int32 = 1
	SupportedMinorVersion int32 = 0
	checksumDomain              = "umpire.evaluation-contract/v1"

	MaximumContractBytes   = 1 << 20
	MaximumProtoJSONBytes  = 2 << 20
	MaximumInputBytes      = 16 << 20
	MaximumEvidenceRecords = 100_000
	MaximumExpressionDepth = 64
	MaximumCollectionItems = 10_000
	MaximumEvaluationWork  = 10_000_000
	MaximumDiagnosticBytes = 64 << 10
	MaximumResultBytes     = 4 << 20
	MaximumDurationMillis  = 5 * 60 * 1_000
)

var (
	deterministicMarshal = proto.MarshalOptions{Deterministic: true}
	canonicalJSONMarshal = protojson.MarshalOptions{Multiline: true, Indent: "  "}
)

// Pack converts exact canonical ProtoJSON into the deterministic protobuf contract artifact.
// A missing checksum is filled structurally; a supplied checksum must already be correct.
func Pack(canonicalProtoJSON []byte) ([]byte, error) {
	if len(canonicalProtoJSON) == 0 || len(canonicalProtoJSON) > MaximumProtoJSONBytes {
		return nil, admissionError(ErrorByteLimit, "$", "ProtoJSON has %d bytes; limit is %d",
			len(canonicalProtoJSON), MaximumProtoJSONBytes)
	}
	if !utf8.Valid(canonicalProtoJSON) {
		return nil, admissionError(ErrorSyntax, "$", "ProtoJSON is not valid UTF-8")
	}

	contract := new(umpirespb.EvaluationContract)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(canonicalProtoJSON, contract); err != nil {
		code := ErrorSyntax
		if strings.Contains(err.Error(), "unknown field") {
			code = ErrorUnknownField
		}
		return nil, admissionError(code, "$", "decode ProtoJSON: %v", err)
	}
	if err := validateProtoSurface(contract.ProtoReflect(), "$", 0); err != nil {
		return nil, err
	}
	canonical, err := CanonicalProtoJSON(contract)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(canonicalProtoJSON, canonical) {
		return nil, admissionError(ErrorNoncanonical, "$", "ProtoJSON is not exact canonical bytes")
	}

	if len(contract.GetArtifactChecksum()) == 0 {
		return seal(contract)
	}
	if err := validateContract(contract, false); err != nil {
		return nil, err
	}
	encoded, err := deterministicMarshal.Marshal(contract)
	if err != nil {
		return nil, admissionError(ErrorMalformedValue, "$", "marshal contract: %v", err)
	}
	if _, err := Admit(encoded); err != nil {
		return nil, err
	}
	return encoded, nil
}

// Admit accepts only a complete contract whose bytes are the deterministic protobuf encoding.
func Admit(encoded []byte) (*umpirespb.EvaluationContract, error) {
	if len(encoded) == 0 || len(encoded) > MaximumContractBytes {
		return nil, admissionError(ErrorByteLimit, "$", "contract has %d bytes; limit is %d",
			len(encoded), MaximumContractBytes)
	}

	contract := new(umpirespb.EvaluationContract)
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(encoded, contract); err != nil {
		return nil, admissionError(ErrorSyntax, "$", "decode protobuf: %v", err)
	}
	if err := validateProtoSurface(contract.ProtoReflect(), "$", 0); err != nil {
		return nil, err
	}
	if err := validateContract(contract, false); err != nil {
		return nil, err
	}
	expected, err := expectedChecksum(contract)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(contract.GetArtifactChecksum(), expected) {
		return nil, admissionError(ErrorChecksum, "$.artifactChecksum", "contract checksum mismatch")
	}
	canonical, err := deterministicMarshal.Marshal(contract)
	if err != nil {
		return nil, admissionError(ErrorMalformedValue, "$", "marshal contract: %v", err)
	}
	if !bytes.Equal(encoded, canonical) {
		return nil, admissionError(ErrorNoncanonical, "$", "contract is not deterministic protobuf bytes")
	}
	if int64(len(encoded)) > contract.GetLimits().GetMaxContractBytes() {
		return nil, admissionError(ErrorLimit, "$.limits.maxContractBytes",
			"contract has %d bytes; declared limit is %d", len(encoded), contract.GetLimits().GetMaxContractBytes())
	}
	return contract, nil
}

// CanonicalProtoJSON returns the sole canonical build-time representation accepted by Pack.
func CanonicalProtoJSON(contract *umpirespb.EvaluationContract) ([]byte, error) {
	if contract == nil {
		return nil, admissionError(ErrorMalformedValue, "$", "contract is required")
	}
	encoded, err := canonicalJSONMarshal.Marshal(contract)
	if err != nil {
		return nil, admissionError(ErrorMalformedValue, "$", "marshal ProtoJSON: %v", err)
	}
	return append(encoded, '\n'), nil
}

func seal(contract *umpirespb.EvaluationContract) ([]byte, error) {
	if len(contract.GetArtifactChecksum()) != 0 {
		return nil, admissionError(ErrorChecksum, "$.artifactChecksum", "checksum must be absent before sealing")
	}
	if err := validateContract(contract, true); err != nil {
		return nil, err
	}
	sealed := proto.CloneOf(contract)
	checksum, err := expectedChecksum(sealed)
	if err != nil {
		return nil, err
	}
	sealed.ArtifactChecksum = checksum
	encoded, err := deterministicMarshal.Marshal(sealed)
	if err != nil {
		return nil, admissionError(ErrorMalformedValue, "$", "marshal contract: %v", err)
	}
	if _, err := Admit(encoded); err != nil {
		return nil, err
	}
	return encoded, nil
}

func expectedChecksum(contract *umpirespb.EvaluationContract) ([]byte, error) {
	preimage := proto.CloneOf(contract)
	preimage.ArtifactChecksum = nil
	encoded, err := deterministicMarshal.Marshal(preimage)
	if err != nil {
		return nil, admissionError(ErrorMalformedValue, "$", "marshal checksum preimage: %v", err)
	}
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(checksumDomain))
	_, _ = hasher.Write([]byte{'\n'})
	_, _ = hasher.Write(encoded)
	return hasher.Sum(nil), nil
}

func validateProtoSurface(message protoreflect.Message, path string, depth int) error {
	if depth > MaximumExpressionDepth*4 {
		return admissionError(ErrorLimit, path, "protobuf message nesting exceeds structural limit")
	}
	if len(message.GetUnknown()) != 0 {
		return admissionError(ErrorUnknownField, path, "protobuf contains unknown fields")
	}
	var validationErr error
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		fieldPath := path + "." + field.JSONName()
		if field.IsList() {
			list := value.List()
			for index := 0; index < list.Len(); index++ {
				if err := validateProtoValue(field, list.Get(index), fmt.Sprintf("%s[%d]", fieldPath, index), depth); err != nil {
					validationErr = err
					return false
				}
			}
			return true
		}
		if field.IsMap() {
			validationErr = admissionError(ErrorUnsupportedOperator, fieldPath, "maps are not in the contract vocabulary")
			return false
		}
		if err := validateProtoValue(field, value, fieldPath, depth); err != nil {
			validationErr = err
			return false
		}
		return true
	})
	return validationErr
}

func validateProtoValue(field protoreflect.FieldDescriptor, value protoreflect.Value, path string, depth int) error {
	switch field.Kind() {
	case protoreflect.EnumKind:
		if field.Enum().Values().ByNumber(value.Enum()) == nil {
			return admissionError(ErrorUnsupportedEnum, path, "enum value %d is not declared", value.Enum())
		}
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if !value.Message().IsValid() {
			return admissionError(ErrorMalformedValue, path, "message is invalid")
		}
		return validateProtoSurface(value.Message(), path, depth+1)
	default:
	}
	return nil
}
