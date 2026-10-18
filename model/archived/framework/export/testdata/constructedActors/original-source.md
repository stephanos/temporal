Captured before production edits at task base `5e1b5e4330a0799f68637ca3952eaa7c762efe72`.
Exact source: adjacent original-source.txt (fixture.constructedactor); no plugin was loaded.
The raw complete old-reader Model is original.json. It is never generated from native conversion.

Commands (repository temporal/):
```sh
mise exec -- scala-cli --power package model/framework/dsl model/framework/realize --server=false --library -f -o .flow/tmp/fn174-actor-old-framework.jar
mise exec -- scala-cli compile model/framework/export/testdata/constructedActors --scala 3.9.0 --jvm 27 --server=false --dependency com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20 --jar .flow/tmp/fn174-actor-old-framework.jar --jar model/build/ir-scalapb.jar --print-class-path --compilation-output .flow/tmp/fn174-actor-old.jar
mise exec -- scala-cli run model/framework/export --server=false --main-class framework.export.lift -- '.flow/tmp/fn174-actor-old.jar=.flow/tmp/constructedActors/' .flow/tmp/fn174-actor-old.classpath .flow/tmp/fn174-actor-original.json 'fixture.constructedactor.ConstructedActors$package$.progress'
```
Final original handles: package 74326 exit 0, compile 1998 exit 0, reader 99261 exit 0.
The compile stdout was retained as .flow/tmp/fn174-actor-old.classpath.
Final reader diagnostics: /tmp/fn174-actor-old-reader.log. Early setup-only logs at that path
were overwritten: reserved actors package/layout and duplicate Action names, not Actor proof.
Native semantic red: /tmp/fn174-actor-native-red.log, handle 73567 exit 1, both comparisons failed.
