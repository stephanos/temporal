# Final profile receiver

`final_profiles.py --config CONFIG --root ROOT` owns exactly the three IDs in
`OWNED`. A final-section entry pins Python, this reader and `--config CONFIG`;
`final.py` adds `--root ROOT`. The `final_profiles` config object binds:

- `measure`, `ordinary`, `diagnostic`: `{binary, build_receipt}` using the
  driver's sealed release build-receipt shape; the ordinary feature list is
  empty and diagnostic is exactly `diagnostic-profile`.
- `stock_go`, `terminal_module`, `terminal_module_receipt`,
  `terminal_helper_receipt`: sealed static native producer inputs.
- `runtime_recipe`: sealed original bounds, seed 7, and a full native request
  template. Generated module path/hash and compile-cache path come from the
  bounded current producer output; all modeled request fields match the
  template exactly. The actual dynamic module provenance, installation and
  retained preparation observations authenticate its source/compiler closure.
- `runtime_reference`: `{python,script,derived,files,build_receipt,binary,helper}`.
  The immutable original archive, derived source manifest, compiler, build
  command receipt, helper and binary are sealed static inputs. The wrapper runs
  both original model owners against the actual current native module/request;
  this receiver recomputes raw Result streams, metadata, nested command custody
  and the source archive/file inventory before comparing the native release.
- `qualified.workflow` and `qualified.activity`: each has sealed `recipe`,
  `release_input`, and `diagnostic_input` paths. Each recipe contains the exact
  `native_input` object, seed, original bounds, module path/hash, selection,
  original arguments and `preparation={provenance,installation,observations}`.
  The preparation inventory authenticates actual compiler commands/source
  closure; package and cwd labels alone cannot admit a module. The reader compares the actual retained request to it.

The receiver constructs named native `runtime-cost-profile` and `terminal-exit`
argv itself. Ordinary qualified execution uses `gomad-measure --linked-run`;
diagnostic execution uses `gomad-profile INPUT OUTPUT`. It creates fresh output
under `ROOT/final-profiles`; these outputs are never required in `config.files`.
Process IDs, argv, timings and exit status come from `driver.command` journals.
Raw stream digests, request equality, module hashes, role/source bindings and
original preservation predicates are recomputed independently.

`final-profiles-recipes.json` records the original Go source revision/hash and
assertions. It is a recipe manifest, not qualification evidence.

## Native applicability gaps

The current native runtime producer repeats the ordinary linked owner twice.
The original `measureRuntimeCostProfile` compares remote-model and helper-model
ownership. The `runtime_reference` invocation supplies both original model-owner runs on
the actual current module. Without that binding the receiver raises
`Unexecuted` instead of accepting ordinary repetition or the `accepted` flag.
The wrapper and native preparation custody are source integrations; their actual
frozen-candidate execution remains pending.

The current native terminal producer records six successful workflow runs. The
original additionally executes the post-load authorization identity mismatch,
checks its exact invalid result, zero instantiation/execution and one-second
terminal bound. That observation is missing; the receiver refuses the current
native terminal shape. A source-backed linked applicability implementation is
needed before either endpoint can pass. The supplemental df0 timing/admission
controls do not execute this original case.

The original result metadata is available natively: `Outcome.local_model` holds
`calls`, `advances`, and `terminal_base64`; `Outcome` holds `fuel_remaining`.
The sealed native request model binds environment/configuration. All are compared
exactly alongside stdout, stderr, WASI, choice, network and outcome. A diagnostic
comparator that removes fuel is insufficient for this receiver.

No actual final gate pass or fn-185.2/.3 performance measurement is claimed.
