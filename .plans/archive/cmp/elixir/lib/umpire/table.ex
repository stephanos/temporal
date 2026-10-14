defmodule Umpire.Table do
  @moduledoc """
  The finite table of a machine: every state, every action class, every enabled row.

  `build!/1` runs in the machine module's `@before_compile`, so a table exists for every Model
  that compiles. Building it is also the Model-specific half of type checking, because it
  evaluates every step on every state and every class of its action:

    * **fall-through**: a (state, class) pair no clause matched. Elixir would raise
      `CaseClauseError` there at run time, and the type checker does not report missing clauses.
      Here it is a `CompileError` listing the uncovered subject values, since every step must say
      `[]` where it is not enabled;
    * **dead arm**: a clause no (state, class) pair reached first. Usually an ordering mistake or
      a typo in a literal, which a pattern over atoms cannot otherwise reveal;
    * **out-of-domain values**: a target phase, an updated field, a fact or an outcome that is not
      a member of its declared domain.

  The table is tiny for the Models it is meant for: 288 protocol states times 22 classes for the
  standalone activity, evaluated in milliseconds by `Umpire.Eval`.
  """

  alias Umpire.{Diagnostic, Domain, Eval, IR, Names}

  defmodule Row do
    @moduledoc "One enabled transition: from a state, under a class, to a step, by a clause."
    defstruct [:from, :class, :outcome, :to, :facts, :clause]
  end

  defstruct [:machine, :states, :classes, :rows, :starts, :ends]

  @type t :: %__MODULE__{}

  @doc "Every action class a machine steps on, in canonical (key) order."
  @spec classes(IR.Machine.t(), %{atom() => IR.Action.t()}) :: [Umpire.class()]
  def classes(%IR.Machine{steps: steps}, actions) do
    steps
    |> Enum.flat_map(fn {action, _step} ->
      inputs = if a = actions[action], do: Enum.map(a.input, fn {_f, t} -> Domain.values(t) end), else: []
      for assignment <- Domain.product(inputs), do: {action, assignment}
    end)
    |> Enum.sort_by(&Names.key/1)
  end

  @doc "Build and check the table. Raises `CompileError` at the offending step or clause."
  @spec build!(IR.Machine.t()) :: t()
  def build!(%IR.Machine{} = machine) do
    actions = machine.module |> Module.get_attribute(:umpire_model) |> Module.get_attribute(:umpire_actions) |> Map.new(&{&1.name, &1})
    states = machine.state.values()
    classes = classes(machine, actions)
    domains = %{state: MapSet.new(states), facts: MapSet.new(machine.facts.values()), outcome: machine.outcome.values()}

    {rows, hits, misses} =
      for state <- states, {action, inputs} = class <- classes, reduce: {[], %{}, %{}} do
        {rows, hits, misses} ->
          step = Map.fetch!(machine.steps, action)

          case Eval.step(step, state, inputs, machine) do
            {:fell_through, subject} ->
              {rows, hits, Map.update(misses, action, MapSet.new([subject]), &MapSet.put(&1, subject))}

            {:ok, index, steps} ->
              Enum.each(steps, &in_domain!(&1, step, index, domains))
              new = Enum.map(steps, &%Row{from: state, class: class, outcome: &1.outcome, to: &1.state, facts: &1.facts, clause: index})
              {Enum.reverse(new, rows), Map.update(hits, action, MapSet.new([index]), &MapSet.put(&1, index)), misses}
          end
      end

    Enum.each(machine.steps, fn {action, step} -> coverage!(step, Map.get(hits, action, MapSet.new()), Map.get(misses, action, MapSet.new())) end)

    %__MODULE__{
      machine: machine.name,
      states: states,
      classes: classes,
      rows: Enum.reverse(rows),
      starts: Enum.map(machine.starts, &Domain.start(machine.state, &1)),
      ends: Enum.filter(states, &(&1.phase in machine.ends))
    }
  end

  defp coverage!(%IR.StepFn{} = step, hits, misses) do
    if MapSet.size(misses) > 0 do
      shown = misses |> Enum.sort() |> Enum.take(12)

      Diagnostic.raise!(step.loc, """
      #{describe(step)} has no clause for #{MapSet.size(misses)} of its #{subjects(step)} values:
          #{Enum.map_join(shown, "\n    ", &inspect/1)}#{if MapSet.size(misses) > 12, do: "\n    ...", else: ""}
        Every combination needs a clause; where the action is not enabled the clause returns [].\
      """)
    end

    for {clause, index} <- Enum.with_index(step.clauses), not MapSet.member?(hits, index) do
      Diagnostic.raise!(clause.loc, """
      this clause of #{describe(step)} never matches first: every #{subjects(step)} value it
        matches is taken by an earlier clause, or it matches none (check its literals against
        their domains).\
      """)
    end
  end

  defp describe(step), do: "defstep #{step.action}/#{length(step.inputs) + 1}"

  defp subjects(%IR.StepFn{subjects: subjects}) do
    names = Enum.map(subjects, fn {_kind, name} -> name end)
    if length(names) == 1, do: "#{hd(names)}", else: "{#{Enum.join(names, ", ")}}"
  end

  defp in_domain!(step_result, step, index, domains) do
    clause = Enum.at(step.clauses, index)

    cond do
      not MapSet.member?(domains.state, step_result.state) ->
        Diagnostic.raise!(clause.loc, "#{describe(step)} produces #{inspect(step_result.state)}, which is not a state of its machine")

      step_result.outcome not in domains.outcome ->
        Diagnostic.raise!(clause.loc, "#{describe(step)} produces outcome #{inspect(step_result.outcome)}, not in #{inspect(domains.outcome)}")

      bad = Enum.find(step_result.facts, &(not MapSet.member?(domains.facts, &1))) ->
        Diagnostic.raise!(clause.loc, "#{describe(step)} records #{inspect(bad)}, which is not a member of the machine's facts")

      true ->
        :ok
    end
  end

  @doc "Rows grouped by `{from, class}`: what the search and the composition index."
  @spec index(t()) :: %{{struct(), Umpire.class()} => [Row.t()]}
  def index(%__MODULE__{rows: rows}), do: Enum.group_by(rows, &{&1.from, &1.class})

  @doc "The states reachable from the starts."
  @spec reachable(t()) :: [struct()]
  def reachable(%__MODULE__{starts: starts, rows: rows}) do
    successors = Enum.group_by(rows, & &1.from, & &1.to)
    walk(starts, MapSet.new(starts), successors) |> MapSet.to_list()
  end

  defp walk([], seen, _successors), do: seen

  defp walk([state | rest], seen, successors) do
    new = successors |> Map.get(state, []) |> Enum.reject(&MapSet.member?(seen, &1)) |> Enum.uniq()
    walk(new ++ rest, Enum.into(new, seen), successors)
  end

  @doc "A reachable non-end state with no enabled row, if any: a machine that can get stuck."
  @spec stuck(t()) :: struct() | nil
  def stuck(%__MODULE__{} = table) do
    enabled = MapSet.new(table.rows, & &1.from)
    Enum.find(reachable(table), &(not MapSet.member?(enabled, &1) and &1 not in table.ends))
  end

  @doc "The same table with only the rows of `only`'s actions: `defmachine name, from:, only:`."
  @spec restrict(t(), atom(), [atom()]) :: t()
  def restrict(%__MODULE__{} = table, name, only) do
    keep? = fn {action, _inputs} -> action in only end
    %{table | machine: name, classes: Enum.filter(table.classes, keep?), rows: Enum.filter(table.rows, &keep?.(&1.class))}
  end
end
