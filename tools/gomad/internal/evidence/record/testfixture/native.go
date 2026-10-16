//go:build gomad_native

package testfixture

import (
	"go.temporal.io/server/tools/gomad/internal/backends/native/transcript"
)

type Spec = transcript.Spec
type Digest = transcript.Digest
type Operation = transcript.Operation

const Deterministic = transcript.Deterministic

var Default = transcript.Default
var BoundaryManifestIdentity = transcript.BoundaryManifestIdentity
var SemanticInstrumentationIdentity = transcript.SemanticInstrumentationIdentity
var RequiredSemanticProbes = transcript.RequiredSemanticProbes
var HashArgument = transcript.HashArgument
var EncodeTranscript = transcript.EncodeTranscript
var DecodeTranscript = transcript.DecodeTranscript
var TranscriptRecordCount = transcript.TranscriptRecordCount
var SummarizeSemanticProbes = transcript.SummarizeSemanticProbes
var DecodeSemanticCoverage = transcript.DecodeSemanticCoverage
var MissingRequiredSemanticProbes = transcript.MissingRequiredSemanticProbes
