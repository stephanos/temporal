package fixture.witness.compiler

import framework.dsl.{records, witness, Query}
import fixture.witness.*
import fixture.witness.worker.finish

val empty: Query = Subject.witness()
val unfinished: Query = Subject.witness(finish(Input.first))
val foreign: Query = Subject.witness(finish(Input.first)).records(Input.first)
val foreignMember: Query = Together
  .witness(Together.own(_.left, finish(Input.first)))
  .records(_.left -> Subject, Input.first)

val emptyRecords: Query = Subject.witness(finish(Input.first)).records()
val noField: Query = Together
  .witness(Together.own(_.left, finish(Input.first)))
  .records(_ => Phase.open -> Subject, Note.completed)
