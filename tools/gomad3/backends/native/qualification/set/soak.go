package set

import (
	"slices"
	"time"
)

// SoakQualifyArguments returns the gomad qualify arguments of one determinism
// soak batch of a workload: the arguments this set runs for the workload and
// seed, with the batch's repetition count and overall deadline, the diagnostic
// trace on, and successful-execution replay off. A soak compares fresh runs, so
// it does not spend a replay on every success; the diagnostic trace is retained
// in each Campaign whether or not a success Artifact is.
func SoakQualifyArguments(manifest Manifest, workload Workload, seed, repeat uint64, overallTimeout time.Duration, artifactRoot string) []string {
	manifest.Repeat = repeat
	workload.OverallTimeout = overallTimeout.String()
	workload.ReplaySuccesses = false
	workload.SuccessArtifactLimit = 0
	workload.SuccessBytesLimit = 0
	args := workloadCommand(Spec{ArtifactRoot: artifactRoot}, manifest, workload, seed).Args
	return slices.Insert(args, 1, "--diagnostics")
}
