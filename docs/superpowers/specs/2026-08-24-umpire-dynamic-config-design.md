# Umpire Temporal dynamic-config design

## Goal

Import Temporal's complete production dynamic-config registry into Lean and provide a small,
typed Umpire abstraction for resolving and interpreting selected settings in semantic models.
The import is mechanical and reproducible. Product meaning remains handwritten.

This slice succeeds when:

- one command regenerates the complete Lean dynamic-config projection from the initialized Go
  registry;
- the generated projection represents every current value type, default shape, filter dimension,
  and all eight Temporal precedence policies without silently dropping a production setting;
- Umpire can classify selected settings, resolve an immutable `ConfigView`, and give models typed
  access through authored interpretations;
- a small callback-admission model consumes the view; and
- Go/Lean fixtures demonstrate agreement for all eight precedence policies and constrained
  defaults.

Raw YAML parsing, arbitrary Go converter execution in Lean, mutable configuration during a trace,
and semantic classification of every imported key are not part of this slice.

## Ownership and module layout

The generated boundary follows the existing `Temporal/API` layout:

```text
model/Temporal/DynamicConfig.lean
model/Temporal/DynamicConfig/Types.lean
model/Temporal/DynamicConfig/Settings.lean
```

`tools/umpire/cmd/umpire-gen-dynamic-config` exclusively owns all three files. Each generation run rebuilds
the complete facade and directory. The generator does not incrementally update individual
settings or preserve handwritten content inside its output.

- `DynamicConfig/Types.lean` defines the structural setting, value, constraint, default, and
  precedence representations.
- `DynamicConfig/Settings.lean` declares every imported production setting, the complete ordered
  setting list, its mechanical identity, and bounded Go-computed resolution fixtures.
- `DynamicConfig.lean` imports the two child modules and is the public structural import.

The handwritten Umpire layer remains outside the generator-owned boundary:

```text
model/Temporal/Experiment/Config.lean
model/Temporal/Experiment/ConfigTests.lean
```

This preserves the same structural-versus-semantic boundary as the Protobuf import: generated
Temporal facts do not assign product meaning, sampling behavior, or model relevance.

## Generated structural model

The generated Lean types represent the information available from Temporal's Go declarations. The
public structural vocabulary is `ConfigKey`, `ConfigValueType`, `ConfigValue`, `ConfigConstraints`,
`ConfigPrecedence`, `ConfigDefault`, and `SettingDeclaration`.

### Keys, types, and values

A config key is Temporal's normalized, case-insensitive registry key. A value schema distinguishes:

- boolean;
- integer;
- floating point, retained in a canonical lossless textual representation;
- string;
- duration, represented as signed nanoseconds;
- map or structurally encoded value; and
- a named Go result type for a typed or custom-converter setting.

`ConfigValue` is a corresponding tagged value. Supported maps and structs use deterministic
canonical structured data. A custom Go result type does not cause Lean declarations for arbitrary
Go structs to be synthesized.

The setting also records its codec class:

- scalar conversion;
- structural conversion; or
- custom converter.

The codec class describes the Go declaration. It does not claim that Lean implements the Go
converter.

### Constraints and precedence

Constraints contain optional values for every current Temporal filter dimension:

- namespace;
- namespace ID;
- task-queue name;
- task-queue type;
- shard ID;
- task type;
- destination; and
- CHASM task type.

The generated model defines all eight current precedence policies:

1. global;
2. namespace;
3. namespace ID;
4. task queue;
5. shard ID;
6. task type;
7. destination; and
8. CHASM task type.

Each policy owns its exact ordered constraint shapes. In particular, task-queue precedence retains
the five Temporal levels and destination precedence retains its four levels. Matching is exact:
set and unset dimensions must agree, just as they do in `dynamicconfig.Collection`.

### Defaults

A generated default has one of three forms:

- one concrete canonical value;
- constrained defaults, each carrying exact constraints and a canonical value; or
- an opaque default carrying the Go type and a deterministic reason that faithful projection was
  unavailable.

An unsupported default is not grounds to omit its setting or fail full generation. The setting
remains visible in the complete catalog. It cannot be resolved from its default for model use until
an authored interpretation supplies a canonical value. Unsupported precedence, missing key/type
metadata, malformed constrained defaults, and other structural corruption do fail generation.

### Setting declaration and identity

Each setting declaration records:

- normalized key;
- value schema and Go result type identity;
- codec class;
- default form;
- precedence policy;
- description; and
- mechanical provenance needed to diagnose registry projection failures.

The complete catalog is sorted by normalized key and elaborates only if keys are unique and every
setting is structurally coherent. Mechanical semantic identity includes the key, schema, codec,
default, and precedence policy. Documentation text remains inspectable metadata but does not alter
model meaning by itself.

## Registry-driven generator

`umpire-gen-dynamic-config` reads initialized Go values rather than reconstructing Go evaluation from
syntax.

### Registry metadata

`common/dynamicconfig` gains a narrow read-only metadata surface. Registered setting types expose
the information needed for projection: key, description, precedence, result type, codec class, and
concrete or constrained defaults. Registry snapshots return immutable copies and preserve the
existing rule that production settings are created during static initialization.

The existing `cmd/tools/gendynamicconfig` template remains the authority for generated setting
families. Its output will populate metadata consistently for the scalar aliases, structural typed
settings, custom converters, and constrained-default variants. Existing setting behavior and
comments remain unchanged.

### Package discovery and export

The generator analyzes production Go packages to identify packages that register dynamic-config
settings. It excludes test files and fails if a discovered production registration site cannot be
loaded. It then creates a temporary helper inside the module so normal Go `internal` import rules
apply, blank-imports all discovered setting packages, and reads the initialized registry snapshot.

Typed package analysis is only responsible for finding packages to initialize. The runtime
registry remains authoritative for the settings and their evaluated metadata. This avoids creating
a second evaluator for computed Go defaults, generic types, and constructor behavior.

### Rendering and publication

The generator sorts and validates its in-memory projection before rendering. It writes a complete
candidate output under a temporary root, verifies the expected file set and Lean syntax, and only
then publishes the facade and directory. A failed run leaves the prior generated output intact and
must not report success after a partial publication.

Successful repeated generation from the same registry is byte-identical. Full generation rejects
duplicate normalized keys, missing production packages, unsupported precedence values, incoherent
defaults, nondeterministic structured encodings, unexpected output paths, and invalid Lean.

## Authored Umpire abstraction

`Temporal.Experiment.Config` gives semantic models a checked interface over the generated catalog.
It has three distinct authored concepts.

### Setting classification

`SettingClassification` identifies a generated key and records a non-empty set of possible impact
classes:

- feature;
- validation;
- externally visible semantics;
- timing;
- topology;
- performance; and
- observability.

The complete mechanical catalog may contain explicitly unclassified keys. A key must be classified
before a semantic model can declare a use of it. This slice classifies only the representative
examples; it does not infer classification from descriptions or require a manual audit of every
generated setting.

### Typed interpretation

`ConfigInterpretation α` connects one generated key to a model-owned type. It records the expected
generated value schema, a checked decoder, an optional canonical replacement for an opaque
mechanical default, and a semantic digest. An opaque-default replacement is bound to the imported
default metadata it interprets so drift in the Go default invalidates the interpretation.
Interpretation is the deliberate boundary for product meaning and for custom typed values such as
callback address rules.

An interpretation does not import or execute a Go converter. It consumes a canonical structural
value and either produces the model value or a structured interpretation error. A meaning-bearing
change to the decoder contract or interpreted schema changes its semantic digest.

### Config use

`ConfigUse α` gives a particular consumer a stable identity and combines:

- the generated key;
- its classification and typed interpretation;
- the resolution context required by the key's precedence policy;
- its sampling point; and
- its change effect.

Sampling points cover live access, entity creation, request, task, and process startup. Change
effects cover next read, new entities only, and restart required.

Timing belongs to the use rather than globally to the key. The same Temporal setting may be cached
at startup by one component and called through a property function for every request by another.
Key-level classification describes possible impacts; a use states the behavior the model actually
depends on.

Sampling and change metadata are descriptive in this slice. A `ConfigView` remains immutable for
one experiment. A later design may introduce an explicit config-change action when behavior across
a live update is itself under test.

## Resolution and `ConfigView`

Resolution accepts:

- the generated complete settings;
- canonical constrained overrides;
- the set of requested config uses; and
- each use's concrete resolution context.

It validates all uses before model planning or transition enumeration. Validation rejects unknown
keys, missing classifications or interpretations, schema disagreement, illegal constraint
dimensions, duplicate values for the same exact constraints, and incomplete contexts.

For each use, resolution builds the ordered exact constraints for its generated precedence policy.
At each level it considers an explicit override before a constrained default at that same level,
then moves to the next level. This retains Temporal's constrained-default interleaving. A simple
default is used only when no applicable override is selected. Selecting an opaque default fails
resolution unless the checked interpretation supplies a canonical replacement bound to that exact
imported default.

Raw YAML values and failed arbitrary Go conversion are outside this Lean boundary. Overrides have
already been converted into canonical structural values. Cross-language agreement therefore covers
accepted canonical values, exact constraints, precedence, and defaults; it does not claim an
implementation of every custom Go parser or fallback diagnostic.

The resolver returns an immutable `ConfigView` keyed by config-use identity rather than only by
setting key. This permits the same key to be resolved for multiple contexts or consumers in one
experiment. Each resolved entry records:

- use identity and normalized key;
- canonical value;
- resolution context and matched constraints;
- whether the value came from an override or default;
- catalog and interpretation digests;
- sampling point; and
- change effect.

Typed reads require the original checked `ConfigUse α`; arbitrary string lookup is not the model
interface. A model may project the view once into a smaller domain-specific configuration record
and pass that record through its existing target setup. Transition kernels never consume raw
override maps.

## Representative examples

The authored examples use real imported settings:

| Setting | Mechanical shape | Authored meaning | Sampling and effect |
| --- | --- | --- | --- |
| `history.enablechasmcallbacks` | namespace boolean | feature, externally visible semantics | entity creation, new entities only |
| `callback.maxperexecution` | namespace integer | validation | request, next read |
| `callback.request.timeout` | destination duration | timing | task, next read |
| `callback.allowedaddresses` | namespace custom typed | validation, externally visible semantics | request, next read |
| `matching.updateackinterval` | task-queue duration with constrained defaults | timing, performance | task, next read |
| `matching.workerregistrynumbuckets` | global integer | topology, performance | process startup, restart required |

Additional bounded fixtures select real namespace-ID, shard-ID, task-type, and CHASM-task-type
settings so every precedence policy is exercised even when that key has no authored semantic
classification.

### Callback-admission model

A small isolated callback model demonstrates consumption. Its setup contains a resolved view or a
typed callback configuration projected from that view. Attachment and dispatch transitions use:

- the CHASM callback feature selection captured at entity creation;
- the per-request maximum callback count;
- interpreted address rules for admission; and
- the destination-specific request timeout for dispatch timing.

The model is intentionally bounded and pure. It demonstrates that different snapshots change
model outcomes while one trace remains pinned to one snapshot. It does not allocate a Temporal
environment, execute callbacks, simulate process restart, or change dynamic config mid-trace.

## Diagnostics

Generator diagnostics identify the stage, package or setting key, offending metadata, and source
error. They distinguish discovery, registry initialization, projection, rendering, validation, and
publication failures.

Lean config diagnostics are deterministic structured values with a stable kind, config-use
identity, key, offending value or context, and related identities. Required error kinds cover:

- unknown or unclassified key;
- missing or incompatible interpretation;
- value-schema mismatch;
- illegal or duplicate constraints;
- missing resolution context;
- opaque default selected;
- malformed config use; and
- interpretation failure.

Invalid configuration fails before target planning. It is not represented as an unsatisfiable
behavior, model outcome, generic formatted error, or successful default resolution.

## Generated drift and commands

The Make interface is:

```sh
make umpire-gen-dynamic-config
make umpire-check-dynamic-config
```

Generation runs:

```sh
mise exec -- go run -tags test_dep ./tools/umpire/cmd/umpire-gen-dynamic-config --output-root model
```

The check target performs the same complete generation under a temporary output root, requires an
empty diff against all three retained files, and runs the focused Go and Lean verification. It is a
drift check, not a partial catalog updater.

The root Lean build imports `Temporal.DynamicConfig`. The focused experiment test root includes
`Temporal.Experiment.ConfigTests` so config abstractions cannot silently drift away from the model
project.

## Verification

### Go verification

Focused Go tests cover:

- metadata for scalar, structural, custom-converter, and constrained-default constructors;
- all eight precedence policies and filter dimensions;
- evaluated concrete defaults and deterministic opaque-default reasons;
- production-package discovery, test-setting exclusion, and failure on unloadable registrations;
- complete deterministic rendering;
- repeated byte-identical generation;
- output-path validation; and
- preservation of the previous generated tree on failure.

A fixture registry provides every mechanical shape without depending only on whichever combinations
happen to exist in the current production catalog.

### Go/Lean parity

The tool contains a bounded, hand-selected set of resolution inputs using real Temporal settings.
Go computes their expected values through the real setting property functions and
`dynamicconfig.Collection`. The generator emits those inputs and expected results into
`Settings.lean`. Lean tests apply the authored resolver and require equality.

Parity cases cover global, namespace, namespace-ID, task-queue, shard-ID, task-type, destination,
and CHASM-task-type precedence; exact unset dimensions; more-specific and fallback matches; and
explicit-value versus constrained-default ordering.

### Lean verification

`ConfigTests.lean` covers:

- generated catalog uniqueness and structural well-formedness;
- presence of all eight policies;
- legal and illegal constraint shapes;
- defaults, constrained defaults, and opaque-default rejection;
- unknown, unclassified, and wrongly interpreted settings;
- source-order independence of overrides and uses;
- deterministic `ConfigView` metadata and semantic identity;
- the same key used by different consumers or contexts;
- every Go/Lean parity fixture;
- callback address interpretation;
- callback-model outcome changes between snapshots; and
- snapshot immutability within one trace, including no restart or live-update simulation.

## Scope boundaries

This design deliberately excludes:

- parsing Temporal dynamic-config YAML in Lean;
- translating or proving arbitrary Go custom converters;
- automatically inferring product impact, sampling points, or restart behavior from descriptions or
  call sites;
- requiring semantic classifications for every generated key;
- changing configuration during an experiment;
- runtime environment presets and application of overrides to a live Temporal server;
- a public Umpire config CLI; and
- changes to the existing Protobuf API importer.

These boundaries keep the generated module mechanically complete while ensuring model claims are
made only through explicit, checked, handwritten interpretations.
