package model

type Observation[A, D any] struct {
	Action            A      `json:"action"`
	PreStateIdentity  string `json:"pre_state_identity"`
	PostStateIdentity string `json:"post_state_identity"`
	ObservableDelta   D      `json:"observable_delta"`
}

type Transition[S, A, D any] struct {
	Observation[A, D]
	PostState S `json:"-"`
}
