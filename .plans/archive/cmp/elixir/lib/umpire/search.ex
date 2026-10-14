defmodule Umpire.Search do
  @moduledoc """
  Bounded search over a table: what a Query answers.

  A scenario fixes the sequence of classes; the search walks every trace of that sequence from
  the scenario's start, branching wherever a row is nondeterministic, and never follows a class
  the scenario does not name. `limits` bound the walk: `steps` is the longest trace, `actions`
  the longest scenario admitted (checked at compile time), `search` the number of steps taken
  before the answer is `:exhausted`.

    * `find` stops at the first step where the claim fires and holds, and returns the trace to it
      as the witness. A claim that fires and fails is not a counterexample to a find; another
      branch may still hold.
    * `verify` checks the claim at every step where it fires, on every trace. A step where it
      fails is a counterexample. A verify whose claim never fired proved nothing, and with
      `require_firing: true` (the default) that is `:vacuous`, an error, rather than a pass.

  A property on another machine is read through the refinement: the view maps each step's state
  with the declared map, and its facts to the target's facts with the same evidence names.

  This walk is written; the exploration for exploratory sets (`explore/2`) is a sketch.
  """

  alias Umpire.{Diagnostic, Domain, Eval, IR, Names, Step, Table}

  defmodule Result do
    @moduledoc """
    `outcome` is one of `:found`, `:not_found`, `:verified_within_limits`, `:violated`,
    `:exhausted`, `:vacuous`. `fired` counts the steps where the claim was evaluated.
    """
    defstruct [:query, :outcome, :witness, :counterexample, fired: 0, nodes: 0, paths: 0]

    @type t :: %__MODULE__{}
  end

  @spec run(module() | IR.Model.t(), atom() | IR.Query.t()) :: Result.t()
  def run(model, query) when is_atom(model), do: run(model.__umpire__(:ir), query)
  def run(%IR.Model{} = model, name) when is_atom(name), do: run(model, Enum.find(model.queries, &(&1.name == name)))

  def run(%IR.Model{} = model, %IR.Query{} = query) do
    property = Enum.find(model.properties, &(&1.name == query.property))
    scenario = Enum.find(model.scenarios, &(&1.name == query.scenario))
    limits = Enum.find(model.limits, &(&1.name == query.limits))
    table = Map.fetch!(model.tables, scenario.model)

    ctx = %{
      query: query,
      property: property,
      limits: limits,
      index: Table.index(table),
      view: view!(property.machine, scenario.model, model, query.loc)
    }

    root = %Step{outcome: nil, state: start(scenario, table), facts: []}
    classes = Enum.map(scenario.actions, &class_of/1)

    result =
      try do
        walk(ctx, root, [], classes, %Result{query: query.name})
      catch
        {:stop, %Result{} = result} -> result
      end

    finish(result, query)
  end

  defp finish(%Result{outcome: outcome} = result, _query) when outcome != nil, do: result
  defp finish(result, %IR.Query{kind: :find}), do: %{result | outcome: :not_found}
  defp finish(%Result{fired: 0} = result, %IR.Query{kind: :verify, require_firing: true}), do: %{result | outcome: :vacuous}
  defp finish(result, %IR.Query{kind: :verify}), do: %{result | outcome: :verified_within_limits}

  defp walk(_ctx, _last, _trace, [], acc), do: %{acc | paths: acc.paths + 1}

  defp walk(ctx, last, trace, [class | rest], acc) do
    if length(trace) >= ctx.limits.steps do
      %{acc | paths: acc.paths + 1}
    else
      ctx.index
      |> Map.get({last.state, class}, [])
      |> Enum.reduce(acc, fn row, acc ->
        acc = %{acc | nodes: acc.nodes + 1}
        if acc.nodes > ctx.limits.search, do: throw({:stop, %{acc | outcome: :exhausted}})

        step = %Step{outcome: row.outcome, state: row.to, facts: row.facts}
        trace = [{class, step} | trace]
        acc = claim(ctx, class, last, step, trace, acc)
        walk(ctx, step, trace, rest, acc)
      end)
    end
  end

  # Evaluate the claim where it fires. A find stops on the first hold; a verify on the first fail.
  defp claim(ctx, class, last, step, trace, acc) do
    %{property: property, query: query, view: view} = ctx

    {fires?, args} =
      case property.kind do
        :same_step -> {fires?(property.when, class), [view.(step)]}
        :transition -> {true, [view.(last), view.(step)]}
      end

    if fires? do
      acc = %{acc | fired: acc.fired + 1}
      holds? = Eval.holds?(property, args)

      case {query.kind, holds?} do
        {:find, true} -> throw({:stop, %{acc | outcome: :found, witness: Enum.reverse(trace)}})
        {:verify, false} -> throw({:stop, %{acc | outcome: :violated, counterexample: Enum.reverse(trace)}})
        # Kept for the message if no branch holds: the first trace where the claim fired and failed.
        {:find, false} -> %{acc | counterexample: acc.counterexample || Enum.reverse(trace)}
        {:verify, true} -> acc
      end
    else
      acc
    end
  end

  defp fires?(%IR.ClassRef{} = ref, {action, inputs}) do
    {ref_action, _} = class_of(%{ref | inputs: []})
    ref_action == action and (ref.inputs == :any or ref.inputs == inputs)
  end

  defp class_of(%IR.ClassRef{member: nil, action: action, inputs: inputs}), do: {action, inputs}
  defp class_of(%IR.ClassRef{member: member, action: action, inputs: inputs}), do: {{member, action}, inputs}

  defp start(%IR.Scenario{starts: phase}, table) when is_atom(phase),
    do: Enum.find(table.starts ++ table.states, &(&1.phase == phase))

  # A composition scenario names the members whose start it fixes; the rest start where they do.
  defp start(%IR.Scenario{starts: starts}, table) do
    Enum.find(table.starts ++ table.states, fn state ->
      Enum.all?(starts, fn {member, phase} -> Map.fetch!(state, member).phase == phase end)
    end)
  end

  @doc """
  How a step of `scenario_machine` reads as a step of `property_machine`: the identity, or the
  refinement map composed along `refines` until it reaches the property's machine. Raises at
  the query when there is no such chain, which is the admission check `Umpire.Check` calls.
  """
  @spec view!(atom(), atom(), IR.Model.t(), Diagnostic.t()) :: (Step.t() -> Step.t())
  def view!(same, same, _model, _loc), do: & &1

  def view!(property_machine, scenario_machine, model, loc) do
    with %IR.Machine{refines: {target, map_name, _rule}} = machine <- Enum.find(model.machines, &(&1.name == scenario_machine)) do
      target_ir = Enum.find(model.machines, &(&1.name == target))
      abstraction = Map.fetch!(machine.maps, map_name)
      step_view = fn step -> project(step, abstraction, machine, target_ir) end
      rest = view!(property_machine, target, model, loc)
      fn step -> step |> step_view.() |> rest.() end
    else
      _ ->
        Diagnostic.raise!(loc, """
        a property on #{inspect(property_machine)} cannot be read on #{inspect(scenario_machine)}:
          #{inspect(scenario_machine)} declares no refinement that reaches it\
        """)
    end
  end

  defp project(%Step{} = step, abstraction, machine, target) do
    {:ok, _clause, state} = Eval.map(abstraction, step.state)
    names = Umpire.Refinement.names(step.facts, machine.evidence)

    facts =
      for fact <- target.facts.values(),
          Keyword.get(target.evidence, Domain.ctor(fact), Domain.ctor(fact)) in names,
          do: fact

    %{step | state: state, facts: facts}
  end

  @doc """
  Sketch: the exploratory set's coverage targets. Breadth-first over every class from the starts,
  to `budget.steps` deep, cut at `budget.search` steps, in catalog order so the enumeration is the
  same on every run. Returns the rows reached, the results (end states) reached, and the input
  class members exercised; a golden test pins the three lists.
  """
  @spec explore(Table.t(), IR.Limits.t()) :: %{rows: list(), results: list(), classMembers: list()}
  def explore(%Table{} = table, %IR.Limits{} = budget) do
    index = Table.index(table)

    {rows, _frontier, _count} =
      Enum.reduce_while(1..budget.steps, {[], table.starts, 0}, fn _depth, {seen, frontier, count} ->
        next = for state <- frontier, class <- table.classes, row <- Map.get(index, {state, class}, []), do: row
        count = count + length(next)
        rows = Enum.uniq(seen ++ next)
        if count >= budget.search, do: {:halt, {rows, [], count}}, else: {:cont, {rows, Enum.uniq(Enum.map(next, & &1.to)), count}}
      end)

    %{
      rows: Enum.map(rows, &{Names.key(&1.from), Names.key(&1.class)}),
      results: rows |> Enum.map(& &1.to) |> Enum.filter(&(&1 in table.ends)) |> Enum.uniq(),
      classMembers: rows |> Enum.map(& &1.class) |> Enum.uniq()
    }
  end

  @doc "A failed Query as an author reads it in `mix test` output."
  @spec format(Result.t(), IR.Model.t()) :: String.t()
  def format(%Result{} = result, %IR.Model{} = model) do
    query = Enum.find(model.queries, &(&1.name == result.query))
    stats = "#{result.nodes} steps searched, #{result.paths} paths, the claim fired #{result.fired} times"

    headline =
      case result.outcome do
        :not_found -> "query #{inspect(query.name)} did not find #{inspect(query.property)} in #{inspect(query.scenario)} within #{inspect(query.limits)}"
        :violated -> "query #{inspect(query.name)}: #{inspect(query.property)} fails on a trace of #{inspect(query.scenario)}"
        :exhausted -> "query #{inspect(query.name)} ran out of search budget (#{inspect(query.limits)}) before an answer"
        :vacuous -> "query #{inspect(query.name)} is vacuous: #{inspect(query.property)} never fired on #{inspect(query.scenario)}; a verify whose claim never fires proves nothing (add the action its when: names to the path, or pass require_firing: false)"
        outcome -> "query #{inspect(query.name)}: #{outcome}"
      end

    trace =
      (result.counterexample || result.witness || [])
      |> Enum.with_index(1)
      |> Enum.map_join("\n", fn {{class, step}, n} -> "    #{n}. #{Names.key(class)} -> #{inspect(step.state)} #{inspect(step.facts)}" end)

    Enum.join([headline, "  " <> stats, trace], "\n")
  end
end
