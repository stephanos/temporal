package framework.actorclient

import framework.dsl.*
import fixture.actorhelpers.Helpers

val subclass = action(new Helpers.Base)
val subclassFair = assume("subclass").fair(subclass)
val subclassProgress = Subject.leadsTo("subclass")(s => !s, s => s, 1, subclassFair)
