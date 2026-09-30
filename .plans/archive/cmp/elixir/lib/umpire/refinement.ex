defmodule Umpire.Refinement do
  @moduledoc """
  `refines :activityProduct, map: :productOf`: every protocol row is a product row between the
  mapped states, or a stutter.

  For each protocol row `(s, class, step)`, with `m = productOf`:

    * if `m(s) == m(step.state)` the row is a **stutter**: the product does not see it;
    * otherwise some product row must go from `m(s)` to `m(step.state)`. Rows are matched by
      mapped states, not by action name: the protocol's `scheduleToStart` row matches the
      product's `timeout` row, and a retryable failure under a requested pause matches the
      product's `control(:pause)`.

  Two rules decide whether a product row between the right states matches:

    * `:mapped_states`, the rule SPEC.md states: any such row matches;
    * `:strict`, the rule the Lean checker (`Umpire/Command/Refinement.lean`) implements and the
      default here: the product row must also have the same outcome, and each of its facts must be
      among the protocol row's facts, compared by evidence name. That is why
      `{:statusTimedOut, :scheduleToStart}` matches `:statusTimedOut`, and why the protocol row
      for `attemptResult({:failed, true})` from `:started` records `:statusScheduled` as well as
      `:attemptCount`: without it, the product's retry row would record a fact the protocol row
      does not.

  The check runs in the protocol machine's `@before_compile`, after the product machine's module
  has compiled, so a Model whose refinement fails does not compile. The map is checked first for
  totality and codomain: every protocol state maps, and to a state of the product.
  """

  alias Umpire.{Diagnostic, Domain, Eval, IR, Names, Table}

  defmodule Result do
    @moduledoc """
    One verdict per protocol row: `:stutter`, `{:matches, product_row}`. `rejected` is the first
    row with neither, or nil.
    """
    defstruct [:machine, :target, :rule, rows: [], rejected: nil]
  end

  @doc "Check the refinement and raise at the `defmap` line when it fails."
  @spec check!(IR.Machine.t(), Table.t(), module()) :: Result.t()
  def check!(%IR.Machine{refines: {target, map_name, rule}} = machine, table, module) do
    target_module = target_module!(module, target, machine.loc)
    product = target_module.__umpire_machine__()
    product_table = target_module.table()
    abstraction = Map.fetch!(machine.maps, map_name)

    mapped = Map.new(table.states, &{&1, map!(abstraction, &1, product)})
    result = check(table, product_table, mapped, rule, {machine.evidence, product.evidence})
    result = %{result | machine: machine.name, target: target}

    case result.rejected do
      nil ->
        result

      {row, from, to} ->
        Diagnostic.raise!(abstraction.loc, """
        #{inspect(machine.name)} does not refine #{inspect(target)} (rule #{inspect(rule)}):
            the row #{Names.key(row.from)} --#{Names.key(row.class)}--> #{Names.key(row.to)}
            maps #{Names.key(from)} --> #{Names.key(to)}, which is not a stutter,
            and #{inspect(target)} has no row between them#{nearest(product_table, from, to, rule)}\
        """)
    end
  end

  @doc "The check itself, on tables and a precomputed map; pure, so tests can call it directly."
  @spec check(Table.t(), Table.t(), %{struct() => struct()}, :strict | :mapped_states, {keyword(), keyword()}) :: Result.t()
  def check(table, product_table, mapped, rule, evidence) do
    between = Enum.group_by(product_table.rows, &{&1.from, &1.to})

    verdicts =
      Enum.map(table.rows, fn row ->
        {from, to} = {mapped[row.from], mapped[row.to]}

        cond do
          from == to -> {row, :stutter}
          match = Enum.find(Map.get(between, {from, to}, []), &matches?(&1, row, rule, evidence)) -> {row, {:matches, match}}
          true -> {row, {:rejected, from, to}}
        end
      end)

    rejected = Enum.find_value(verdicts, fn {row, verdict} -> match?({:rejected, _, _}, verdict) && {row, elem(verdict, 1), elem(verdict, 2)} end)
    %Result{rule: rule, rows: verdicts, rejected: rejected}
  end

  defp matches?(_product_row, _row, :mapped_states, _evidence), do: true

  defp matches?(product_row, row, :strict, {evidence, product_evidence}) do
    product_row.outcome == row.outcome and
      MapSet.subset?(names(product_row.facts, product_evidence), names(row.facts, evidence))
  end

  @doc "The evidence names of a list of facts: a constructor's evidence line, else its own name."
  def names(facts, evidence), do: MapSet.new(facts, &Keyword.get(evidence, Domain.ctor(&1), Domain.ctor(&1)))

  defp map!(abstraction, state, product) do
    case Eval.map(abstraction, state) do
      {:ok, _index, mapped} ->
        if mapped in product.state.values(),
          do: mapped,
          else: Diagnostic.raise!(abstraction.loc, "defmap #{abstraction.name} maps #{inspect(state)} to #{inspect(mapped)}, which is not a state of #{inspect(product.name)}")

      {:fell_through, subject} ->
        Diagnostic.raise!(abstraction.loc, "defmap #{abstraction.name} has no clause for #{inspect(subject)}; a map is total")
    end
  end

  defp nearest(product_table, from, to, :strict) do
    case Enum.filter(product_table.rows, &(&1.from == from and &1.to == to)) do
      [] -> ""
      rows -> "\n    (#{length(rows)} product rows go there, but differ in outcome or record a fact this row does not)"
    end
  end

  defp nearest(_product_table, _from, _to, _rule), do: ""

  defp target_module!(module, target, loc) do
    model = Module.get_attribute(module, :umpire_model)

    Enum.find(Module.get_attribute(model, :umpire_machines), &(&1.__umpire_machine__().name == target)) ||
      Diagnostic.raise!(loc, "refines #{inspect(target)}: no machine of that name is declared above this one")
  end
end
