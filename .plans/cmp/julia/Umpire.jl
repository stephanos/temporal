"""
    Umpire

The framework surface the Model files author against. Nothing in this file has been run through a
toolchain; it shows the types the Models bind to, the shape of the DSL macros, and where each check
runs. Bodies that are not load-bearing for reading a Model are sketched with a comment.

Three Julia facts carry the design:

1. A top-level `include` expands and evaluates one form at a time. So when `@machine` expands, every
   `@action` above it has already been evaluated and registered, and the macro can reject an
   undeclared action at expansion time, pinned to the offending line.
2. A `const` computed at top level inside a package is computed once, during precompilation, and
   cached. The state table, the refinement walk and the Query searches are ordinary functions called
   from the `const` the macros emit, so a Model that fails its refinement fails to precompile.
3. Multiple dispatch. `finite`, `key`, `keypart` and `factname` are one generic function each with
   one method per representation (Base enums, EnumX enums, Moshi sum types, `Fin`, structs), so a
   new state type needs no registration to be enumerated, named and compared.
"""
module Umpire

using EnumX: @enumx            # flat enums with a namespace: `Phase.scheduled`, type `Phase.T`
using Moshi.Data: @data        # sum types with fields: `Reply.handlerError(true)`, type `Reply.Type`
using Moshi.Match: @match      # pattern matching over both, plus literals
import Moshi

export Step, Machine, Refinement, Property, Scenario, Limits, Query, Compose, ActionClass
export Fin, finite, key, with, saturatingSucc
export @enumx, @data, @match
export @entity, @action, @observation, @machine, @compose, @property, @scenario, @limits, @query,
    @set

# ---------------------------------------------------------------------------------------------------
# Finite domains
# ---------------------------------------------------------------------------------------------------

"""
    Fin{N}

An integer in `0:N`. Lean's `Fin (attemptBound + 1)`; Julia has no range types, so the bound is a
type parameter and `convert` lets a `@kwdef` constructor accept a plain literal.
"""
struct Fin{N}
    n::Int
    function Fin{N}(n::Integer) where {N}
        0 <= n <= N || throw(DomainError(n, "Fin{$N}: outside 0:$N"))
        return new{N}(Int(n))
    end
end
Base.convert(::Type{Fin{N}}, n::Integer) where {N} = Fin{N}(n)
Base.:(==)(a::Fin{N}, b::Integer) where {N} = a.n == b

"""The successor that stays inside the bound, so a retry cannot leave the finite state space."""
saturatingSucc(a::Fin{N}) where {N} = Fin{N}(min(a.n + 1, N))

"""
    finite(T) -> Vector{T}

Every value of a finite type, in a fixed order. One generic function, one method per representation;
the struct method is the cartesian product of its fields' domains, so `Base.@kwdef struct` state
types are finite with no derive step. `Iterators.product` varies the first field fastest and
allocates nothing until `vec` copies the (isbits) values out, which is why 288 states cost nothing
to enumerate and a table of a few thousand rows builds in milliseconds.
"""
finite(::Type{T}) where {T<:Enum} = collect(instances(T))         # Base.@enum and EnumX both
finite(::Type{Bool}) = [false, true]
finite(::Type{Fin{N}}) where {N} = Fin{N}.(0:N)
function finite(::Type{T}) where {T}
    if Moshi.Data.is_data_type(T)
        # A variant with fields contributes one member per assignment of them:
        # `handlerError(retryable::Bool)` is one constructor and two classes, as in Lean.
        out = T[]
        for ctor in Moshi.Data.variants(T)
            fts = Moshi.Data.variant_fieldtypes(ctor)
            # A singleton variant is a value, not a constructor to call.
            isempty(fts) ? push!(out, ctor) :
                append!(out, ctor(vals...) for vals in Iterators.product(finite.(fts)...))
        end
        return out
    end
    isstructtype(T) && isconcretetype(T) ||
        throw(ArgumentError("finite: $T is not an enum, a Fin, a @data sum type or a concrete struct"))
    return vec([T(vals...) for vals in Iterators.product((finite(ft) for ft in fieldtypes(T))...)])
end

"""
    with(state; field = value, ...)

Lean's `{ state with phase := .started }`. Rebuilds the struct with the named fields replaced; the
rest are copied. Kept here rather than pulling Accessors.jl, whose `@set` would shadow ours.
"""
function with(s::T; kw...) where {T}
    return T((haskey(kw, f) ? kw[f] : getfield(s, f) for f in fieldnames(T))...)
end

"""
    keypart(x) -> String

One segment of a state or class key. Dispatch picks the spelling: an enum by its name, a `Fin` by
its number, a sum-type variant by its name and then its fields, so `handlerError(true)` keys as
`handlerError-true` from one generic function.
"""
keypart(x::Enum) = string(Symbol(x))
keypart(x::Fin) = string(x.n)
keypart(x::Bool) = string(x)
function keypart(x)
    if Moshi.Data.is_data_type(typeof(x))
        return join((string(Moshi.Data.variant_name(x)), keypart.(Moshi.Data.variant_fields(x))...), "-")
    end
    return string(x)
end

"""The name a Lean reader gives a state: every field's `keypart`, dash-joined (`scheduled-0-unset-unset-unset`)."""
key(s::T) where {T} = join((keypart(getfield(s, f)) for f in fieldnames(T)), "-")

"""The constructor name of a fact, which is what an `evidence` line is keyed by."""
factname(f::Enum) = Symbol(f)
factname(f) = Moshi.Data.variant_name(f)

# ---------------------------------------------------------------------------------------------------
# Steps
# ---------------------------------------------------------------------------------------------------

"""
    Step{S,O,F}

`{ outcome, state, facts }`. A step function `S -> inputs... -> Vector{Step{S,O,F}}` returns an
empty vector when the action is not enabled. Models alias their instance
(`const PStep = Step{ProductState,ProductOutcome.T,ProductFact.T}`) so `facts = []` types as
`Vector{F}` rather than `Vector{Any}`.
"""
struct Step{S,O,F}
    outcome::O
    state::S
    facts::Vector{F}
    Step{S,O,F}(; outcome, state, facts = F[]) where {S,O,F} = new{S,O,F}(outcome, state, collect(F, facts))
end

"""One action with one assignment of its finite inputs; `inputs === nothing` names the action's every class."""
struct ActionClass
    action::Symbol
    inputs::Union{Nothing,Tuple}
end
key(c::ActionClass) = c.inputs === nothing ? string(c.action) :
    join((string(c.action), keypart.(c.inputs)...), "-")

# ---------------------------------------------------------------------------------------------------
# The vocabulary registries
# ---------------------------------------------------------------------------------------------------

struct Entity
    name::Symbol
    key::Union{Nothing,Symbol}
    refer::Dict{Symbol,Symbol}
end

struct Action
    name::Symbol
    party::Symbol
    creates::Union{Nothing,Symbol}
    on::Union{Nothing,Symbol}
    schema::Union{Nothing,String}
    input::Vector{Pair{Symbol,DataType}}
    results::Union{Nothing,DataType}
    examples::Vector{Pair{Any,String}}
end

struct Observation
    name::Symbol
    on::Symbol
    read::Symbol
end

# Populated by the `@entity`/`@action`/`@observation` forms as they are evaluated, top to bottom.
# `@machine` reads ACTIONS at expansion time, which is legitimate because a later top-level form is
# not expanded before an earlier one has run.
const ENTITIES = Dict{Symbol,Entity}()
const ACTIONS = Dict{Symbol,Action}()
const OBSERVATIONS = Dict{Symbol,Observation}()

"""Every class of an action: the cartesian product of its input domains, one `ActionClass` each."""
function classes(a::Action)
    isempty(a.input) && return [ActionClass(a.name, ())]
    return vec([ActionClass(a.name, Tuple(vals))
                for vals in Iterators.product((finite(t) for (_, t) in a.input)...)])
end

"""A system timer the machine owns: a party-`system` action with no input and no entity."""
timer(name::Symbol) = Action(name, :system, nothing, nothing, nothing, Pair{Symbol,DataType}[], nothing, [])

# ---------------------------------------------------------------------------------------------------
# The state table, the machine and the refinement
# ---------------------------------------------------------------------------------------------------

"""
    Table{S,O,F}

Every state times every action class, each with the steps the step function returned. A row with no
steps is an action that is not enabled there. Building it calls every step function on every
state and class, which is also what makes a missing `@match` arm surface here rather than in a
test: total enumeration exercises every arm.
"""
struct Table{S,O,F}
    states::Vector{S}
    classes::Vector{ActionClass}
    rows::Dict{Tuple{S,ActionClass},Vector{Step{S,O,F}}}
end

function build_table(::Type{S}, ::Type{O}, ::Type{F}, actions::Vector{Action},
                     steps::Dict{Symbol,<:Function}) where {S,O,F}
    states = finite(S)
    cls = reduce(vcat, classes.(actions); init = ActionClass[])
    rows = Dict{Tuple{S,ActionClass},Vector{Step{S,O,F}}}()
    sizehint!(rows, length(states) * length(cls))
    for s in states, c in cls
        rows[(s, c)] = steps[c.action](s, c.inputs...)::Vector{Step{S,O,F}}
    end
    return Table{S,O,F}(states, cls, rows)
end

"""
    Refinement

`refines: target` with `map: f`. For every protocol transition `(s, a, s')`, either
`map(s) == map(s')` (a product stutter) or the product has some transition from `map(s)` to
`map(s')` under any action class. The check is by mapped states, not by action name: a protocol
timer row maps to the product's `timeout` row, and a retryable failure that the product reads as a
different action is still covered. `rows` keeps the verdict per protocol row keyed as Lean does
(`scheduled-0-unset-unset-unset-handlerReply-async`): `nothing` for a stutter, the product row's
key otherwise; `rejected` is the first row that fits neither, or `nothing`.
"""
struct Refinement
    target::Any                       # ::Machine; abstract to avoid a recursive parametric type
    map::Function
    rows::Dict{String,Union{Nothing,String}}
    rejected::Union{Nothing,String}
end

mutable struct Machine{S,O,F}
    name::Symbol
    entity::Union{Nothing,Symbol}
    starts::Vector{S}
    ends::Vector{S}
    timers::Vector{Symbol}
    unobservable::Vector{Symbol}
    evidence::Dict{Symbol,Symbol}
    actions::Vector{Action}
    steps::Dict{Symbol,Function}
    table::Table{S,O,F}
    refinement::Union{Nothing,Refinement}
end

Base.show(io::IO, m::Machine) = print(io, "Machine(", m.name, ", ", length(m.table.states), " states)")

"""Rows with at least one step, as `(state, class, step)` triples."""
transitions(m::Machine) = [(s, c, st) for ((s, c), steps) in m.table.rows for st in steps]

"""Every action class the machine steps on, in catalog order (sorted by key, as the Lean table is)."""
actionKeys(m::Machine) = sort!(key.(m.table.classes))

"""The states a walk from `starts` over the table reaches."""
function reachable(m::Machine{S}) where {S}
    seen = Base.Set{S}(m.starts); frontier = copy(m.starts)
    while !isempty(frontier)
        s = pop!(frontier)
        for c in m.table.classes, st in m.table.rows[(s, c)]
            st.state in seen || (push!(seen, st.state); push!(frontier, st.state))
        end
    end
    return collect(seen)
end

"""A reachable non-end state with no enabled action, or `nothing`; Lean's `stuck`."""
function stuck(m::Machine)
    for s in reachable(m)
        s in m.ends && continue
        any(!isempty(m.table.rows[(s, c)]) for c in m.table.classes) || return s
    end
    return nothing
end

"""The refinement walk: mapped states only, as the Lean checker does."""
function check_refinement(proto::Machine, target::Machine, map::Function)
    rows = Dict{String,Union{Nothing,String}}()
    rejected = nothing
    for (s, c, st) in transitions(proto)
        rowkey = key(s) * "-" * key(c)
        before, after = map(s), map(st.state)
        if before == after
            rows[rowkey] = nothing                       # a product stutter
            continue
        end
        hit = findfirst(pc -> any(pst -> pst.state == after, target.table.rows[(before, pc)]),
                        target.table.classes)
        if hit === nothing
            rejected = something(rejected, rowkey)
            rows[rowkey] = "rejected"
        else
            rows[rowkey] = key(before) * "-" * key(target.table.classes[hit])
        end
    end
    return Refinement(target, map, rows, rejected)
end

"""
    build_machine(name; kw...)

What `@machine` emits a `const` of. Enumerates the state type, builds the table, and if `refines`
is given walks the refinement and throws when it is rejected, so the failure is a precompile
failure of the Model package rather than a red test.
"""
function build_machine(name::Symbol; entity, state::Type{S}, outcome::Type{O}, fact::Type{F},
                       starts, ends, timers = Symbol[], unobservable = Symbol[],
                       evidence = Dict{Symbol,Symbol}(), steps, refines = nothing, map = nothing) where {S,O,F}
    actions = Action[haskey(ACTIONS, a) ? ACTIONS[a] : timer(a) for a in keys(steps)]
    table = build_table(S, O, F, actions, steps)
    m = Machine{S,O,F}(name, entity, starts, ends, timers, unobservable, evidence, actions, steps,
                       table, nothing)
    if refines !== nothing
        m.refinement = check_refinement(m, refines, map)
        m.refinement.rejected === nothing ||
            error("machine $name: refinement of $(refines.name) rejected at row `$(m.refinement.rejected)`")
    end
    return m
end

"""`from: machine` with `restrict: [actions]`: the same table with the other actions' rows dropped."""
function restrict(m::Machine{S,O,F}, keep::Vector{Symbol}, name::Symbol) where {S,O,F}
    steps = Dict(a => f for (a, f) in m.steps if a in keep)
    actions = filter(a -> a.name in keep, m.actions)
    return Machine{S,O,F}(name, m.entity, m.starts, m.ends, filter(in(keep), m.timers),
                          filter(in(keep), m.unobservable), m.evidence, actions, steps,
                          build_table(S, O, F, actions, steps), nothing)
end

"""The states of `S` whose `phase` is one of `phases`, other fields at every value (Lean's `ends: [..]`)."""
ending(::Type{S}, phases::Vector{Symbol}) where {S} =
    [s for s in finite(S) if Symbol(keypart(s.phase)) in phases]

"""The one state per phase whose other fields are at their first finite value (Lean's `starts: [..]`)."""
starting(::Type{S}, phases::Vector{Symbol}) where {S} =
    [first(s for s in finite(S) if Symbol(keypart(s.phase)) == p) for p in phases]

# ---------------------------------------------------------------------------------------------------
# Composition
# ---------------------------------------------------------------------------------------------------

"""
    Compose

The product of machines of different entities. A `sync` pair fires two member actions as one step
under one name; a member action no `sync` names stays executable on its own under its dotted name
(`operation.schedule`). Built as a `Machine` over the composite state type so Properties, Scenarios
and Queries read it as any other machine.
"""
struct Compose
    name::Symbol
    entities::Vector{Symbol}
    members::Vector{Pair{Symbol,Machine}}
    sync::Dict{Symbol,Vector{Pair{Symbol,Symbol}}}   # name => [member => action, ...]
    machine::Machine
end

function build_compose(name::Symbol; entities, state::Type{S}, members, sync, starts, ends) where {S}
    # Sketched. The composite table is the product of the member tables: a synced step is the pair
    # of member steps under the sync name, facts concatenated; an unsynced member step keeps the
    # other members' states. `starts`/`ends` are resolved per member as `member.phase`.
    machine = error("build_compose: sketched")
    return Compose(name, entities, members, sync, machine)
end

"""A Property or Scenario names a machine or a composition; both read as the machine."""
machine_of(m::Machine) = m
machine_of(c::Compose) = c.machine

# ---------------------------------------------------------------------------------------------------
# Properties, Scenarios, Limits, Queries, Sets
# ---------------------------------------------------------------------------------------------------

"""
    Property

`when` set: a same-step claim, `holds(step)`; unset: a transition claim, `holds(before, after)`. The
arity is read off the function at build time rather than declared, which is what lets the author
write `holds = step -> ...` or `holds = (before, after) -> ...` with no annotation.
"""
struct Property
    name::Symbol
    machine::Machine
    when::Union{Nothing,ActionClass}
    holds::Function
    function Property(name, machine, when, holds)
        arity = when === nothing ? 2 : 1
        hasmethod(holds, NTuple{arity,Any}) ||
            error("property $name: `holds` must take $(arity) argument(s) " *
                  (arity == 1 ? "(a same-step claim under `when`)" : "(a transition claim: before, after)"))
        return new(name, machine, when, holds)
    end
end

struct Scenario
    name::Symbol
    model::Machine
    starts::Any                      # a start state of `model`
    actions::Vector{ActionClass}
end

struct Limits
    name::Symbol
    steps::Int
    actions::Int
    search::Int
end

struct Query
    name::Symbol
    kind::Symbol                     # :find | :verify
    property::Property
    scenario::Scenario
    limits::Limits
end

"""
    Set

Shadows `Base.Set` inside this module (the framework uses `Base.Set` where it needs one) and is not
exported, so a Model refers to it as `Umpire.Set`. The spec's name is kept on purpose.
"""
struct Set
    name::Symbol
    purpose::Symbol                  # :functional | :canary | :exploratory
    bind::Dict{Symbol,Symbol}        # party => :driven | :observed
    repeat::Union{Nothing,Symbol}
    queries::Vector{Query}
    machine::Union{Nothing,Machine}
    cover::Vector{Symbol}
    budget::Union{Nothing,Limits}
end

# ---------------------------------------------------------------------------------------------------
# Search
# ---------------------------------------------------------------------------------------------------

struct Witness{S,O,F}
    path::Vector{Tuple{ActionClass,Step{S,O,F}}}
end

struct Found
    query::Query
    witness::Witness
end

struct Verified
    query::Query
    paths::Int
end

struct SearchFailed <: Exception
    query::Query
    reason::String
end
Base.showerror(io::IO, e::SearchFailed) =
    print(io, "query ", e.query.name, ": ", e.reason)

"""
    run(query) -> Found | Verified

Walks the table from the Scenario's start over every enabled class, at most `limits.steps` deep and
at most `limits.search` candidates. `find`: the witness is the candidate whose classes are the
Scenario's actions in order and whose `when` step satisfies the Property; `verify`: the Property
holds on every candidate along which the Scenario's actions occur, and nothing is realized.
Throws `SearchFailed` otherwise, so a `const` of it fails precompile the way a Lean `query` command
fails elaboration.
"""
function run(q::Query)
    # Sketched: iterative DFS over q.scenario.model.table.rows with the two cuts above.
    error("run: sketched")
end

# ---------------------------------------------------------------------------------------------------
# The DSL
# ---------------------------------------------------------------------------------------------------

"""An authoring mistake, pinned to the line of the `key = value` that carries it."""
struct DSLError <: Exception
    msg::String
    line::LineNumberNode
end
Base.showerror(io::IO, e::DSLError) =
    print(io, "Umpire: ", e.msg, "\n  at ", e.line.file, ":", e.line.line)
dslerror(msg, line) = throw(DSLError(msg, line))

"""
    keyed(block, allowed, __source__, what) -> Dict{Symbol,(value, line)}

The one parser every command shares. A command body is a `begin ... end` block of `key = value`
lines; Julia has already parsed it, so this only walks `block.args`, tracks the `LineNumberNode`
that precedes each line, and rejects an unknown or repeated key at that line. `=` was chosen over
`key: value` because assignment has the lowest precedence and so swallows a `->` lambda, a `|`
union or a `[...]` list without parentheses; the cost is `var"for"` and `var"in"`, since both are
Julia keywords.
"""
function keyed(block, allowed, __source__::LineNumberNode, what::AbstractString)
    Meta.isexpr(block, :block) ||
        dslerror("$what: expected a `begin ... end` block of `key = value` lines", __source__)
    out = Dict{Symbol,Tuple{Any,LineNumberNode}}()
    line = __source__
    for arg in block.args
        if arg isa LineNumberNode
            line = arg
            continue
        end
        Meta.isexpr(arg, :(=), 2) && arg.args[1] isa Symbol ||
            dslerror("$what: expected `key = value`, got `$arg`", line)
        k = arg.args[1]
        k in allowed || dslerror("$what: unknown key `$k`; one of $(join(allowed, ", "))", line)
        haskey(out, k) && dslerror("$what: `$k` given twice", line)
        out[k] = (arg.args[2], line)
    end
    return out
end

"""`a = b` lines of a nested `begin ... end` (or a `(a = b, c = d)` tuple) as `Symbol => Expr` pairs."""
function pairs_of(value, line)
    Meta.isexpr(value, :block) && return [a.args[1] => a.args[2] for a in value.args if Meta.isexpr(a, :(=))]
    Meta.isexpr(value, :tuple) && return [a.args[1] => a.args[2] for a in value.args if Meta.isexpr(a, :(=))]
    Meta.isexpr(value, :(=)) && return [value.args[1] => value.args[2]]
    dslerror("expected `a = b` lines, got `$value`", line)
end

"""`[a, b, c]` as a `Vector{Symbol}`; `a | b | c` as the same."""
function names_of(value, line)
    Meta.isexpr(value, :vect) && return Symbol[v for v in value.args]
    value isa Symbol && return [value]
    Meta.isexpr(value, :call) && value.args[1] == :| &&
        return vcat(names_of(value.args[2], line), names_of(value.args[3], line))
    dslerror("expected `[a, b, ...]`, got `$value`", line)
end

"""
    resolve(T, ex)

A class member written bare, as Lean writes `.unset`: `unset` against `Timeout.T` is
`Timeout.unset`; `handlerError(false)` against `Reply.Type` is `Reply.handlerError(false)`. Both
EnumX and Moshi put a type's members in a module of the type's name, which is what makes one
`getproperty` on `parentmodule(T)` do for both.
"""
resolve(::Type{T}, ex::Symbol) where {T} = getproperty(parentmodule(T), ex)
resolve(::Type{T}, ex::Expr) where {T} =
    getproperty(parentmodule(T), ex.args[1])(ex.args[2:end]...)
resolve(::Type{T}, lit) where {T} = lit

"""`handlerReply(handlerError(false))` as an `ActionClass`, inputs resolved against the action's input types."""
function classof(action::Symbol, args::Union{Nothing,Tuple})
    args === nothing && return ActionClass(action, nothing)
    base = Symbol(last(split(String(action), '.')))                  # `activity.start` is declared as `start`
    a = haskey(ACTIONS, base) ? ACTIONS[base] : timer(base)           # a timer has no registry entry
    length(args) == length(a.input) ||
        throw(ArgumentError("$action takes $(length(a.input)) input(s), $(length(args)) given"))
    return ActionClass(action, Tuple(resolve(t, ex) for ((_, t), ex) in zip(a.input, args)))
end

"""`schedule(unset, expires, unset)` or `operation.schedule(...)` or bare `workerStop`, quoted for `classof`."""
function class_expr(ex, line)
    if ex isa Symbol
        return :(Umpire.classof($(QuoteNode(ex)), ()))
    elseif Meta.isexpr(ex, :.)                                     # operation.scheduleToStart
        return :(Umpire.classof($(QuoteNode(Symbol(ex.args[1], ".", ex.args[2].value))), ()))
    elseif Meta.isexpr(ex, :call)
        head = ex.args[1]
        name = head isa Symbol ? head : Symbol(head.args[1], ".", head.args[2].value)
        return :(Umpire.classof($(QuoteNode(name)), $(Expr(:tuple, QuoteNode.(ex.args[2:end])...))))
    end
    dslerror("expected an action, `action(inputs...)` or `member.action(...)`, got `$ex`", line)
end

macro entity(name, block = Expr(:block))
    kw = keyed(block, (:key, :refer), __source__, "entity $name")
    k = haskey(kw, :key) ? QuoteNode(kw[:key][1]) : nothing
    refer = haskey(kw, :refer) ? Dict(a => b for (a, b) in pairs_of(kw[:refer]...)) : Dict{Symbol,Symbol}()
    return quote
        Umpire.ENTITIES[$(QuoteNode(name))] = Umpire.Entity($(QuoteNode(name)), $k, $refer)
        nothing
    end
end

"""
    @action name begin party = ...; creates|on = entity; schema = "..."; input = (f = T, ...); results = T; examples = (...) end

Registers the action. `input` types are evaluated in the Model's module, so `Timeout.T` and
`Reply.Type` are the actual types and `classes` can enumerate them. `creates`/`on` must name a
declared entity, checked at expansion time against ENTITIES.
"""
macro action(name, block = Expr(:block))
    kw = keyed(block, (:party, :creates, :on, :schema, :input, :results, :examples), __source__,
               "action $name")
    haskey(kw, :party) || dslerror("action $name: `party` is required", __source__)
    for k in (:creates, :on)
        haskey(kw, k) && !haskey(ENTITIES, kw[k][1]) &&
            dslerror("action $name: `$k = $(kw[k][1])` names no declared entity", kw[k][2])
    end
    opt(k) = haskey(kw, k) ? QuoteNode(kw[k][1]) : nothing
    input = haskey(kw, :input) ?
        Expr(:vect, (:($(QuoteNode(f)) => $(esc(t))) for (f, t) in pairs_of(kw[:input]...))...) :
        :(Pair{Symbol,DataType}[])
    schema = haskey(kw, :schema) ? string(kw[:schema][1]) : nothing
    results = haskey(kw, :results) ? esc(kw[:results][1]) : nothing
    examples = haskey(kw, :examples) ?
        Expr(:vect, (:($(QuoteNode(c)) => $(string(v))) for (c, v) in pairs_of(kw[:examples]...))...) :
        :(Pair{Any,String}[])
    return quote
        Umpire.ACTIONS[$(QuoteNode(name))] = Umpire.Action($(QuoteNode(name)), $(opt(:party)),
            $(opt(:creates)), $(opt(:on)), $schema, $input, $results, $examples)
        nothing
    end
end

macro observation(name, block)
    kw = keyed(block, (:on, :read), __source__, "observation $name")
    return quote
        Umpire.OBSERVATIONS[$(QuoteNode(name))] =
            Umpire.Observation($(QuoteNode(name)), $(QuoteNode(kw[:on][1])), $(QuoteNode(kw[:read][1])))
        nothing
    end
end

"""
    @machine name begin var"for" = entity; state = S; outcome = O; fact = F; starts = [..]; ends = [..];
                        timers = [..]; unobservable = [..]; evidence = (..); steps = (action = fn, ..);
                        refines = m; map = f end
    @machine name begin from = m; restrict = [actions...] end

Emits `const name = Umpire.build_machine(...)`. Expansion-time checks, each pinned to its line:
every `steps` key is a declared action or a listed timer; every `timers`/`unobservable` entry is a
`steps` key; `refines` and `map` come together. Everything else (the table, the refinement) runs
when the `const` is evaluated, i.e. at precompile of the Model package.
"""
macro machine(name, block)
    kw = keyed(block, (:for, :state, :outcome, :fact, :starts, :ends, :timers, :unobservable, :evidence,
                       :steps, :refines, :map, :from, :restrict), __source__, "machine $name")
    if haskey(kw, :from)
        keep = names_of(kw[:restrict]...)
        return :(const $(esc(name)) = Umpire.restrict($(esc(kw[:from][1])), $keep, $(QuoteNode(name))))
    end
    for k in (:for, :state, :outcome, :fact, :starts, :ends, :steps)
        haskey(kw, k) || dslerror("machine $name: `$k` is required", __source__)
    end
    timers = haskey(kw, :timers) ? names_of(kw[:timers]...) : Symbol[]
    steps = pairs_of(kw[:steps]...)
    for (action, _) in steps
        haskey(ACTIONS, action) || action in timers ||
            dslerror("machine $name: `steps` names `$action`, which is neither a declared action nor " *
                     "one of this machine's `timers`", kw[:steps][2])
    end
    for t in timers
        any(==(t), first.(steps)) ||
            dslerror("machine $name: timer `$t` has no `steps` entry", kw[:timers][2])
    end
    haskey(kw, :refines) == haskey(kw, :map) ||
        dslerror("machine $name: `refines` and `map` come together", __source__)
    S = esc(kw[:state][1])
    evidence = haskey(kw, :evidence) ?
        Expr(:call, :Dict, (:($(QuoteNode(a)) => $(QuoteNode(b))) for (a, b) in pairs_of(kw[:evidence]...))...) :
        :(Dict{Symbol,Symbol}())
    stepdict = Expr(:call, :Dict, (:($(QuoteNode(a)) => $(esc(f))) for (a, f) in steps)...)
    phases(k) = Expr(:vect, QuoteNode.(names_of(kw[k]...))...)
    return quote
        const $(esc(name)) = Umpire.build_machine($(QuoteNode(name));
            entity = $(QuoteNode(kw[:for][1])),
            state = $S, outcome = $(esc(kw[:outcome][1])), fact = $(esc(kw[:fact][1])),
            starts = Umpire.starting($S, $(phases(:starts))),
            ends = Umpire.ending($S, $(phases(:ends))),
            timers = $timers,
            unobservable = $(haskey(kw, :unobservable) ? names_of(kw[:unobservable]...) : Symbol[]),
            evidence = $evidence,
            steps = $stepdict,
            refines = $(haskey(kw, :refines) ? esc(kw[:refines][1]) : nothing),
            map = $(haskey(kw, :map) ? esc(kw[:map][1]) : nothing))
    end
end

"""
    @compose name begin var"for" = [e1, e2]; state = S; members = (m1 = machine, ..);
                        sync = (name = m1.action ∥ m2.action, ..); starts = [m1.phase, ..]; ends = [..] end

`∥` (U+2225) is an infix operator Julia's parser already knows, so `a.x ∥ b.y` arrives as a call
expression and no parsing is done here.
"""
macro compose(name, block)
    kw = keyed(block, (:for, :state, :members, :sync, :starts, :ends), __source__, "compose $name")
    members = Expr(:vect, (:($(QuoteNode(m)) => $(esc(f))) for (m, f) in pairs_of(kw[:members]...))...)
    function syncpair(ex, line)
        Meta.isexpr(ex, :call) && ex.args[1] == :∥ ||
            dslerror("compose $name: a `sync` line is `name = a.action ∥ b.action`, got `$ex`", line)
        return Expr(:vect, (:($(QuoteNode(side.args[1])) => $(QuoteNode(side.args[2].value))) for side in ex.args[2:3])...)
    end
    sync = Expr(:call, :Dict, (:($(QuoteNode(n)) => $(syncpair(ex, kw[:sync][2]))) for (n, ex) in pairs_of(kw[:sync]...))...)
    dotted(v) = Expr(:vect, (:($(QuoteNode(d.args[1])) => $(QuoteNode(d.args[2].value))) for d in v.args)...)
    return quote
        const $(esc(name)) = Umpire.build_compose($(QuoteNode(name));
            entities = $(Expr(:vect, QuoteNode.(kw[:for][1].args)...)),
            state = $(esc(kw[:state][1])), members = $members, sync = $sync,
            starts = $(dotted(kw[:starts][1])), ends = $(dotted(kw[:ends][1])))
    end
end

"""
    @property name begin machine = m; when = action(class); holds = step -> ... end
    @property name begin machine = m; holds = (before, after) -> ... end
"""
macro property(name, block)
    kw = keyed(block, (:machine, :when, :holds), __source__, "property $name")
    when = haskey(kw, :when) ? class_or_action(kw[:when]...) : nothing
    return quote
        const $(esc(name)) = Umpire.Property($(QuoteNode(name)), Umpire.machine_of($(esc(kw[:machine][1]))),
                                              $when, $(esc(kw[:holds][1])))
    end
end

"""A bare action names every class (`when = handlerReply`); a call names one (`when = handlerReply(async)`)."""
class_or_action(ex::Symbol, line) = :(Umpire.classof($(QuoteNode(ex)), nothing))
class_or_action(ex, line) = class_expr(ex, line)

macro scenario(name, block)
    kw = keyed(block, (:model, :starts, :actions), __source__, "scenario $name")
    acts = kw[:actions][1]
    Meta.isexpr(acts, :vect) || dslerror("scenario $name: `actions` is a `[...]` list", kw[:actions][2])
    return quote
        const $(esc(name)) = Umpire.Scenario($(QuoteNode(name)), Umpire.machine_of($(esc(kw[:model][1]))),
            Umpire.start_of(Umpire.machine_of($(esc(kw[:model][1]))), $(QuoteNode(kw[:starts][1]))),
            [$((class_expr(a, kw[:actions][2]) for a in acts.args)...)])
    end
end

"""`starts = unscheduled` or `starts = operation.unscheduled` resolved to the model's start state of that phase."""
function start_of(m::Machine, phase::Symbol)
    i = findfirst(s -> Symbol(keypart(s.phase)) == phase, m.starts)
    i === nothing && throw(ArgumentError("scenario: $(m.name) has no start in phase `$phase`"))
    return m.starts[i]
end
start_of(m::Machine, dotted::Expr) = start_of(m, dotted.args[2].value)   # composition: member.phase

macro limits(name, block)
    kw = keyed(block, (:steps, :actions, :search), __source__, "limits $name")
    return :(const $(esc(name)) = Umpire.Limits($(QuoteNode(name)), $(kw[:steps][1]), $(kw[:actions][1]), $(kw[:search][1])))
end

"""
    @query name begin find|verify = property; var"in" = scenario; limits = l end

Emits the `Query` and, as a second `const`, its result: `run` throws when the search fails, so a
Query that does not find its Property fails the Model's precompile. `pins.jl` reads the result
consts (`syncCompletion_result`) rather than re-running the search.
"""
macro query(name, block)
    kw = keyed(block, (:find, :verify, :in, :limits), __source__, "query $name")
    haskey(kw, :find) ⊻ haskey(kw, :verify) ||
        dslerror("query $name: exactly one of `find`/`verify`", __source__)
    kind, prop = haskey(kw, :find) ? (:find, kw[:find][1]) : (:verify, kw[:verify][1])
    result = Symbol(name, :_result)
    return quote
        const $(esc(name)) = Umpire.Query($(QuoteNode(name)), $(QuoteNode(kind)), $(esc(prop)),
                                           $(esc(kw[:in][1])), $(esc(kw[:limits][1])))
        const $(esc(result)) = Umpire.run($(esc(name)))
    end
end

"""
    @set name begin purpose = functional|canary|exploratory; bind = (party = driven|observed, ..);
                    repeat = implementation; queries = [..]; machine = m; cover = rows | results | classMembers;
                    budget = limits end
"""
macro set(name, block)
    kw = keyed(block, (:purpose, :bind, :repeat, :queries, :machine, :cover, :budget), __source__, "set $name")
    purpose = kw[:purpose][1]
    purpose in (:functional, :canary, :exploratory) ||
        dslerror("set $name: `purpose` is functional, canary or exploratory", kw[:purpose][2])
    if purpose == :exploratory
        for k in (:machine, :cover, :budget)
            haskey(kw, k) || dslerror("set $name: an exploratory set needs `$k`", __source__)
        end
        haskey(kw, :queries) && dslerror("set $name: an exploratory set covers a machine, it lists no `queries`", kw[:queries][2])
    else
        haskey(kw, :queries) || dslerror("set $name: `queries` is required", __source__)
    end
    bind = Expr(:call, :Dict, (:($(QuoteNode(p)) => $(QuoteNode(b))) for (p, b) in pairs_of(kw[:bind]...))...)
    queries = haskey(kw, :queries) ? Expr(:vect, esc.(kw[:queries][1].args)...) : :(Umpire.Query[])
    cover = haskey(kw, :cover) ? names_of(kw[:cover]...) : Symbol[]
    return quote
        const $(esc(name)) = Umpire.Set($(QuoteNode(name)), $(QuoteNode(purpose)), $bind,
            $(haskey(kw, :repeat) ? QuoteNode(kw[:repeat][1]) : nothing), $queries,
            $(haskey(kw, :machine) ? esc(kw[:machine][1]) : nothing), $cover,
            $(haskey(kw, :budget) ? esc(kw[:budget][1]) : nothing))
    end
end

end # module Umpire
