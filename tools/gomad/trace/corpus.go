package trace

import (
	sharedtrace "go.temporal.io/server/tools/common/formal/trace"
	"go.temporal.io/server/tools/gomad/virtualtime"
)

const CorpusSchema = "gomad.virtual-time-corpus/v1"

type GenerationContract = sharedtrace.GenerationContract

type Corpus = sharedtrace.Corpus[
	virtualtime.StateSnapshot,
	virtualtime.Action,
	virtualtime.ObservableDelta,
	virtualtime.RejectionCode,
]

type BehaviorTrace = sharedtrace.BehaviorTrace[
	virtualtime.StateSnapshot,
	virtualtime.Action,
	virtualtime.ObservableDelta,
]

type StepRecord = sharedtrace.StepRecord[
	virtualtime.Action,
	virtualtime.ObservableDelta,
]

type RejectionCase = sharedtrace.RejectionCase[
	virtualtime.StateSnapshot,
	virtualtime.Action,
	virtualtime.RejectionCode,
]

var codec = sharedtrace.Codec[
	virtualtime.StateSnapshot,
	virtualtime.Action,
	virtualtime.ObservableDelta,
	virtualtime.RejectionCode,
]{
	Subject:              "virtual time",
	CorpusSchema:         CorpusSchema,
	SemanticDigestDomain: "gomad.virtual-time-corpus-semantic/v1",
	CorpusDigestDomain:   "gomad.virtual-time-corpus/v1",
}

func Finalize(corpus *Corpus) error {
	return codec.Finalize(corpus)
}

func Encode(corpus Corpus) ([]byte, error) {
	return codec.Encode(corpus)
}

func Decode(data []byte) (Corpus, error) {
	return codec.Decode(data)
}
