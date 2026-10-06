package umpire

// A design whose checks are meant to find it wrong, mixed into its object:
// `object LateRecord extends Derived(OrderRecord.rebind(...)), NegativeControl`. It is a
// deliberately wrong design, which no feature ships, kept so that the checks that must refuse it
// are seen to: the IR generator (model/irgen) refuses a negative control that no Query lifted with
// it can refute -- a `verify`, whose answer may be a counterexample, or a Query whose Run is expected
// violated -- one that a machine refines, and one that declares a refinement of its own, as a
// feature's System does.
//
// A marker is transparent to Definition IDs and to the IR: it changes no name, ID, table or answer.
trait NegativeControl:
  self: Model =>

// The real design under a fault the environment can cause, whose promise must still hold, mixed
// into its object: `object LostAnswer extends Machine[...], FailureModel`. The IR generator refuses
// a failure model that binds no fault -- an action of the actor `fault` --
// and one whose every Query expects its Run to violate the promise. It refuses as well a machine
// that binds a fault and is marked neither a failure model nor a negative control, since a fault
// says what the machine is for.
//
// A marker is transparent to Definition IDs and to the IR: it changes no name, ID, table or answer.
trait FailureModel:
  self: Model =>
