defmodule Umpire.Compose do
  @moduledoc """
  The product of machines of different entities.

  A composition's states are every combination of its members' states. Its classes are:

    * one class per `sync:` line and per assignment of the two member actions' inputs, named by
      the line: `attemptStart: activity.attemptStart || worker.serve` is the class
      `{:attemptStart, []}`, enabled where both members have a row, and it moves both;
    * every member action no sync line names, qualified by member: `{{:activity, :backoff}, []}`,
      which moves that member alone.

  A restricted member (`only: [:workerStop, :serve]`) contributes only the named actions, which
  is how the worker's `workerResume` is kept out: an unsynchronized member action stays
  executable on its own.

  The composite table is built in the Model's `@before_compile` from the member tables, so it is
  data like any other table and the search reads it the same way.
  """

  alias Umpire.{Diagnostic, Domain, Names, Step, Table}

  @spec table!(Umpire.IR.Composition.t(), %{atom() => Table.t()}) :: Table.t()
  def table!(composition, member_tables) do
    members = Keyword.keys(composition.members)
    tables = Map.new(composition.members, fn {member, machine} -> {member, Map.fetch!(member_tables, machine)} end)
    indexes = Map.new(tables, fn {member, table} -> {member, Table.index(table)} end)

    states =
      members
      |> Enum.map(&tables[&1].states)
      |> Domain.product()
      |> Enum.map(&struct!(composition.state, Enum.zip(members, &1)))

    synced = for {_name, pairs} <- composition.sync, pair <- pairs, into: MapSet.new(), do: pair
    classes = sync_classes(composition, tables) ++ solo_classes(members, tables, synced)

    rows =
      for state <- states, class <- classes, step <- fire(class, state, indexes) do
        %Table.Row{from: state, class: class_key(class), outcome: step.outcome, to: step.state, facts: step.facts}
      end

    %Table{
      machine: composition.name,
      states: states,
      classes: Enum.map(classes, &class_key/1),
      rows: rows,
      starts: [start(composition, tables)],
      ends: Enum.filter(states, &ended?(&1, composition.ends))
    }
  end

  # {name, [{member, class}, {member, class}]}: one per assignment of both sides' inputs.
  defp sync_classes(composition, tables) do
    for {name, [{m1, a1}, {m2, a2}]} <- composition.sync,
        {^a1, _} = c1 <- tables[m1].classes,
        {^a2, _} = c2 <- tables[m2].classes,
        do: {:sync, name, [{m1, c1}, {m2, c2}]}
  end

  defp solo_classes(members, tables, synced) do
    for member <- members, {action, _} = class <- tables[member].classes, {member, action} not in synced,
        do: {:solo, member, class}
  end

  # The class as the scenario spells it: `attemptStart` or `activity.backoff`.
  defp class_key({:sync, name, [{_, {_, i1}}, {_, {_, i2}}]}), do: {name, i1 ++ i2}
  defp class_key({:solo, member, {action, inputs}}), do: {{member, action}, inputs}

  defp fire({:solo, member, class}, state, indexes) do
    for row <- Map.get(indexes[member], {Map.fetch!(state, member), class}, []),
        do: %Step{outcome: row.outcome, state: Map.put(state, member, row.to), facts: Enum.map(row.facts, &{member, &1})}
  end

  defp fire({:sync, _name, [{m1, c1}, {m2, c2}]}, state, indexes) do
    for r1 <- Map.get(indexes[m1], {Map.fetch!(state, m1), c1}, []),
        r2 <- Map.get(indexes[m2], {Map.fetch!(state, m2), c2}, []) do
      %Step{
        outcome: r1.outcome,
        state: state |> Map.put(m1, r1.to) |> Map.put(m2, r2.to),
        facts: Enum.map(r1.facts, &{m1, &1}) ++ Enum.map(r2.facts, &{m2, &1})
      }
    end
  end

  defp start(composition, tables) do
    fields =
      Enum.map(composition.members, fn {member, _machine} ->
        table = tables[member]

        state =
          case Keyword.fetch(composition.starts, member) do
            {:ok, phase} -> Enum.find(table.starts ++ table.states, &(&1.phase == phase))
            :error -> hd(table.starts)
          end

        state || Diagnostic.raise!(composition.loc, "defcompose #{composition.name}: #{member} has no state in phase #{inspect(composition.starts[member])}")
        {member, state}
      end)

    struct!(composition.state, fields)
  end

  defp ended?(state, ends), do: Enum.all?(ends, fn {member, phases} -> Map.fetch!(state, member).phase in phases end)

  @doc "A composite state's key, for messages."
  def key(state), do: Names.key(state)
end
