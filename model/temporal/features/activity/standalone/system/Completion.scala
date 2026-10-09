// Standalone activity completion, verified on the unchanged lifecycle.
package temporal
package features.activity
package standalone
package system

import framework.*

object Completion extends Derived(ActivitySystem.rebind())
