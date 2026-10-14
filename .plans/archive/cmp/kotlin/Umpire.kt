// The framework surface the Kotlin samples author against. Bodies are sketched or elided; the
// signatures are what a Model file binds to. See README.md for what runs when.
@file:Suppress("UNUSED_PARAMETER", "unused")

package umpire

import kotlin.reflect.KClass
import kotlin.reflect.KFunction
import kotlin.reflect.KProperty1
import kotlin.reflect.full.companionObjectInstance
import kotlin.reflect.full.primaryConstructor

// ---------------------------------------------------------------------------------------------
// Steps
// ---------------------------------------------------------------------------------------------

/**
 * One row a step function produces: the outcome the party observes, the state the machine moves to,
 * and the facts the step records. A step function `(S, inputs) -> List<Step<S, O, F>>` returns the
 * empty list when the action is not enabled.
 */
data class Step<S, O, F>(val outcome: O, val state: S, val facts: List<F>)

// ---------------------------------------------------------------------------------------------
// Finite enumeration
// ---------------------------------------------------------------------------------------------

/**
 * A type whose values can be listed. The table of a machine is built by enumerating `S` and every
 * action's inputs, so both must be `Finite`.
 *
 * Three ways in, all resolved by [Finite.of]:
 *  - an `enum class` lists its `entries`;
 *  - a `sealed interface` lists each `data object` once and each `data class` subtype once per
 *    assignment of its (finite) constructor parameters, which is what makes
 *    `HandlerError(retryable: Boolean)` one constructor and two classes;
 *  - a `data class` state is the cartesian product of its constructor parameters;
 *  - a type may opt in explicitly with `companion object : Finite<T>` (bounded ints do this).
 *
 * Reflection over `sealedSubclasses` needs `kotlin-reflect` on the classpath and runs once per type
 * at class-load time. A KSP processor could generate the same list as a `val entries` at build
 * time; the resolution order below would consult the generated companion first, so nothing in the
 * Model files changes if that is swapped in.
 */
interface Finite<T> {
    val values: List<T>

    companion object {
        private val cache = HashMap<KClass<*>, Finite<*>>()

        @Suppress("UNCHECKED_CAST")
        fun <T : Any> of(type: KClass<T>): Finite<T> = cache.getOrPut(type) {
            when {
                type == Boolean::class -> listOf(false, true).asFinite()
                type.companionObjectInstance is Finite<*> -> type.companionObjectInstance as Finite<*>
                type.java.isEnum -> type.java.enumConstants.toList().asFinite()
                type.isSealed -> type.sealedSubclasses.flatMap { of(it).values }.asFinite()
                type.objectInstance != null -> listOf(type.objectInstance!!).asFinite()
                type.isData -> product(type.primaryConstructor!!).asFinite()
                else -> throw IllegalArgumentException(
                    "${type.simpleName} is not finite: it is neither an enum, a sealed type, a data " +
                        "class of finite fields, nor does its companion implement Finite",
                )
            }
        } as Finite<T>

        inline fun <reified T : Any> of(): Finite<T> = of(T::class)

        /** Every assignment of a constructor's parameters, each of which must itself be finite. */
        private fun <T : Any> product(constructor: KFunction<T>): List<T> {
            val domains = constructor.parameters.map { of(it.type.classifier as KClass<*>).values }
            return cartesian(domains).map { args -> constructor.call(*args.toTypedArray()) }
        }

        private fun cartesian(domains: List<List<Any?>>): List<List<Any?>> =
            domains.fold(listOf(emptyList())) { acc, domain -> acc.flatMap { prefix -> domain.map { prefix + it } } }

        private fun <T> List<T>.asFinite(): Finite<T> = object : Finite<T> {
            override val values = this@asFinite
        }
    }
}

// ---------------------------------------------------------------------------------------------
// Vocabulary: entities, parties, actions, observations
// ---------------------------------------------------------------------------------------------

@DslMarker
annotation class UmpireDsl

class Entity internal constructor(val name: String, val key: String?, val refers: Map<String, Entity>)

@UmpireDsl
class EntityScope internal constructor() {
    var key: String? = null
    internal val refers = LinkedHashMap<String, Entity>()

    /** `refer("caller", workflow)`: this entity's instances point at one of the other's. */
    fun refer(role: String, target: Entity) {
        refers[role] = target
    }
}

fun entity(name: String, configure: EntityScope.() -> Unit = {}): Entity =
    EntityScope().apply(configure).let { Entity(name, it.key, it.refers) }

/** Who performs an action. `system` is reserved for the timers a machine owns. */
@JvmInline
value class Party(val name: String)

val caller = Party("caller")
val handler = Party("handler")
val worker = Party("worker")
val network = Party("network")
val operator = Party("operator")
val system = Party("system")

/** A derived read used as evidence where no history event exists. */
class Observation internal constructor(val name: String, val on: Entity, val read: String)

@UmpireDsl
class ObservationScope internal constructor() {
    lateinit var on: Entity
    lateinit var read: String
}

fun observation(name: String, configure: ObservationScope.() -> Unit): Observation =
    ObservationScope().apply(configure).let { Observation(name, it.on, it.read) }

/**
 * Something a Scenario can list and a Property can name under `when`: an action with all of its
 * inputs fixed (one action class), an action of any class, a timer, or a member's action inside a
 * composition. `Action0` and `Timer` are their own single class.
 */
sealed interface Trigger {
    val name: String
}

/**
 * The declared trigger a listed one belongs to: an action class is its action, a member's classed
 * action is the member's action. What a machine's `actions` are compared against.
 */
fun Trigger.declared(): Trigger = when (this) {
    is ActionClass -> action
    is MemberTrigger -> MemberTrigger(member, inner.declared())
    is MemberAction1<*> -> MemberTrigger(member, action)
    is MemberAction3<*, *, *> -> MemberTrigger(member, action)
    else -> this
}

/** An action with every input fixed; the unit a Scenario lists and an example is written at. */
class ActionClass internal constructor(val action: Action, val inputs: List<Any?>) : Trigger {
    override val name: String get() = (listOf(action.name) + inputs.map(::className)).joinToString("-")
}

/** A declared action, before its inputs are fixed. Subtypes fix the arity so a step binding is typed. */
sealed class Action(
    override val name: String,
    val party: Party,
    val creates: Entity?,
    val on: Entity?,
    val schema: String?,
    val inputNames: List<String>,
    val inputDomains: List<Finite<*>>,
    val results: Finite<*>?,
    val examples: Map<Any, String>,
) : Trigger {
    /** One class per assignment of the finite inputs; `handlerReply` has six, `schedule` eight. */
    val classes: List<ActionClass>
        get() = cartesian(inputDomains.map { it.values }).map { ActionClass(this, it) }
}

class Action0 internal constructor(name: String, s: ActionSpec) :
    Action(name, s.declaredParty, s.creates, s.on, s.schema, emptyList(), emptyList(), s.resultDomain, s.examples)

class Action1<A> internal constructor(name: String, s: ActionSpec) :
    Action(name, s.declaredParty, s.creates, s.on, s.schema, s.inputNames, s.inputDomains, s.resultDomain, s.examples) {
    operator fun invoke(a: A): ActionClass = ActionClass(this, listOf(a))
}

class Action3<A, B, C> internal constructor(name: String, s: ActionSpec) :
    Action(name, s.declaredParty, s.creates, s.on, s.schema, s.inputNames, s.inputDomains, s.resultDomain, s.examples) {
    operator fun invoke(a: A, b: B, c: C): ActionClass = ActionClass(this, listOf(a, b, c))
}

/** A `system` action a machine owns. Declared beside the machine, then listed under `timers(...)`. */
class Timer internal constructor(override val name: String) : Trigger

fun timer(name: String): Timer = Timer(name)

@UmpireDsl
class ActionSpec internal constructor(internal val name: String) {
    /** Nullable rather than `lateinit`: `lateinit` is not allowed on a value-class typed property. */
    var party: Party? = null
    internal val declaredParty: Party get() = checkNotNull(party) { "action $name: no party declared" }
    var creates: Entity? = null
    var on: Entity? = null
    var schema: String? = null
    internal var inputNames: List<String> = emptyList()
    internal var inputDomains: List<Finite<*>> = emptyList()
    internal var resultDomain: Finite<*>? = null
    internal val examples = LinkedHashMap<Any, String>()

    /** Field names of the inputs, in the order of the type arguments given to `action<...>`. */
    fun input(vararg names: String) {
        inputNames = names.toList()
    }

    /** The outcome enum the party observes, when the action has one. */
    inline fun <reified R : Any> results() = results(Finite.of<R>())
    fun results(domain: Finite<*>) {
        resultDomain = domain
    }

    fun examples(configure: ExamplesScope.() -> Unit) = ExamplesScope(examples).configure()
}

@UmpireDsl
class ExamplesScope internal constructor(private val sink: MutableMap<Any, String>) {
    /** `HandlerError(retryable = false) realizedAs "BadRequest"`: one class, one concrete value. */
    infix fun Any.realizedAs(value: String) {
        sink[this] = value
    }
}

private fun ActionSpec.validated(): ActionSpec = apply {
    declaredParty
    check(inputNames.size == inputDomains.size) {
        "action $name: ${inputDomains.size} input type(s) but ${inputNames.size} field name(s); call input(...) once per type argument"
    }
    examples.keys.forEach { example ->
        check(inputDomains.any { example in it.values }) { "action $name: example '$example' is not a class of any input" }
    }
}

fun action(name: String, configure: ActionSpec.() -> Unit): Action0 =
    Action0(name, ActionSpec(name).apply(configure).validated())

// The `reified` entry points only resolve the `Finite` domains and delegate: a public `inline`
// function may not reach `internal` constructors, so the builders below are ordinary functions.
inline fun <reified A : Any> action(name: String, noinline configure: ActionSpec.() -> Unit): Action1<A> =
    action1(name, Finite.of<A>(), configure)

inline fun <reified A : Any, reified B : Any, reified C : Any> action(
    name: String,
    noinline configure: ActionSpec.() -> Unit,
): Action3<A, B, C> = action3(name, Finite.of<A>(), Finite.of<B>(), Finite.of<C>(), configure)

fun <A : Any> action1(name: String, domain: Finite<A>, configure: ActionSpec.() -> Unit): Action1<A> =
    Action1(name, ActionSpec(name).apply { inputDomains = listOf(domain) }.apply(configure).validated())

fun <A : Any, B : Any, C : Any> action3(
    name: String,
    a: Finite<A>,
    b: Finite<B>,
    c: Finite<C>,
    configure: ActionSpec.() -> Unit,
): Action3<A, B, C> =
    Action3(name, ActionSpec(name).apply { inputDomains = listOf(a, b, c) }.apply(configure).validated())

// ---------------------------------------------------------------------------------------------
// Machines
// ---------------------------------------------------------------------------------------------

/** What a fact resolves to when the Case reads it back. */
sealed interface Evidence {
    /** A name resolved against the realization's catalog: a history event, or a status value. */
    data class Named(val name: String) : Evidence

    /** A derived read declared as an [Observation]. */
    data class Read(val observation: Observation) : Evidence
}

fun event(name: String): Evidence = Evidence.Named(name)
fun status(name: String): Evidence = Evidence.Named(name)
fun read(observation: Observation): Evidence = Evidence.Read(observation)

/** One row of the finite table: from a state, one action class produces one step. */
data class Transition<S, O, F>(val from: S, val action: Trigger, val step: Step<S, O, F>)

/**
 * The derived step mapping of a refinement. A protocol row (s, a, s') is a stutter (`null`) when
 * `map(s) == map(s')`, and otherwise must map onto some row `map(s) -> map(s')` of the target, of any
 * action class; the key of that row is the value. [rejected] names the first protocol row that does
 * neither.
 */
class Refinement<S, P> internal constructor(
    val target: Machine<P, *, *>,
    val map: (S) -> P,
    val rows: Map<String, String?>,
    val rejected: String?,
)

/** A machine's finite table: every state of `S` and every row a step function produces. */
class Table<S, O, F> internal constructor(val states: List<S>, val transitions: List<Transition<S, O, F>>)

open class Machine<S, O, F> internal constructor(
    val name: String,
    val entity: Entity?,
    val stateType: Finite<S>,
    val starts: List<S>,
    private val endPredicate: S.() -> Boolean,
    val timers: List<Timer>,
    val unobservable: List<Timer>,
    val evidence: (F) -> Evidence?,
    internal val steps: Map<Trigger, (S, List<Any?>) -> List<Step<S, O, F>>>,
    val refinement: Refinement<S, *>?,
) {
    /** Every action class the machine steps on, by key, in canonical order. Timers count once each. */
    val actionKeys: List<String> get() = steps.keys.flatMap { trigger ->
        when (trigger) {
            is Action -> trigger.classes.map { it.name }
            else -> listOf(trigger.name)
        }
    }.sorted()

    open val actions: kotlin.collections.Set<Trigger> get() = steps.keys // `Set` is shadowed in this package, see below

    val table: Table<S, O, F> by lazy { Table(stateType.values, computeTransitions()) }
    val transitions: List<Transition<S, O, F>> get() = table.transitions
    val ends: List<S> get() = stateType.values.filter(endPredicate)

    /** States a run from a start can reach; the Behavior Fingerprint reads this, not `states`. */
    fun reachable(): List<S> = TODO("breadth-first over transitions from starts")

    /** A non-end state no row leaves, or `null`. */
    val stuck: S? get() = stateType.values.firstOrNull { s -> !s.endPredicate() && transitions.none { it.from == s } }

    fun isEnd(state: S): Boolean = state.endPredicate()

    private fun computeTransitions(): List<Transition<S, O, F>> = TODO("every state x every action class x step function")
}

@UmpireDsl
class MachineScope<S : Any, O : Any, F : Any> internal constructor(
    internal val name: String,
    private val stateType: Finite<S>,
) {
    var entity: Entity? = null
    private var starts: List<S> = emptyList()
    private var ends: (S.() -> Boolean)? = null
    private var timers: List<Timer> = emptyList()
    private var unobservable: List<Timer> = emptyList()
    private var evidence: ((F) -> Evidence?)? = null
    private var refines: (() -> Refinement<S, *>)? = null
    internal val steps = LinkedHashMap<Trigger, (S, List<Any?>) -> List<Step<S, O, F>>>()

    fun starts(vararg states: S) {
        starts = states.toList()
    }

    /** A predicate on the state, written with the state as receiver: `ends { terminalPhase(phase) }`. */
    fun ends(predicate: S.() -> Boolean) {
        ends = predicate
    }

    fun timers(vararg owned: Timer) {
        timers = owned.toList()
    }

    fun unobservable(vararg silent: Timer) {
        unobservable = silent.toList()
    }

    /**
     * How each fact reads back. Written as an exhaustive `when` over the fact type, so a fact added
     * without deciding its evidence is a compile error, not a missing line. `null` says no step of
     * this machine records the fact.
     */
    fun evidence(resolve: (F) -> Evidence?) {
        evidence = resolve
    }

    /** `refines(nexusProduct) via ::productOf`. The check runs when the machine is built. */
    fun <P : Any> refines(target: Machine<P, *, *>): RefinesHalf<P> = RefinesHalf(target)

    inner class RefinesHalf<P : Any>(private val target: Machine<P, *, *>) {
        infix fun via(map: (S) -> P) {
            // Deferred: the rows are walked in build(), once every step is bound.
            refines = { deriveRefinement(target, map) }
        }
    }

    fun steps(bind: StepsScope<S, O, F>.() -> Unit) = StepsScope(this).bind()

    internal fun build(): Machine<S, O, F> {
        check(starts.isNotEmpty()) { "machine $name: no starts(...)" }
        val ends = checkNotNull(ends) { "machine $name: no ends { ... }" }
        timers.forEach { check(it in steps) { "machine $name: timer '${it.name}' has no step; bind it under steps { }" } }
        steps.keys.filterIsInstance<Timer>().forEach {
            check(it in timers) { "machine $name: step for timer '${it.name}' but timers(...) does not list it" }
        }
        unobservable.forEach { check(it in timers) { "machine $name: unobservable '${it.name}' is not one of the machine's timers" } }
        val refinement = refines?.invoke()
        refinement?.rejected?.let { row ->
            throw IllegalStateException("machine $name does not refine ${refinement.target.name}: row $row maps to no target row and is not a stutter")
        }
        return Machine(name, entity, stateType, starts, ends, timers, unobservable, evidence ?: { null }, steps, refinement)
    }

    /**
     * Walk every row (s, a, s') of this machine's table through the map. `map(s) == map(s')` is a
     * stutter (`null`); otherwise the row is the key of some target row `map(s) -> map(s')`, of any
     * action class; a row that is neither sets `rejected`. The rows are computed from `steps` and
     * `stateType` directly, before the `Machine` exists. Body elided.
     */
    private fun <P : Any> deriveRefinement(target: Machine<P, *, *>, map: (S) -> P): Refinement<S, P> =
        TODO("enumerate stateType x steps, map each row, look it up in target.transitions")
}

@UmpireDsl
class StepsScope<S : Any, O : Any, F : Any> internal constructor(private val machine: MachineScope<S, O, F>) {
    private fun register(trigger: Trigger, fn: (S, List<Any?>) -> List<Step<S, O, F>>) {
        check(trigger !in machine.steps) { "machine ${machine.name}: '${trigger.name}' is bound twice" }
        machine.steps[trigger] = fn
    }

    /** `handlerReply runs ::handlerReplyStep`; the function's parameter types must match the action's inputs. */
    infix fun Action0.runs(fn: (S) -> List<Step<S, O, F>>) = register(this) { s, _ -> fn(s) }

    @Suppress("UNCHECKED_CAST")
    infix fun <A> Action1<A>.runs(fn: (S, A) -> List<Step<S, O, F>>) =
        register(this) { s, i -> fn(s, i[0] as A) }

    @Suppress("UNCHECKED_CAST")
    infix fun <A, B, C> Action3<A, B, C>.runs(fn: (S, A, B, C) -> List<Step<S, O, F>>) =
        register(this) { s, i -> fn(s, i[0] as A, i[1] as B, i[2] as C) }

    infix fun Timer.runs(fn: (S) -> List<Step<S, O, F>>) = register(this) { s, _ -> fn(s) }
}

inline fun <reified S : Any, reified O : Any, reified F : Any> machine(
    name: String,
    noinline configure: MachineScope<S, O, F>.() -> Unit,
): Machine<S, O, F> = machineOf(name, Finite.of<S>(), configure)

fun <S : Any, O : Any, F : Any> machineOf(
    name: String,
    stateType: Finite<S>,
    configure: MachineScope<S, O, F>.() -> Unit,
): Machine<S, O, F> = MachineScope<S, O, F>(name, stateType).apply(configure).build()

/** `polling.restrict("handlerWorker", workerStop, serve)`: the same machine with only the named actions. */
fun <S, O, F> Machine<S, O, F>.restrict(name: String, vararg keep: Trigger): Machine<S, O, F> {
    keep.forEach { check(it in steps) { "machine $name: restrict names '${it.name}', which ${this.name} does not have" } }
    return Machine(name, entity, stateType, starts, { isEnd(this) }, timers.filter { it in keep }, unobservable, evidence, steps.filterKeys { it in keep }, refinement)
}

// ---------------------------------------------------------------------------------------------
// Properties
// ---------------------------------------------------------------------------------------------

sealed interface Claim<S, O, F> {
    /** `when: <trigger>` and a predicate on the step it produces. */
    class SameStep<S, O, F>(val trigger: Trigger, val holds: (Step<S, O, F>) -> Boolean) : Claim<S, O, F>

    /** No `when`: a predicate on consecutive steps. Searched and verified, never realized. */
    class Transition<S, O, F>(val holds: (before: Step<S, O, F>, after: Step<S, O, F>) -> Boolean) : Claim<S, O, F>
}

class Property<S, O, F> internal constructor(val name: String, val machine: Machine<S, O, F>, val claim: Claim<S, O, F>)

@UmpireDsl
class PropertyScope<S, O, F> internal constructor(private val name: String, private val machine: Machine<S, O, F>) {
    internal var claim: Claim<S, O, F>? = null

    private fun set(c: Claim<S, O, F>) {
        check(claim == null) { "property $name: two claims; a Property makes one" }
        claim = c
    }

    /** `handlerReply(SyncSuccess) holds { step -> ... }`: the same-step claim. */
    infix fun Trigger.holds(predicate: (Step<S, O, F>) -> Boolean) {
        val action = declared()
        check(machine.actions.contains(action)) {
            "property $name: ${machine.name} has no action '${action.name}'"
        }
        set(Claim.SameStep(this, predicate))
    }

    /** `holds { before, after -> ... }`: the transition claim. Overloaded by lambda arity. */
    fun holds(predicate: (before: Step<S, O, F>, after: Step<S, O, F>) -> Boolean) = set(Claim.Transition(predicate))
}

fun <S, O, F> property(name: String, machine: Machine<S, O, F>, configure: PropertyScope<S, O, F>.() -> Unit): Property<S, O, F> {
    val scope = PropertyScope(name, machine).apply(configure)
    return Property(name, machine, checkNotNull(scope.claim) { "property $name: no holds { ... }" })
}

// ---------------------------------------------------------------------------------------------
// Scenarios, Limits, Queries
// ---------------------------------------------------------------------------------------------

class Scenario<S, O, F> internal constructor(val name: String, val model: Machine<S, O, F>, val starts: S, val actions: List<Trigger>)

@UmpireDsl
class ScenarioScope<S, O, F> internal constructor(private val name: String, private val model: Machine<S, O, F>) {
    internal var starts: S? = null
    internal var actions: List<Trigger> = emptyList()

    fun starts(state: S) {
        starts = state
    }

    fun actions(vararg path: Trigger) {
        path.forEach { t ->
            val action = t.declared()
            check(model.actions.contains(action)) { "scenario $name: ${model.name} has no action '${action.name}'" }
        }
        actions = path.toList()
    }
}

fun <S, O, F> scenario(name: String, model: Machine<S, O, F>, configure: ScenarioScope<S, O, F>.() -> Unit): Scenario<S, O, F> {
    val scope = ScenarioScope(name, model).apply(configure)
    return Scenario(name, model, checkNotNull(scope.starts) { "scenario $name: no starts(...)" }, scope.actions)
}

data class Limits(val name: String, val steps: Int, val actions: Int, val search: Int)

fun limits(name: String, steps: Int, actions: Int, search: Int) = Limits(name, steps, actions, search)

/** What Search answers. `Found` and `NotFound` answer a `find`; the other two a `verify`. */
sealed interface Outcome {
    data class Found(val witness: List<Trigger>) : Outcome
    data object NotFound : Outcome
    data object VerifiedWithinLimits : Outcome
    data class Violated(val counterexample: List<Trigger>) : Outcome
}

enum class Mode { Find, Verify }

class Query internal constructor(
    val name: String,
    val mode: Mode,
    val property: Property<*, *, *>,
    val scenario: Scenario<*, *, *>,
    val limits: Limits,
) {
    /** Search over the scenario's automaton within the limits. In this sample it runs from the pins. */
    fun run(): Outcome = TODO("bounded search; a find returns the first witness, a verify the first counterexample")
}

@UmpireDsl
class QueryScope internal constructor(private val name: String) {
    internal var query: Query? = null

    inner class Half(private val mode: Mode, private val property: Property<*, *, *>) {
        /** `find(syncSucceeds) on syncReplied within two`. */
        infix fun on(scenario: Scenario<*, *, *>): Bounded {
            val m = scenario.model
            // A Property on the refined machine is read on the refining one through the map.
            check(property.machine == m || m.refinement?.target == property.machine) {
                "query $name: property ${property.name} is about ${property.machine.name}, which scenario " +
                    "${scenario.name}'s model ${m.name} neither is nor refines"
            }
            if (property.claim is Claim.SameStep<*, *, *>) {
                val action = (property.claim as Claim.SameStep<*, *, *>).trigger.declared()
                check(m.actions.contains(action)) {
                    "query $name: the Property names the action '${action.name}' of '${property.machine.name}', and " +
                        "'${m.name}' has no action of that name; a Property on the refined machine is read on the " +
                        "refining one through the values of the same name, and a state through its map"
                }
            }
            return Bounded(mode, property, scenario)
        }
    }

    inner class Bounded(private val mode: Mode, private val property: Property<*, *, *>, private val scenario: Scenario<*, *, *>) {
        infix fun within(limits: Limits) {
            check(query == null) { "query $name: declared twice" }
            query = Query(name, mode, property, scenario, limits)
        }
    }

    fun find(property: Property<*, *, *>): Half {
        check(property.claim is Claim.SameStep<*, *, *>) { "query $name: a transition claim is verified, not found" }
        return Half(Mode.Find, property)
    }

    fun verify(property: Property<*, *, *>): Half = Half(Mode.Verify, property)
}

fun query(name: String, configure: QueryScope.() -> Unit): Query =
    QueryScope(name).apply(configure).let { checkNotNull(it.query) { "query $name: no find(...) or verify(...)" } }

// ---------------------------------------------------------------------------------------------
// Sets
// ---------------------------------------------------------------------------------------------

enum class Purpose { Functional, Canary, Exploratory }
enum class Binding { Driven, Observed }
enum class Repeat { Implementation }
enum class Cover { Rows, Results, ClassMembers }

val driven = Binding.Driven
val observed = Binding.Observed

/**
 * A group of Queries bound to parties, or an exploration of one machine. Named `Set` after the
 * Lean keyword; inside package `umpire` the collection type is therefore spelled
 * `kotlin.collections.Set`, and a Model file that star-imports `umpire.*` sees the same shadow.
 */
class Set internal constructor(
    val name: String,
    val purpose: Purpose,
    val bindings: Map<Party, Binding>,
    val repeat: Repeat?,
    val queries: List<Query>,
    val machine: Machine<*, *, *>?,
    val cover: List<Cover>,
    val budget: Limits?,
)

@UmpireDsl
class SetScope internal constructor(private val name: String) {
    lateinit var purpose: Purpose
    var repeat: Repeat? = null
    var machine: Machine<*, *, *>? = null
    var budget: Limits? = null
    private var bindings: Map<Party, Binding> = emptyMap()
    private var queries: List<Query> = emptyList()
    private var cover: List<Cover> = emptyList()

    /** `bind(caller to driven, handler to observed, ...)`. */
    fun bind(vararg pairs: Pair<Party, Binding>) {
        bindings = pairs.toMap(LinkedHashMap())
    }

    fun queries(vararg named: Query) {
        queries = named.toList()
    }

    fun cover(vararg targets: Cover) {
        cover = targets.toList()
    }

    internal fun build(): Set {
        check(::purpose.isInitialized) { "set $name: no purpose" }
        check(system !in bindings) { "set $name: 'system' is the server's party and is never bound" }
        when (purpose) {
            Purpose.Exploratory -> {
                checkNotNull(machine) { "set $name: an exploratory set names a machine" }
                checkNotNull(budget) { "set $name: an exploratory set names a budget" }
                check(queries.isEmpty()) { "set $name: an exploratory set covers a machine, it lists no queries" }
            }
            else -> {
                check(queries.isNotEmpty()) { "set $name: no queries" }
                queries.forEach { q -> check(q.mode == Mode.Find) { "set $name: query ${q.name} verifies; a verify Query realizes nothing" } }
                // A canary's silent-step rejection (a path with an unobservable timer or a step that
                // records nothing) is the realization's check and lands where Cases are produced.
            }
        }
        return Set(name, purpose, bindings, repeat, queries, machine, cover, budget)
    }
}

fun set(name: String, configure: SetScope.() -> Unit): Set = SetScope(name).apply(configure).build()

// ---------------------------------------------------------------------------------------------
// Composition
// ---------------------------------------------------------------------------------------------

/** A member's trigger inside a composition: `operation[schedule]` or `worker[serve]`. */
data class MemberTrigger internal constructor(val member: Member<*, *, *, *>, val inner: Trigger) : Trigger {
    override val name: String get() = "${member.name}.${inner.name}"
}

class MemberAction1<A> internal constructor(val member: Member<*, *, *, *>, val action: Action1<A>) : Trigger {
    override val name get() = "${member.name}.${action.name}"
    operator fun invoke(a: A): Trigger = MemberTrigger(member, action(a))
}

class MemberAction3<A, B, C> internal constructor(val member: Member<*, *, *, *>, val action: Action3<A, B, C>) : Trigger {
    override val name get() = "${member.name}.${action.name}"
    operator fun invoke(a: A, b: B, c: C): Trigger = MemberTrigger(member, action(a, b, c))
}

/** One machine inside a composition, with the projection from the composite state to its own. */
class Member<C, S, O, F> internal constructor(val name: String, val machine: Machine<S, O, F>, val project: KProperty1<C, S>) {
    private fun <T : Trigger> checked(t: T): T = t.also {
        check(machine.actions.contains(it)) { "composition member $name: ${machine.name} has no action '${it.name}'" }
    }

    operator fun get(action: Action0): Trigger = MemberTrigger(this, checked(action))
    operator fun get(timer: Timer): Trigger = MemberTrigger(this, checked(timer))
    operator fun <A> get(action: Action1<A>): MemberAction1<A> = MemberAction1(this, checked(action))
    operator fun <A, B, C> get(action: Action3<A, B, C>): MemberAction3<A, B, C> = MemberAction3(this, checked(action))
}

/** Two member actions firing as one, under the name of the composite action. */
class Sync internal constructor(val name: Trigger, val left: Trigger, val right: Trigger)

/** Outcomes and facts of a composition are the members' own, tagged by member. */
data class Tagged(val member: String, val value: Any?)

class Composition<C : Any> internal constructor(
    name: String,
    stateType: Finite<C>,
    starts: List<C>,
    isEnd: C.() -> Boolean,
    val members: List<Member<C, *, *, *>>,
    val syncs: List<Sync>,
) : Machine<C, Tagged, Tagged>(
    name, null, stateType, starts, isEnd, emptyList(), emptyList(), { null }, composedSteps(members, syncs), null,
) {
    /** The handle a Scenario or Property outside the `compose` block reaches a member through. */
    @Suppress("UNCHECKED_CAST")
    fun <S> member(project: KProperty1<C, S>): Member<C, S, *, *> =
        members.firstOrNull { it.project == project } as Member<C, S, *, *>?
            ?: throw IllegalArgumentException("compose $name: no member projected by ${project.name}")

    /** A composition's actions are its syncs plus every member action no sync names. */
    override val actions: kotlin.collections.Set<Trigger>
        get() = steps.keys
}

/**
 * The product's step functions. A sync fires both member steps on the projected states and combines
 * their rows; a member action no sync names fires alone and leaves the other members' states as they
 * are. Outcomes and facts are the members' own, tagged by member. Body elided: it needs the member
 * projections' inverses (rebuilding `C` from member states), which the sketch takes from the data
 * class's primary constructor by parameter name.
 */
private fun <C : Any> composedSteps(
    members: List<Member<C, *, *, *>>,
    syncs: List<Sync>,
): Map<Trigger, (C, List<Any?>) -> List<Step<C, Tagged, Tagged>>> {
    val synced = syncs.flatMap { listOf(it.left, it.right) }.toSet()
    val free = members.flatMap { m -> m.machine.actions.map { MemberTrigger(m, it) } }.filter { it !in synced }
    return (syncs.map { it.name } + free).associateWith { TODO("step function of the product row") }
}

@UmpireDsl
class ComposeScope<C : Any> internal constructor(private val name: String) {
    internal val members = ArrayList<Member<C, *, *, *>>()
    internal val syncs = ArrayList<Sync>()
    internal var starts: List<C> = emptyList()
    internal var ends: (C.() -> Boolean)? = null

    /** `val operation = member(NexusCallerState::operation, nexusProtocol)`: the property is the projection. */
    fun <S, O, F> member(project: KProperty1<C, S>, machine: Machine<S, O, F>): Member<C, S, O, F> =
        Member(project.name, machine, project).also { members += it }

    /** `workerStop syncs operation[workerStop] with worker[workerStop]`. */
    infix fun Trigger.syncs(left: Trigger): SyncHalf = SyncHalf(this, left)

    inner class SyncHalf(private val composite: Trigger, private val left: Trigger) {
        infix fun with(right: Trigger) {
            check((left as MemberTrigger).member != (right as MemberTrigger).member) {
                "compose $name: sync '${composite.name}' pairs two actions of the same member"
            }
            syncs += Sync(composite, left, right)
        }
    }

    fun starts(vararg states: C) {
        starts = states.toList()
    }

    fun ends(predicate: C.() -> Boolean) {
        ends = predicate
    }
}

inline fun <reified C : Any> compose(name: String, noinline configure: ComposeScope<C>.() -> Unit): Composition<C> =
    composeOf(name, Finite.of<C>(), configure)

fun <C : Any> composeOf(name: String, stateType: Finite<C>, configure: ComposeScope<C>.() -> Unit): Composition<C> {
    val scope = ComposeScope<C>(name).apply(configure)
    check(scope.members.size >= 2) { "compose $name: fewer than two members" }
    check(scope.members.map { it.machine.entity }.distinct().size == scope.members.size) {
        "compose $name: members must be machines of different entities"
    }
    return Composition(name, stateType, scope.starts, checkNotNull(scope.ends) { "compose $name: no ends { }" }, scope.members, scope.syncs)
}

// ---------------------------------------------------------------------------------------------
// Internals shared above
// ---------------------------------------------------------------------------------------------

/** How an input value prints in a key: `syncSuccess`, `handlerError-true`, `unset`. */
internal fun className(value: Any?): String = when (value) {
    is Enum<*> -> value.name.replaceFirstChar(Char::lowercase)
    null -> "none"
    else -> value.toString().replaceFirstChar(Char::lowercase) // data object -> its name, data class -> "HandlerError(retryable=true)" normalized
}

internal fun cartesian(domains: List<List<Any?>>): List<List<Any?>> =
    domains.fold(listOf(emptyList())) { acc, domain -> acc.flatMap { prefix -> domain.map { prefix + it } } }
