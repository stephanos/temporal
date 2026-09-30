defmodule Umpire.Model do
  @moduledoc """
  The declaration macros, in the style of `Ecto.Schema` and `Phoenix.Router`: `use Umpire.Model`
  registers accumulating attributes and a `@before_compile` hook, each macro appends one IR node,
  and the hook checks them together.

  Every macro works in two passes, for a reason worth knowing when reading the code:

    1. **At expansion** it checks only its own shape: known keyword keys, an atom name, a head
       that is a call of plain variables, a scenario list of literal classes. Nothing else exists
       yet, so nothing else is checked.
    2. **At module-body evaluation** it lowers the declaration to IR (`Umpire.Lower`), resolving
       aliases and `@attributes`. This cannot happen at expansion: Elixir expands a module body
       before it evaluates it, so a macro that read `@running` while expanding would see `nil`.
       Functions a declaration emits (steps, maps, property predicates) use unquote fragments
       (`def unquote(fun)(...)` under `bind_quoted`), the same technique `Phoenix.Router` uses to
       define functions from data computed at evaluation.

  Cross-declaration checks (does the query's scenario exist, does the property's `when:` name an
  action of the scenario's machine, do the sync lines name member actions) run in
  `__before_compile__/1`, and each raises a `CompileError` at the offending declaration's line.
  """

  alias Umpire.{Diagnostic, Domain, IR, Lower}

  @accumulated [
    :umpire_entities,
    :umpire_domains,
    :umpire_states,
    :umpire_actions,
    :umpire_observations,
    :umpire_machines,
    :umpire_restrictions,
    :umpire_properties,
    :umpire_scenarios,
    :umpire_limits,
    :umpire_queries,
    :umpire_sets,
    :umpire_compositions
  ]

  @parties [:caller, :handler, :worker, :network, :operator]

  defmacro __using__(_opts) do
    quote do
      import Umpire.Model

      for attribute <- unquote(@accumulated) do
        Module.register_attribute(__MODULE__, attribute, accumulate: true)
      end

      @before_compile Umpire.Model
    end
  end

  @doc "The parties an action may name. `:system` is reserved for timers a machine owns."
  def parties, do: @parties

  ## Vocabulary

  @doc "`entity :operation, refer: [caller: :workflow], key: :scheduledEvent`"
  defmacro entity(name, opts \\ []) do
    loc = Diagnostic.loc(__CALLER__)
    Diagnostic.atom!(name, loc, "entity name")
    Diagnostic.keys!(opts, [:key, :refer], loc, "entity #{inspect(name)}")

    quote do
      @umpire_entities %IR.Entity{
        name: unquote(name),
        key: unquote(opts[:key]),
        refer: unquote(opts[:refer] || []),
        loc: unquote(Macro.escape(loc))
      }
    end
  end

  @doc """
  `domain AttemptResult, [:completed, {:failed, retryable: :boolean}, :canceled]`

  Defines `<Model>.AttemptResult` and aliases it. The member list is evaluated in the Model's
  scope, so a field type may be another domain's alias.
  """
  defmacro domain(alias_ast, members) do
    loc = Diagnostic.loc(__CALLER__)
    short = Diagnostic.alias!(alias_ast, loc, "domain name")
    Lower.members_shape!(members, loc, "domain #{short}")
    module = Module.concat(__CALLER__.module, short)

    quote do
      alias unquote(module)
      Umpire.Model.__domain__(unquote(module), unquote(members), unquote(Macro.escape(loc)), __ENV__)
    end
  end

  @doc false
  def __domain__(module, members, loc, env) do
    ir = Domain.declare!(module, members, loc)
    values = Domain.values(ir)

    Module.create(
      module,
      quote do
        @moduledoc false
        @type t :: unquote(Domain.typespec(ir))
        def __umpire_domain__, do: unquote(Macro.escape(ir))
        def values, do: unquote(Macro.escape(values))
        def size, do: unquote(length(values))
      end,
      Macro.Env.location(env)
    )

    Module.put_attribute(env.module, :umpire_domains, ir)
  end

  @doc """
  `defstate ProtocolState, phase: Phase, attempts: 0..@attemptBound, ...`

  Defines a struct module whose `values/0` is the product of its fields' types.
  """
  defmacro defstate(alias_ast, fields) do
    loc = Diagnostic.loc(__CALLER__)
    short = Diagnostic.alias!(alias_ast, loc, "state name")
    Lower.keyword!(fields, loc, "defstate #{short}")
    module = Module.concat(__CALLER__.module, short)

    quote do
      alias unquote(module)
      Umpire.Model.__defstate__(unquote(module), unquote(fields), unquote(Macro.escape(loc)), __ENV__)
    end
  end

  @doc false
  def __defstate__(module, fields, loc, env) do
    ir = Domain.declare_state!(module, fields, loc)
    keys = Keyword.keys(fields)
    values = Domain.values(ir)

    Module.create(
      module,
      quote do
        @moduledoc false
        @enforce_keys unquote(keys)
        defstruct unquote(keys)
        @type t :: unquote(Domain.state_typespec(ir))
        def __umpire_state__, do: unquote(Macro.escape(ir))
        # Enumerated once, before the module exists, and compiled in as a literal; the product is
        # at most a few thousand states for any Model this layer is meant for.
        def values, do: unquote(Macro.escape(values))
        def size, do: unquote(length(values))
      end,
      Macro.Env.location(env)
    )

    Module.put_attribute(env.module, :umpire_states, ir)
  end

  @doc "`action :control, party: :caller, on: :activity, input: [control: Control], results: Delivery`"
  defmacro action(name, opts) do
    loc = Diagnostic.loc(__CALLER__)
    Diagnostic.atom!(name, loc, "action name")

    Diagnostic.keys!(opts, [:party, :creates, :on, :input, :results, :schema, :examples], loc, "action #{inspect(name)}")

    quote do
      @umpire_actions Umpire.Lower.action!(unquote(name), unquote(opts), unquote(Macro.escape(loc)))
    end
  end

  @doc "`import_actions Worker, [:workerStop]`: share another Model's action declarations."
  defmacro import_actions(model, names) do
    loc = Diagnostic.loc(__CALLER__)

    quote do
      for action <- Umpire.Model.__imported__(unquote(model), unquote(names), unquote(Macro.escape(loc))) do
        Module.put_attribute(__MODULE__, :umpire_actions, action)
      end
    end
  end

  @doc false
  def __imported__(model, names, loc) do
    declared = Map.new(model.__umpire__(:ir).actions, &{&1.name, &1})

    Enum.map(names, fn name ->
      Map.get(declared, name) ||
        Diagnostic.raise!(loc, "#{inspect(model)} declares no action #{inspect(name)}" <> Diagnostic.suggest(name, Map.keys(declared)))
    end)
  end

  @doc "`observation :attemptCount, on: :activity, read: :attempt`"
  defmacro observation(name, opts) do
    loc = Diagnostic.loc(__CALLER__)
    Diagnostic.atom!(name, loc, "observation name")
    Diagnostic.keys!(opts, [:on, :read], loc, "observation #{inspect(name)}")

    quote do
      @umpire_observations %IR.Observation{
        name: unquote(name),
        on: unquote(opts[:on]),
        read: unquote(opts[:read]),
        loc: unquote(Macro.escape(loc))
      }
    end
  end

  ## Machines

  @doc """
  `defmachine :activityProtocol, for: :activity, state: ProtocolState, ... do ... end`

  Defines the nested module `<Model>.ActivityProtocol` and evaluates the block inside it with
  `Umpire.Machine.DSL` imported. The machine's own `@before_compile` builds and checks its
  table; this Model's hook later reads the result through `__umpire_machine__/0`.
  """
  defmacro defmachine(name, opts, do: block) do
    loc = Diagnostic.loc(__CALLER__)
    Diagnostic.atom!(name, loc, "machine name")
    Diagnostic.keys!(opts, [:for, :state, :outcome, :facts], loc, "defmachine #{inspect(name)}")
    module = Umpire.Names.module(__CALLER__.module, name)

    quote do
      defmodule unquote(module) do
        use Umpire.Machine,
          model: unquote(__CALLER__.module),
          name: unquote(name),
          header: unquote(opts),
          loc: unquote(Macro.escape(loc))

        unquote(block)
      end

      alias unquote(module)
      @umpire_machines unquote(module)
    end
  end

  @doc "`defmachine :activityWorker, from: {Worker, :polling}, only: [:workerStop, :serve]`"
  defmacro defmachine(name, opts) do
    loc = Diagnostic.loc(__CALLER__)
    Diagnostic.atom!(name, loc, "machine name")

    unless Keyword.keyword?(opts) and Keyword.has_key?(opts, :from) and Keyword.has_key?(opts, :only) do
      Diagnostic.raise!(loc, "defmachine #{inspect(name)} without a do block restricts a machine: from: {Model, :machine}, only: [actions]")
    end

    quote do
      @umpire_restrictions %IR.Restriction{
        name: unquote(name),
        from: unquote(opts[:from]),
        only: unquote(opts[:only]),
        loc: unquote(Macro.escape(loc))
      }
    end
  end

  ## Promises

  @doc """
  `defproperty :completes, machine: ..., when: attemptResult(:completed), holds: fn step -> ... end`

  The `holds:` function is lowered to an IR expression and also emitted as the Model function
  `completes/1`, so the type checker sees it.
  """
  defmacro defproperty(name, opts) do
    loc = Diagnostic.loc(__CALLER__)
    Diagnostic.atom!(name, loc, "property name")
    Diagnostic.keys!(opts, [:machine, :when, :holds], loc, "defproperty #{inspect(name)}")
    {params, body} = Lower.fn_shape!(Keyword.fetch!(opts, :holds), loc)
    body = Lower.expand_aliases(body, __CALLER__)
    trigger = opts[:when] && Lower.class_ref!(opts[:when], loc, any_inputs: true)

    if (trigger == nil) != (length(params) == 2) do
      Diagnostic.raise!(loc, """
      defproperty #{inspect(name)}: a same-step claim takes one argument and names its action under when:;
        a transition claim takes two (before, next) and has no when:\
      """)
    end

    quote bind_quoted: [
            name: name,
            machine: opts[:machine],
            trigger: Macro.escape(trigger),
            params: Macro.escape(params),
            body: Macro.escape(body),
            loc: Macro.escape(loc),
            fun: Umpire.Names.fun(name)
          ] do
      @umpire_properties Umpire.Lower.property!(__MODULE__, name, machine, trigger, params, body, loc)

      @doc false
      def unquote(fun)(unquote_splicing(params)), do: unquote(body)
    end
  end

  @doc """
  `defscenario :completed, model: :activityProtocol, starts: :unstarted, actions: [...]`

  The action list is read, never evaluated: `start(:unset, :unset, :unset)` is a class, `backoff`
  a class with no input, `activity.backoff` a composition member's class. Inputs are literals.
  """
  defmacro defscenario(name, opts) do
    loc = Diagnostic.loc(__CALLER__)
    Diagnostic.atom!(name, loc, "scenario name")
    Diagnostic.keys!(opts, [:model, :starts, :actions], loc, "defscenario #{inspect(name)}")
    actions = Enum.map(Keyword.fetch!(opts, :actions), &Lower.class_ref!(&1, loc, any_inputs: false))

    quote do
      @umpire_scenarios %IR.Scenario{
        name: unquote(name),
        model: unquote(opts[:model]),
        starts: unquote(opts[:starts]),
        actions: unquote(Macro.escape(actions)),
        loc: unquote(Macro.escape(loc))
      }
    end
  end

  @doc "`deflimits :three, steps: 3, actions: 3, search: 4096`"
  defmacro deflimits(name, opts) do
    loc = Diagnostic.loc(__CALLER__)
    Diagnostic.keys!(opts, [:steps, :actions, :search], loc, "deflimits #{inspect(name)}")

    for key <- [:steps, :actions, :search], not (is_integer(opts[key]) and opts[key] > 0) do
      Diagnostic.raise!(loc, "deflimits #{inspect(name)}: #{key}: must be a positive integer literal")
    end

    quote do
      @umpire_limits unquote(Macro.escape(struct!(IR.Limits, [name: name, loc: loc] ++ opts)))
    end
  end

  @doc "`defquery :retry, find: :retryCompletes, in: :retriedThenCompleted, limits: :six`"
  defmacro defquery(name, opts) do
    loc = Diagnostic.loc(__CALLER__)
    Diagnostic.keys!(opts, [:find, :verify, :in, :limits, :require_firing], loc, "defquery #{inspect(name)}")

    {kind, property} =
      case {opts[:find], opts[:verify]} do
        {property, nil} when is_atom(property) and property != nil -> {:find, property}
        {nil, property} when is_atom(property) and property != nil -> {:verify, property}
        _ -> Diagnostic.raise!(loc, "defquery #{inspect(name)} needs exactly one of find: or verify:")
      end

    query = %IR.Query{
      name: name,
      kind: kind,
      property: property,
      scenario: opts[:in],
      limits: opts[:limits],
      require_firing: Keyword.get(opts, :require_firing, true),
      loc: loc
    }

    quote do: @umpire_queries(unquote(Macro.escape(query)))
  end

  @doc """
  `defset :standaloneActivityTests, purpose: :functional, bind: [...], queries: [...]`

  Exploratory sets name `machine:`, `cover:` and `budget:` instead of `queries:`.
  """
  defmacro defset(name, opts) do
    loc = Diagnostic.loc(__CALLER__)

    allowed =
      case opts[:purpose] do
        :exploratory -> [:purpose, :bind, :repeat, :machine, :cover, :budget]
        purpose when purpose in [:functional, :canary] -> [:purpose, :bind, :repeat, :queries]
        other -> Diagnostic.raise!(loc, "defset #{inspect(name)}: purpose: #{inspect(other)} is not :functional, :canary or :exploratory")
      end

    Diagnostic.keys!(opts, allowed, loc, "defset #{inspect(name)} (#{opts[:purpose]})")

    for {party, mode} <- opts[:bind] || [], party not in @parties or mode not in [:driven, :observed] do
      Diagnostic.raise!(loc, "defset #{inspect(name)}: bind #{inspect(party)}: #{inspect(mode)} is not a party bound :driven or :observed")
    end

    set = struct!(IR.TestSet, [name: name, loc: loc] ++ opts)
    quote do: @umpire_sets(unquote(Macro.escape(set)))
  end

  @doc """
  `defcompose :standaloneActivity, members: [...], sync: [attemptStart: activity.attemptStart || worker.serve], ...`

  Also defines the composite state struct named by `state:`, one field per member.
  """
  defmacro defcompose(name, opts) do
    loc = Diagnostic.loc(__CALLER__)
    Diagnostic.keys!(opts, [:for, :state, :members, :sync, :starts, :ends], loc, "defcompose #{inspect(name)}")
    short = Diagnostic.alias!(Keyword.fetch!(opts, :state), loc, "defcompose state")
    state_module = Module.concat(__CALLER__.module, short)
    members = Keyword.fetch!(opts, :members)
    sync = Lower.sync!(Keyword.fetch!(opts, :sync), Keyword.keys(members), loc)

    quote do
      alias unquote(state_module)
      Umpire.Model.__composite_state__(unquote(state_module), unquote(Keyword.keys(members)), __ENV__)

      @umpire_compositions %IR.Composition{
        name: unquote(name),
        for: unquote(opts[:for]),
        state: unquote(state_module),
        members: unquote(members),
        sync: unquote(Macro.escape(sync)),
        starts: unquote(opts[:starts]),
        ends: unquote(opts[:ends]),
        loc: unquote(Macro.escape(loc))
      }
    end
  end

  @doc false
  def __composite_state__(module, members, env) do
    Module.create(
      module,
      quote do
        @moduledoc false
        @enforce_keys unquote(members)
        defstruct unquote(members)
        # The composite's fields are member states, so the key and the table enumerate them
        # through the members' own types; see `Umpire.Compose`.
        def __umpire_state__, do: %Umpire.IR.StateType{module: __MODULE__, fields: unquote(Enum.map(members, &{&1, :member}))}
      end,
      Macro.Env.location(env)
    )
  end

  ## The hook

  defmacro __before_compile__(env) do
    model = Umpire.Check.model!(env.module)

    quote do
      @doc false
      def __umpire__(:ir), do: unquote(Macro.escape(model))
      def __umpire__({:table, name}), do: Map.fetch!(unquote(Macro.escape(model.tables)), name)
      def __umpire__({:query, name}), do: Enum.find(unquote(Macro.escape(model.queries)), &(&1.name == name))
    end
  end
end
