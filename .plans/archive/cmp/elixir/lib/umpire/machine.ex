defmodule Umpire.Machine do
  @moduledoc """
  The inside of a `defmachine` block: a nested module with its own attributes, its step
  functions, and a `@before_compile` hook that turns them into a checked `Umpire.IR.Machine`.

  The hook is where the Model-specific checks happen for one machine, in this order:

    1. names: every `defstep` names a declared action or one of this machine's timers, every timer
       has a step, `unobservable` is a subset of `timers`, `starts`/`ends` are phases, evidence
       keys are fact constructors (`Umpire.Check.machine!/1`);
    2. the table: every step evaluated on every state and every class of its action, which is
       also the exhaustiveness and dead-arm check, and the check that every produced state, fact
       and outcome is in its domain (`Umpire.Table.build!/1`);
    3. the refinement, if the machine declares one (`Umpire.Refinement.check!/3`).

  The result is compiled into the module as data: `__umpire_machine__/0`, `table/0` and
  `refinement/0`, next to the step functions themselves.
  """

  defmacro __using__(opts) do
    quote do
      import Umpire.Machine.DSL

      @umpire_model unquote(opts[:model])
      @umpire_name unquote(opts[:name])
      @umpire_header unquote(opts[:header])
      @umpire_loc unquote(opts[:loc])

      Module.register_attribute(__MODULE__, :umpire_steps, accumulate: true)
      Module.register_attribute(__MODULE__, :umpire_maps, accumulate: true)

      @before_compile Umpire.Machine
    end
  end

  defmacro __before_compile__(env) do
    machine = Umpire.Check.machine!(env.module)
    table = Umpire.Table.build!(machine)
    refinement = machine.refines && Umpire.Refinement.check!(machine, table, env.module)

    quote do
      @doc false
      def __umpire_machine__, do: unquote(Macro.escape(machine))

      @doc "The finite table: every state, every class, every enabled row."
      def table, do: unquote(Macro.escape(table))

      @doc "The refinement rows, or nil when this machine refines nothing."
      def refinement, do: unquote(Macro.escape(refinement))
    end
  end

  @doc false
  # Stores a machine header line once; a second `starts` is an error at its own line.
  def __put_once__(module, key, value, loc) do
    if Module.has_attribute?(module, key) do
      Umpire.Diagnostic.raise!(loc, "#{key |> Atom.to_string() |> String.trim_leading("umpire_")} is declared twice in this machine")
    end

    Module.put_attribute(module, key, value)
  end
end

defmodule Umpire.Machine.DSL do
  @moduledoc """
  The macros available inside `defmachine ... do ... end`.

  Header lines (`starts`, `ends`, `timers`, `unobservable`, `evidence`, `refines`) take literal
  atoms, keyword lists or `@attributes`; an attribute is looked up in this machine's module
  first and then in the Model's, so `ends @productTerminal` reads the Model's list.

  `defstep` and `defmap` read like `def`: a head naming the action and binding the state and
  inputs, and a body that is one `case` over state fields and inputs. The body is lowered by
  `Umpire.Lower.step!/5` and emitted as a real function.

  Inside a body, the step constructors are:

    * `[]`: not enabled here;
    * `moves(phase, facts)` and `moves(phase, facts, field: value, ...)`: accepted, the state
      with that phase and those fields, those facts;
    * `stay()`: accepted, the state unchanged, no facts (a stutter);
    * `not_found()`: outcome `:notFound`, the state unchanged, no facts;
    * `succ(state.attempts)`: the saturating successor, bounded by the field's range.

  They exist only as IR; `Umpire.Lower` rewrites each into the `%Umpire.Step{}` list it stands
  for, so there are no `moves/2` functions to call at run time.
  """

  alias Umpire.Diagnostic

  for {name, key} <- [starts: :umpire_starts, ends: :umpire_ends, timers: :umpire_timers, unobservable: :umpire_unobservable, evidence: :umpire_evidence] do
    @doc "Machine header line `#{name}`."
    defmacro unquote(name)(value) do
      header(unquote(key), value, __CALLER__)
    end
  end

  @doc "`refines :activityProduct, map: :productOf` (optionally `rule: :mapped_states`)."
  defmacro refines(target, opts) do
    loc = Diagnostic.loc(__CALLER__)
    Diagnostic.keys!(opts, [:map, :rule], loc, "refines")
    rule = Keyword.get(opts, :rule, :strict)

    unless rule in [:strict, :mapped_states] do
      Diagnostic.raise!(loc, "refines rule: must be :strict or :mapped_states, got #{inspect(rule)}")
    end

    quote do
      Umpire.Machine.__put_once__(
        __MODULE__,
        :umpire_refines,
        {unquote(target), unquote(opts[:map]), unquote(rule)},
        unquote(Macro.escape(loc))
      )
    end
  end

  defp header(key, value, caller) do
    loc = Diagnostic.loc(caller)
    Umpire.Lower.header_shape!(value, loc)

    quote do
      Umpire.Machine.__put_once__(
        __MODULE__,
        unquote(key),
        Umpire.Lower.resolve!(unquote(Macro.escape(value)), __MODULE__, unquote(Macro.escape(loc))),
        unquote(Macro.escape(loc))
      )
    end
  end

  @doc """
  `defstep attemptResult(state, result) do case {state.phase, result} do ... end end`

  The head names the action; its first argument binds the state and the rest bind the action's
  inputs in declaration order (a name may be the input's own or its snake_case form).
  """
  defmacro defstep(head, do: body) do
    loc = Diagnostic.loc(__CALLER__)
    {action, state_var, input_vars} = Umpire.Lower.head_shape!(head, loc, "defstep")
    body = Umpire.Lower.expand_aliases(body, __CALLER__)

    quote bind_quoted: [
            action: action,
            state_var: state_var,
            input_vars: input_vars,
            body: Macro.escape(body),
            loc: Macro.escape(loc)
          ] do
      {step, code} = Umpire.Lower.step!(__MODULE__, action, {state_var, input_vars}, body, loc)
      @umpire_steps step

      @doc false
      def unquote(step.fun)(unquote_splicing(code.params)) when unquote(code.guard), do: unquote(code.body)
    end
  end

  @doc """
  `defmap productOf(state) do case state.phase do ... end end`

  An abstraction function to another machine's state. Its bodies are struct literals of the
  target state; totality and codomain are checked when the refinement is.
  """
  defmacro defmap(head, do: body) do
    loc = Diagnostic.loc(__CALLER__)
    {name, state_var, inputs} = Umpire.Lower.head_shape!(head, loc, "defmap")
    if inputs != [], do: Diagnostic.raise!(loc, "defmap #{name} takes the state and nothing else")
    body = Umpire.Lower.expand_aliases(body, __CALLER__)

    quote bind_quoted: [name: name, state_var: state_var, body: Macro.escape(body), loc: Macro.escape(loc)] do
      {abstraction, code} = Umpire.Lower.map!(__MODULE__, name, state_var, body, loc)
      @umpire_maps abstraction

      @doc false
      def unquote(abstraction.fun)(unquote_splicing(code.params)), do: unquote(code.body)
    end
  end
end
