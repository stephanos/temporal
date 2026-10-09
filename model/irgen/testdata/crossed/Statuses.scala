// A phase enum that declares the status of its cases, one case of which declares none (fn-135.5):
// the parameter has no default, so the build refuses the case at its line.
package fixture.crossed

import framework.*

enum Status derives Finite:
  case statusIdle, statusBusy

enum Stage(val status: Status) extends Recorded[Status] derives Finite:
  case idle extends Stage(Status.statusIdle)
  case busy
