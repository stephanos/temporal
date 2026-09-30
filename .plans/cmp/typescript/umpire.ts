// umpire.ts — the framework surface the Models author against.
//
// The mechanism, in one sentence: every declaration is an object literal handed to a builder whose
// generic parameters are `const` so they capture the literal types of what was written, and later
// declarations are typed in terms of earlier ones, so `tsc` alone checks that a `steps:` key is a
// declared action, a Scenario's actions belong to its Model and a `starts:` is one of the Model's
// phases. What needs a runtime — the finite table, the bounded search, the refinement walk — runs
// under vitest; those bodies are sketched here with `// elided:` comments describing the algorithm.

// ----------------------------------------------------------------------------------------------
// Finite domains
// ----------------------------------------------------------------------------------------------

/**
 * A finite domain: every member in catalog order, and a canonical key for each. TypeScript has no
 * runtime sum types, so the member list is the source of truth and the *type* is read off it with
 * `Member<typeof X>`; a value the list does not contain cannot be typed as an `X`.
 *
 * `key` is a method rather than a function-typed property so that `Finite<"a" | "b">` is assignable
 * to `Finite<unknown>` (methods are bivariant in their parameters; properties are not under
 * `strictFunctionTypes`).
 */
export interface Finite<T> {
  readonly members: readonly T[];
  key(value: T): string;
}

export type Member<D> = D extends Finite<infer T> ? T : never;

/** A plain enum: `enumOf(["unset", "expires"])` is `Finite<"unset" | "expires">`. */
export function enumOf<const T extends readonly string[]>(members: T): Finite<T[number]> {
  return { members, key: (value) => value };
}

type Kinds<T> = T extends { readonly kind: infer K extends string } ? K : never;
type MissingKinds<T, M extends readonly T[]> = Exclude<Kinds<T>, Kinds<M[number]>>;

/**
 * A domain whose constructors carry fields: a discriminated union on `kind`, listed one class per
 * assignment of the fields (`handlerError` is one constructor and two classes). The compiler checks
 * that every listed member is a `T` and that every `kind` of `T` appears at least once. It cannot
 * check that both `retryable` assignments appear — that is a runtime pin (`Reply.members.length`).
 *
 * Curried because TypeScript has no partial type-argument inference: `T` is written, `M` inferred.
 */
export function domain<T extends { readonly kind: string }>(): <const M extends readonly T[]>(
  members: M &
    ([MissingKinds<T, M>] extends [never] ? unknown : { readonly "missing kind": MissingKinds<T, M> }),
) => Finite<T> {
  // The key is the constructor name followed by its fields in *declaration order* of the literal,
  // which JavaScript preserves for string keys. Write `kind` first, always.
  return (members) => ({ members, key: (value) => Object.values(value).map(String).join("-") });
}

/**
 * `Fin<3>` is `0 | 1 | 2` and `UpTo<2>` is `0 | 1 | 2`. Type-level arithmetic by tuple length: it
 * works, it reads badly, and it hits the instantiation-depth limit near a thousand.
 */
export type Fin<N extends number, Acc extends readonly number[] = []> = Acc["length"] extends N
  ? Acc[number]
  : Fin<N, [...Acc, Acc["length"]]>;
export type UpTo<N extends number> = Fin<N> | N;

export function upTo<const N extends number>(bound: N): Finite<UpTo<N>> {
  const members = Array.from({ length: bound + 1 }, (_, i) => i as UpTo<N>);
  return { members, key: String };
}

/**
 * `n + 1`, capped at `bound`. Arithmetic widens to `number`, so the cast is unavoidable; the
 * enumeration is what keeps it honest.
 */
export function saturatingSucc<N extends number>(n: UpTo<N>, bound: N): UpTo<N> {
  return Math.min(n + 1, bound) as UpTo<N>;
}

/**
 * The product of finite fields, enumerated in field order: the state structures. The state type
 * is read back with `Member<typeof State>`, so it is finite by construction and there is one source
 * of truth for the fields.
 */
export function struct<const Fields extends Record<string, Finite<unknown>>>(
  fields: Fields,
): Finite<{ readonly [K in keyof Fields]: Member<Fields[K]> }> {
  type S = { readonly [K in keyof Fields]: Member<Fields[K]> };
  const names = Object.keys(fields) as (keyof Fields & string)[];
  const members = names.reduce<Record<string, unknown>[]>(
    (acc, name) => acc.flatMap((partial) => fields[name].members.map((v) => ({ ...partial, [name]: v }))),
    [{}],
  ) as S[];
  return { members, key: (s) => names.map((n) => fields[n].key(s[n])).join("-") };
}

/** Structural equality through the canonical key. `===` on two objects is identity in JavaScript. */
export function same<T>(domain: Finite<T>, a: T, b: T): boolean {
  return domain.key(a) === domain.key(b);
}

/**
 * Exhaustiveness. In the `default` of a `switch` over a discriminated union the value has type
 * `never` only when every case is handled; a missing case is `tsc` error TS2345 at this call.
 */
export function assertNever(value: never): never {
  throw new Error(`unreachable: ${JSON.stringify(value)}`);
}

// ----------------------------------------------------------------------------------------------
// Entities, parties, actions, observations
// ----------------------------------------------------------------------------------------------

export interface Entity<Name extends string = string> {
  readonly name: Name;
  /** Which recorded field names an instance. */
  readonly key?: string;
  readonly refer?: Readonly<Record<string, Entity>>;
}

export function entity<const Name extends string>(
  name: Name,
  decl: { readonly key?: string; readonly refer?: Readonly<Record<string, Entity>> } = {},
): Entity<Name> {
  return { name, ...decl };
}

/** Who performs an action. `system` is reserved for timers, declared under `timers:` on a machine. */
export type Party = "caller" | "handler" | "worker" | "network" | "operator";
export type SystemParty = "system";

export type InputSpec = Readonly<Record<string, Finite<unknown>>>;

/** The concrete input of an action: one member per field, or `void` for an action with no input. */
export type Inputs<I extends InputSpec | undefined> = I extends InputSpec
  ? { readonly [K in keyof I]: Member<I[K]> }
  : void;

/** One class of an action: its name and one assignment of its inputs. */
export interface Classed<Name extends string, In> {
  readonly action: Name;
  readonly input: In;
}

export interface Action<Name extends string = string, I extends InputSpec | undefined = InputSpec | undefined> {
  readonly name: Name;
  readonly party: Party;
  readonly creates?: Entity;
  readonly on?: Entity;
  /** Protobuf message name(s). With protobuf-es this could be a generated descriptor instead of a string. */
  readonly schema?: string;
  readonly input: I;
  /** The outcome enum, when the action has one. */
  readonly results?: Finite<string>;
  /** An input class mapped to the concrete realization value it stands for. */
  readonly examples?: readonly (readonly [Partial<Inputs<I>>, string])[];
  /** One class of this action, for a Scenario or a `when:`. An action with no input has no `of`: its name is its class. */
  readonly of: I extends InputSpec ? (input: Inputs<I>) => Classed<Name, Inputs<I>> : undefined;
}

export type AnyAction = Action<string, InputSpec | undefined>;

type ActionDecl<Name extends string, I extends InputSpec | undefined> = {
  readonly name: Name;
  readonly party: Party;
  readonly creates?: Entity;
  readonly on?: Entity;
  readonly schema?: string;
  readonly input?: I;
  readonly results?: Finite<string>;
  readonly examples?: readonly (readonly [Partial<Inputs<I>>, string])[];
};

export function action<const Name extends string, const I extends InputSpec>(
  decl: ActionDecl<Name, I> & { readonly input: I },
): Action<Name, I>;
export function action<const Name extends string>(
  decl: ActionDecl<Name, undefined> & { readonly input?: undefined },
): Action<Name, undefined>;
export function action(decl: ActionDecl<string, InputSpec | undefined>): AnyAction {
  const of = decl.input ? (input: unknown) => ({ action: decl.name, input }) : undefined;
  return { ...decl, input: decl.input, of } as AnyAction;
}

/** A derived read used as evidence with no history event. */
export interface Observation<Name extends string = string> {
  readonly name: Name;
  readonly on: Entity;
  readonly read: string;
}

export function observation<const Name extends string>(
  name: Name,
  decl: { readonly on: Entity; readonly read: string },
): Observation<Name> {
  return { name, ...decl };
}

// ----------------------------------------------------------------------------------------------
// Steps and machines
// ----------------------------------------------------------------------------------------------

/** `Step S O F`: what one action does from one state. A step function returns `[]` when the action is not enabled. */
export interface Step<S, O, F> {
  readonly outcome: O;
  readonly state: S;
  readonly facts: readonly F[];
}

export type StepFn<S, O, F, In> = (state: S, input: In) => readonly Step<S, O, F>[];

/**
 * The name of a fact without its payload: `nexusOperationTimedOut:scheduleToStart` is evidence
 * `nexusOperationTimedOut`. A fact with a field stays a string so that `facts.includes(...)`, deep
 * equality and JSON all work without a helper; the template literal keeps the payload typed.
 */
export type FactName<F extends string> = F extends `${infer Head}:${string}` ? Head : F;

/**
 * The `steps:` block: exactly one step function per declared action (typed by that action's input)
 * and per declared timer (no input). An extra key is TS2353, a missing one is TS2741.
 */
export type StepsOf<S, O, F, A extends readonly AnyAction[], T extends readonly string[]> = {
  readonly [X in A[number] as X["name"]]: StepFn<S, O, F, Inputs<X["input"]>>;
} & { readonly [K in T[number]]: StepFn<S, O, F, void> };

export interface Refinement<S, S2> {
  readonly machine: MachineOn<S2>;
  /** The abstraction function; a method signature so `Refinement<ProtocolState, ProductState>` is assignable to `Refinement<any, any>`. Checked at test time. */
  map(state: S): S2;
}

export interface Row<S, O, F> {
  readonly from: S;
  /** The action class key, `handlerReply-handlerError-true` style. */
  readonly action: string;
  readonly step: Step<S, O, F>;
}

export interface Table<S, O, F> {
  readonly states: readonly S[];
  readonly rows: readonly Row<S, O, F>[];
}

export interface Machine<
  Name extends string,
  S,
  O extends string,
  F extends string,
  A extends readonly AnyAction[],
  T extends readonly string[],
  P extends string,
> {
  readonly name: Name;
  readonly for: readonly Entity[];
  readonly state: Finite<S>;
  readonly starts: readonly P[];
  readonly ends: readonly P[];
  readonly actions: A;
  /** System actions the machine owns. */
  readonly timers: T;
  /** Timers that record nothing. */
  readonly unobservable: readonly T[number][];
  /** Fact name -> recorded event or observation name. */
  readonly evidence: Partial<Record<FactName<F>, string>>;
  readonly steps: StepsOf<S, O, F, A, T>;
  readonly refines?: Refinement<S, unknown>;
  /** Whether a state is at a phase. A composed state is at one phase per member. */
  at(state: S, phase: P): boolean;
  /** Derived once, on first read. The table is what Search, the refinement check and the fingerprint walk. */
  readonly table: Table<S, O, F>;
  /** Every action class in catalog order. */
  readonly actionKeys: readonly string[];
  readonly reachable: readonly S[];
  readonly endStates: readonly S[];
}

export type AnyMachine = Machine<string, any, any, any, any, any, any>;
export type MachineOn<S> = Machine<string, S, any, any, any, any, any>;

export type StateOf<M> = M extends Machine<any, infer S, any, any, any, any, any> ? S : never;
export type StepOf<M> = M extends Machine<any, infer S, infer O, infer F, any, any, any> ? Step<S, O, F> : never;
export type PhaseOf<M> = M extends Machine<any, any, any, any, any, any, infer P> ? P : never;
export type ActionsOf<M> = M extends Machine<any, any, any, any, infer A, any, any> ? A : never;
export type TimersOf<M> = M extends Machine<any, any, any, any, any, infer T, any> ? T[number] : never;
export type ActionName<M> = ActionsOf<M>[number]["name"] | TimersOf<M>;
export type RefinedBy<M> = M extends { readonly refines: Refinement<any, infer S2> } ? MachineOn<S2> : never;

/**
 * An action as a Scenario lists it: a `Classed` for an action with input, the bare name for an
 * action without one or for a timer.
 */
export type ClassedAction<M> =
  | {
      [X in ActionsOf<M>[number] as X["name"]]: X["input"] extends InputSpec
        ? Classed<X["name"], Inputs<X["input"]>>
        : X["name"];
    }[ActionsOf<M>[number]["name"]]
  | TimersOf<M>;

/** What a same-step claim is about: one class, or an action of any class by its name alone. */
export type When<M> = ClassedAction<M> | ActionsOf<M>[number]["name"];

type MachineDecl<
  Name extends string,
  S extends { readonly phase: string },
  O extends string,
  F extends string,
  A extends readonly AnyAction[],
  T extends readonly string[],
  S2,
> = {
  readonly name: Name;
  readonly for: Entity;
  readonly state: Finite<S>;
  // `NoInfer`: `S` comes from `state:` and `F` from the step functions' return types; without it the
  // compiler would also try to infer them from these positions and report the mismatch in the wrong place.
  readonly starts: readonly NoInfer<S>["phase"][];
  readonly ends: readonly NoInfer<S>["phase"][];
  readonly actions: A;
  readonly timers?: T;
  readonly unobservable?: readonly T[number][];
  readonly evidence: Partial<Record<FactName<NoInfer<F>>, string>>;
  readonly steps: StepsOf<S, O, F, A, T>;
  readonly refines?: { readonly machine: MachineOn<S2>; readonly map: (state: NoInfer<S>) => S2 };
};

/**
 * The `machine` command. The phase is the `phase` field of the state, so `starts:` and `ends:` are
 * typed by it; `S`, `O` and `F` are inferred from `state:` and the step functions.
 */
export function defineMachine<
  const Name extends string,
  S extends { readonly phase: string },
  O extends string,
  F extends string,
  const A extends readonly AnyAction[],
  const T extends readonly string[] = readonly [],
  S2 = never,
>(
  decl: MachineDecl<Name, S, O, F, A, T, S2>,
): Machine<Name, S, O, F, A, T, S["phase"]> & { readonly refines: Refinement<S, S2> } {
  const base = {
    ...decl,
    for: [decl.for],
    timers: (decl.timers ?? []) as T,
    unobservable: decl.unobservable ?? [],
    at: (state: S, phase: S["phase"]) => state.phase === phase,
  };
  return withTables(base as never) as never;
}

/** The classes of one action: the cartesian product of its input fields, keyed `name-field1-field2`. */
export function classesOf(action: AnyAction): readonly { readonly key: string; readonly input: unknown }[] {
  if (!action.input) return [{ key: action.name, input: undefined }];
  const fields = action.input;
  const names = Object.keys(fields);
  return names
    .reduce<Record<string, unknown>[]>(
      (acc, n) => acc.flatMap((partial) => fields[n].members.map((v) => ({ ...partial, [n]: v }))),
      [{}],
    )
    .map((input) => ({ key: [action.name, ...names.map((n) => fields[n].key(input[n]))].join("-"), input }));
}

function withTables<M extends AnyMachine>(machine: Omit<M, "table" | "actionKeys" | "reachable" | "endStates">): M {
  // elided: define `table`, `actionKeys`, `reachable` and `endStates` as memoized getters.
  //   actionKeys = [...machine.actions.flatMap(classesOf).map(c => c.key), ...machine.timers].sort()
  //   table.rows  = for each state in machine.state.members, for each class: steps[name](state, input)
  //                 -> one Row per returned Step, keyed by state.key + "-" + class.key
  //   reachable   = BFS from the start states over table.rows
  //   endStates   = machine.state.members.filter(s => machine.ends.some(p => machine.at(s, p)))
  return machine as M;
}

/** `machine X from: Y restrict: [...]`: the same machine over a subset of its actions. */
export function restrict<M extends AnyMachine, const N extends readonly ActionName<M>[]>(
  machine: M,
  names: N,
): Machine<
  M["name"],
  StateOf<M>,
  StepOf<M>["outcome"],
  StepOf<M>["facts"][number],
  readonly Extract<ActionsOf<M>[number], { readonly name: N[number] }>[],
  readonly Extract<TimersOf<M>, N[number]>[],
  PhaseOf<M>
> {
  // elided: filter `actions`, `timers` and `steps` by `names`, then `withTables`.
  void names;
  return machine as never;
}

// ----------------------------------------------------------------------------------------------
// Properties, scenarios, limits, queries
// ----------------------------------------------------------------------------------------------

export interface Property<M extends AnyMachine, Name extends string = string> {
  readonly name: Name;
  readonly machine: M;
  readonly when?: When<M>;
  readonly holds: ((step: StepOf<M>) => boolean) | ((before: StepOf<M>, after: StepOf<M>) => boolean);
}

/** A same-step claim: `when:` names the action class, `holds` sees the step it produces. */
export function property<const Name extends string, M extends AnyMachine>(decl: {
  readonly name: Name;
  readonly machine: M;
  readonly when: When<M>;
  readonly holds: (step: StepOf<M>) => boolean;
}): Property<M, Name>;
/** A transition claim: `holds` sees the step before and the step after. Searched and verified, never realized. */
export function property<const Name extends string, M extends AnyMachine>(decl: {
  readonly name: Name;
  readonly machine: M;
  readonly holds: (before: StepOf<M>, after: StepOf<M>) => boolean;
}): Property<M, Name>;
export function property(decl: Property<AnyMachine>): Property<AnyMachine> {
  return decl;
}

export interface Scenario<M extends AnyMachine, Name extends string = string> {
  readonly name: Name;
  readonly model: M;
  readonly starts: PhaseOf<M>;
  readonly actions: readonly ClassedAction<M>[];
}

/** A path: its start by phase and its classed actions in order, both typed by the Model. */
export function scenario<const Name extends string, M extends AnyMachine>(decl: Scenario<M, Name>): Scenario<M, Name> {
  return decl;
}

export interface Limits<Name extends string = string> {
  readonly name: Name;
  readonly steps: number;
  readonly actions: number;
  readonly search: number;
}

export function limits<const Name extends string>(decl: Limits<Name>): Limits<Name> {
  return decl;
}

export interface Query<M extends AnyMachine, Name extends string = string> {
  readonly name: Name;
  readonly mode: "find" | "verify";
  readonly property: Property<M> | Property<RefinedBy<M>>;
  readonly in: Scenario<M>;
  readonly limits: Limits;
}

type QueryDecl<M extends AnyMachine, Name extends string> = {
  readonly name: Name;
  readonly in: Scenario<M>;
  readonly limits: Limits;
} & (
  | { readonly find: Property<M> | Property<RefinedBy<M>>; readonly verify?: undefined }
  | { readonly verify: Property<M> | Property<RefinedBy<M>>; readonly find?: undefined }
);

/**
 * `find:` a same-step claim on a path (realized by a set) or `verify:` a claim over every trace of
 * a path (never realized). The Property may be the Model's own or its refined machine's, read
 * through the map. The search runs at test time via `run`.
 */
export function query<const Name extends string, M extends AnyMachine>(decl: QueryDecl<M, Name>): Query<M, Name> {
  const { name, in: scenario, limits } = decl;
  return decl.find
    ? { name, mode: "find", property: decl.find, in: scenario, limits }
    : { name, mode: "verify", property: decl.verify, in: scenario, limits };
}

export type QueryOutcome = "found" | "notFound" | "verified" | "violated" | "exhausted";

export interface QueryResult<M extends AnyMachine> {
  readonly outcome: QueryOutcome;
  /** The witness trace for `found`, the counterexample for `violated`. */
  readonly trace?: readonly Row<StateOf<M>, StepOf<M>["outcome"], StepOf<M>["facts"][number]>[];
}

/** Bounded breadth-first search over the table, cut at `limits.search` candidates. */
export function run<M extends AnyMachine>(query: Query<M>): QueryResult<M> {
  // elided: BFS from the states `at` scenario.starts; a candidate is a row sequence whose action
  // classes are the scenario's in order (steps <= limits.steps, distinct actions <= limits.actions);
  // `find` stops at the first candidate whose triggering step satisfies `holds`; `verify` walks
  // every candidate and every adjacent row pair; over budget is `exhausted`.
  void query;
  return { outcome: "exhausted" };
}

export interface RefinementResult {
  /** `null` when every protocol row maps to a product row or a product stutter; otherwise the offending row key. */
  readonly rejected: string | null;
  /** Protocol row key -> the product action class whose row it maps to, or `null` for a stutter. */
  readonly rows: ReadonlyMap<string, string | null>;
}

/**
 * The refinement rule, by mapped states: a protocol row `(s, a, s')` is fine when `map s == map s'`
 * (a stutter) or the product has some row from `map s` to `map s'` under *any* action class. The
 * match is not by action name: a protocol timer row maps to the product's `timeout` row.
 */
export function checkRefinement(machine: AnyMachine & { readonly refines: Refinement<any, any> }): RefinementResult {
  // elided: for each row (from, action, step) of machine.table: a = map(from), b = map(step.state);
  //   a == b -> stutter; else find any product row a -> b -> its class; none -> rejected.
  void machine;
  return { rejected: null, rows: new Map() };
}

// ----------------------------------------------------------------------------------------------
// Sets
// ----------------------------------------------------------------------------------------------

export type Binding = "driven" | "observed";
export type Cover = "rows" | "results" | "classMembers";

type SetBase<Name extends string> = {
  readonly name: Name;
  readonly bind: Partial<Readonly<Record<Party, Binding>>>;
};

export type SetDecl<Name extends string = string> =
  | (SetBase<Name> & {
      readonly purpose: "functional";
      /** Run once per value of the implementation switch (HSM / CHASM). */
      readonly repeat?: "implementation";
      readonly queries: readonly Query<AnyMachine>[];
    })
  | (SetBase<Name> & { readonly purpose: "canary"; readonly queries: readonly Query<AnyMachine>[] })
  | (SetBase<Name> & {
      readonly purpose: "exploratory";
      readonly machine: AnyMachine;
      readonly cover: readonly Cover[];
      readonly budget: Limits;
    });

export function set<const Name extends string>(decl: SetDecl<Name>): SetDecl<Name> {
  return decl;
}

// ----------------------------------------------------------------------------------------------
// Composition
// ----------------------------------------------------------------------------------------------

type Members = Readonly<Record<string, AnyMachine>>;

/** `operation.unscheduled`, `worker.polling`: a member's phase, qualified by the member name. */
export type MemberPhase<Ms extends Members> = {
  [K in keyof Ms & string]: `${K}.${PhaseOf<Ms[K]>}`;
}[keyof Ms & string];

/** `operation.handlerReply`, `worker.serve`: a member's action or timer, qualified. */
export type MemberAction<Ms extends Members> = {
  [K in keyof Ms & string]: `${K}.${ActionName<Ms[K]>}`;
}[keyof Ms & string];

export type ComposedState<Ms extends Members> = { readonly [K in keyof Ms]: StateOf<Ms[K]> };

/** Parse `member.action` back into the member's Action declaration, to type a synced action's input by it. */
type Resolve<Ms extends Members, Q> = Q extends `${infer K extends keyof Ms & string}.${infer N}`
  ? Extract<ActionsOf<Ms[K]>[number], { readonly name: N }>
  : never;

type SyncSpec<Ms extends Members> = Readonly<Record<string, readonly [MemberAction<Ms>, MemberAction<Ms>]>>;

/** Each `sync:` line is an action of the composition, named by the line and taking the first member action's input. */
type SyncActions<Ms extends Members, Sync extends SyncSpec<Ms>> = {
  [N in keyof Sync & string]: Action<N, Resolve<Ms, Sync[N][0]>["input"]>;
}[keyof Sync & string];

/** Every member action no `sync:` line names stays executable on its own, under its qualified name. */
type QualifiedActions<Ms extends Members, Sync extends SyncSpec<Ms>> = {
  [K in keyof Ms & string]: {
    [X in ActionsOf<Ms[K]>[number] as X["name"]]: `${K}.${X["name"]}` extends Sync[keyof Sync][number]
      ? never
      : Action<`${K}.${X["name"]}`, X["input"]>;
  }[ActionsOf<Ms[K]>[number]["name"]];
}[keyof Ms & string];

type QualifiedTimers<Ms extends Members> = {
  [K in keyof Ms & string]: `${K}.${TimersOf<Ms[K]>}`;
}[keyof Ms & string];

type MemberOutcome<Ms extends Members> = StepOf<Ms[keyof Ms]>["outcome"];
type MemberFact<Ms extends Members> = StepOf<Ms[keyof Ms]>["facts"][number];

export type Composed<Name extends string, Ms extends Members, Sync extends SyncSpec<Ms>> = Machine<
  Name,
  ComposedState<Ms>,
  MemberOutcome<Ms>,
  MemberFact<Ms>,
  readonly (SyncActions<Ms, Sync> | QualifiedActions<Ms, Sync>)[],
  readonly QualifiedTimers<Ms>[],
  MemberPhase<Ms>
>;

/**
 * `compose`: the product of machines of different entities. `sync:` pairs fire as one action; the
 * state is one field per member; starts, ends and the qualified action names are template-literal
 * types, so a typo in `"operation.unscheduled"` is a type error.
 */
export function compose<const Name extends string, const Ms extends Members, const Sync extends SyncSpec<Ms>>(decl: {
  readonly name: Name;
  readonly members: Ms;
  readonly sync: Sync;
  readonly starts: readonly MemberPhase<Ms>[];
  readonly ends: readonly MemberPhase<Ms>[];
}): Composed<Name, Ms, Sync> {
  // elided: state = struct over members' states; steps for a sync line run both member steps and
  // keep the pair only when both return a Step (a stopped worker has no `serve` row, so a synced
  // reply has none either); qualified steps run one member and keep the other's state; then withTables.
  return decl as never;
}

/** `member("operation", schedule.of({...}))`: a member's classed action, qualified for a composed Scenario. */
export function member<const K extends string, const N extends string, In>(
  name: K,
  classed: Classed<N, In>,
): Classed<`${K}.${N}`, In> {
  return { action: `${name}.${classed.action}`, input: classed.input };
}
