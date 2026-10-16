# Independent Inbox oracle

The capture ran at commit `668de703ad643ee8475b042e5f5c6b59c75dae8e`, tree
`deb8dd12b432f40e835105b0ae9281b7ca759ee6`, before production edits.
The existing declaration reader produced complete raw Models. These fixtures compare
every Type and Channel field, removing only each message's Position. Type and
Channel sequence order, record field order and enum case order remain exact.

Original source is retained byte-for-byte in these authored fixtures:

| Source | SHA256 |
| --- | --- |
| Inboxes.scala | db030bdbf593623a3911474cd9fe59d3b4335e9b283e7240494ea3b3cf0eb829 |
| Forms.scala | 9dc4ebb35472ff24a8d45a7bc75e180aed1339002037091aec7069a7b5b1cf6c |
| Helpers.scala | a84a52e13ade83ac0d3ea1b7f534c6e49700af92e64b2ce6bb2425c5ac3e8552 |
| Client.scala | c93a2f5144b1bef98ebb3a21f3bdbb7df92c0c120748eb78587afdf0b922569f |

The complete raw outputs are `original.json` (SHA256
`0370781813eeba8a33db76afeadb8d0770ea555aa35de9deb5ce0fd8b263f69a`),
`alias-old.json` (`73b8b522c9d53734446f553c9a243ba43dc79fe8345412b21860c5c85c7fd928`)
and `helper-old.json`
(`5cbb17762afd2b676058f039358890ee27603ea15264948e510917d8f663e765`).
The first contains seven types and six channels. Aliases and helper jars each
contribute two types and one channel.

## Capture commands and original handles

The first fixture attempt, session 32783, failed compilation with E161 anonymous
Given name collisions. Its retained log is `/tmp/fn174-inbox-old-freeze.log`.
It supplies no semantic evidence.

Session 52267 ran the corrected positive fixture with this exact command and exited
0 with one passing case, before production changes:

```sh
FN174_INBOX_FREEZE=1 mise exec -- scala-cli test model/framework/export --server=false --test-only framework.export.ConstructedInboxesSuite
```

Its output is `/tmp/fn174-inbox-old-freeze2.log`. The capture copied source,
raw output, classpath and jars into `.flow/tmp/fn174-inbox-freeze` with
create-only guards. The independent build packaged framework DSL/realization
with `--library` and the plugin with `--assembly --preamble=false
--main-class-in-manifest=false --provided org.scala-lang:scala3-compiler_3`.
Fixture compilation used Scala 3.9.0, JVM 27, `--server=false`, the packaged
framework and generated IR jar, ScalaPB runtime 0.11.20, and the absolute plugin
jar path. The positive capture preceded the later strict-fixture flags.

Session 20244 ran the same suite command under the same capture variable, with
strict fixture options `-Werror -deprecation -feature -unchecked -Wunused:all
-Wvalue-discard -Wnonunit-statement -Wsafe-init -Wimplausible-patterns
-language:strictEquality`. Its log is
`/tmp/fn174-inbox-old-controls-red.log`. Old alias and helper-jar controls
passed; duplicate record storage, duplicate storage across enum cases and
missing lexical ownership each returned reader exit 1. The suite itself exited
1 because the separate native consumer lacked a Given import (E172). That setup
failure is not the semantic red.

Retained capture jar hashes are:

| Jar | SHA256 |
| --- | --- |
| framework.jar | ccde20b407edf3caff5d3a537c8ddaa710522b041cdadc463b633649b37492af |
| reify.jar | 430dfe8fc75b3dbf14e1c365ddd154193a02d624d1ee0ab68e9dbe9270ac468a |
| original.jar | e9e5aa508e04011344910fda9440d6e50029c364bd8ab70c38bd197186943fa4 |
| forms.jar | e23ce682bc2420f87ed77225ce2ad60e2fb196ac0edb675a8e1a3e804d73127f |
| helper.jar | c824c5d444e5fdbd7b0df9b87248a21199a5dcf7eec36c7934175bdc10d3f75a |
| client.jar | 96c205b8b834d0881653a8429abaab0996300585255f9f11eeaa7ebf0e0aea85 |

## Native pre-change observation

Session 42694 compiled the corrected consumer against frozen artifacts and
observed a runtime placeholder assertion failure. Its relative consumer plugin
path emitted "No plugin in path"; this attempt is qualified and retained in
`native-create-baseline.log`, never overwritten.

Session 34962 corrected the plugin path. Compilation exited 0 without the missing
plugin diagnostic; its JVM exited 1 at the intended assertion, printing
`TypeRef(Channel(fixture.constructedinbox.Message),UnknownFieldSet(Map()))`.
Logs are `/tmp/fn174-inbox-native-red2-compile.log` and
`/tmp/fn174-inbox-native-semantic-red2.log`. The exact compile command was:

```sh
mise exec -- scala-cli compile model/framework/export/testdata/constructedInboxes/NativeBaseline.scala --scala 3.9.0 --jvm 27 --server=false --dependency com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20 --jar .flow/tmp/fn174-inbox-freeze/framework.jar --jar .flow/tmp/fn174-inbox-freeze/original.jar --jar model/build/ir-scalapb.jar --scalac-option -Xplugin:/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn174-inbox-freeze/reify.jar --print-class-path --compilation-output .flow/tmp/fn174-inbox-freeze/native-baseline2.jar
mise exec -- java -cp "$(tail -1 /tmp/fn174-inbox-native-red2.classpath)" framework.inboxbaseline.NativeBaseline
```

The pre-change consumer source was:

```scala
package framework.inboxbaseline

import framework.dsl.*
import fixture.constructedinbox.*
import fixture.constructedinbox.given

object NativeBaseline:
  def main(args: Array[String]): Unit =
    val schema = summon[Finite[First]].schema.get
    val actual = schema.types.find(_.name == schema.identity).get.getRecord.fields.head.getType
    println(s"native-placeholder:$actual")
    assert(actual.getChannel == "fixture.constructedinbox.first", "state field still carries its message placeholder")
```

The baseline demonstrates the absent native field contribution. Structural
`Finite.schema` intentionally continues carrying placeholders after the
change; the head consumer now requests private `Signature.stateSchema` and
`stateTypes`. This receipt does not claim that the frozen raw-schema consumer
passes unchanged on the head.

## Admission qualifications

Record, enum, nested, separate same-message states, bounded/nominal opaque
messages, message/Inbox/channel aliases and helper-jar state witnesses have
actual old admission. Existing helpers can forward an already retained witness.
The nested First declaration retains its own Inbox ownership; Nested stores no
recursive root-held channel.

Direct duplicate storage is old-refused even across distinct enum cases.
Same-state distinct same-message channels have no supported authored spelling
here. The final separately compiled helper-returned Inbox catalog and catalog
alias controls are refused by both the unchanged declaration reader and native
conversion; they do not claim historical positive parity or broaden author API.

This source checkpoint leaves production activation, complete Machine
delivery/loss binding, consumer fairness, full .1/.2/.3 integration, reader
retirement, independent batch review, the nested Some(None) qualification and
R5/system/Go equivalence open.

## Native re-anchoring (fn-174)

Native state refuses an opaque Inbox message, so the `Opaque` state, its `Domains` messages, the
`left` and `right` channels and `OpaqueMachine` moved from `Inboxes.scala` to `Refused.scala`,
where the suite asserts the compiler refusal at the state's witness and at its machine. The
frozen `original.json` is unchanged; the suite leaves its `Opaque` type and its `left`/`right`
channels out of the comparison and compares every other entry exactly. `Evaluation.scala` and
`Helpers.scala` name their values-only witnesses, since the reify plugin does not compile an anonymous `new Finite[T]:`.
