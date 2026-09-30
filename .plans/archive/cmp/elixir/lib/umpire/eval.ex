defmodule Umpire.Eval do
  @moduledoc """
  The IR interpreter: what a step, a map or a predicate means, computed from data.

  The table, the coverage check, the refinement and the search read the IR through this module,
  at compile time and in tests, and never call the emitted functions. The emitted functions are
  the same clauses compiled by the BEAM; `Umpire.Case.assert_agrees/2` compares the two on every
  row, so a disagreement between this matcher and Elixir's is a failing test rather than a silent
  divergence.
  """

  alias Umpire.{IR, Step}

  @typedoc "Which clause answered, by index, and the steps it produced; or no clause at all."
  @type answer :: {:ok, non_neg_integer(), [Step.t()]} | {:fell_through, term()}

  @doc "Evaluate a step on one state and one assignment of its inputs."
  @spec step(IR.StepFn.t(), struct(), [term()], IR.Machine.t()) :: answer()
  def step(%IR.StepFn{} = step, state, inputs, machine) do
    env = %{state: state, inputs: step.inputs |> Keyword.keys() |> Enum.zip(inputs) |> Map.new(), vars: %{}}

    case first_clause(step.subjects, step.clauses, env) do
      {index, clause, env} -> {:ok, index, body(clause.body, env, machine)}
      nil -> {:fell_through, subject_value(step.subjects, env)}
    end
  end

  @doc "Evaluate a map on one state."
  @spec map(IR.Abstraction.t(), struct()) :: {:ok, non_neg_integer(), struct()} | {:fell_through, term()}
  def map(%IR.Abstraction{} = abstraction, state) do
    env = %{state: state, inputs: %{}, vars: %{}}

    case first_clause(abstraction.subjects, abstraction.clauses, env) do
      {index, %IR.Clause{body: {:state, module, fields}}, env} ->
        {:ok, index, struct!(module, Enum.map(fields, fn {f, e} -> {f, expr(e, env)} end))}

      nil ->
        {:fell_through, subject_value(abstraction.subjects, env)}
    end
  end

  @doc "Evaluate a property predicate on its arguments (one step, or before and next)."
  @spec holds?(IR.Property.t(), [Step.t()]) :: boolean()
  def holds?(%IR.Property{holds: holds}, args), do: expr(holds, %{args: List.to_tuple(args)}) == true

  defp first_clause(subjects, clauses, env) do
    value = subject_value(subjects, env)

    clauses
    |> Enum.with_index()
    |> Enum.find_value(fn {clause, index} ->
      with {:ok, vars} <- match(clause.pattern, value, %{}),
           env = %{env | vars: vars},
           true <- expr(clause.guard, env) do
        {index, clause, env}
      else
        _ -> nil
      end
    end)
  end

  # One subject is matched as itself; several as the tuple of their values, as `case` does.
  defp subject_value([], _env), do: nil
  defp subject_value([one], env), do: expr(one, env)
  defp subject_value(many, env), do: many |> Enum.map(&expr(&1, env)) |> List.to_tuple()

  @doc false
  def match(:any, _value, vars), do: {:ok, vars}
  def match({:bind, name}, value, vars), do: {:ok, Map.put(vars, name, value)}
  def match({:lit, literal}, value, vars), do: if(literal === value, do: {:ok, vars}, else: :no_match)

  def match({:tuple, patterns}, value, vars) when is_tuple(value) and tuple_size(value) == length(patterns) do
    patterns
    |> Enum.zip(Tuple.to_list(value))
    |> Enum.reduce_while({:ok, vars}, fn {pattern, element}, {:ok, vars} ->
      case match(pattern, element, vars) do
        {:ok, vars} -> {:cont, {:ok, vars}}
        :no_match -> {:halt, :no_match}
      end
    end)
  end

  def match(_pattern, _value, _vars), do: :no_match

  defp body(:disabled, _env, _machine), do: []
  defp body(:stay, env, _machine), do: [%Step{outcome: :accepted, state: env.state, facts: []}]
  defp body(:not_found, env, _machine), do: [%Step{outcome: :notFound, state: env.state, facts: []}]

  defp body({:moves, phase, facts, updates}, env, _machine) do
    fields = [{:phase, expr(phase, env)} | Enum.map(updates, fn {f, e} -> {f, expr(e, env)} end)]
    [%Step{outcome: :accepted, state: struct!(env.state, fields), facts: Enum.map(facts, &expr(&1, env))}]
  end

  @doc false
  def expr({:lit, value}, _env), do: value
  def expr({:var, name}, env), do: Map.fetch!(env.vars, name)
  def expr({:input, field}, env), do: Map.fetch!(env.inputs, field)
  def expr({:field, field}, env), do: Map.fetch!(env.state, field)
  def expr({:succ, inner, max}, env), do: min(expr(inner, env) + 1, max)
  def expr({:tuple, elements}, env), do: elements |> Enum.map(&expr(&1, env)) |> List.to_tuple()
  def expr({:path, index, path}, env), do: Enum.reduce(path, elem(env.args, index), &Map.fetch!(&2, &1))
  def expr({:eq, left, right}, env), do: expr(left, env) === expr(right, env)
  def expr({:neq, left, right}, env), do: expr(left, env) !== expr(right, env)
  def expr({:in, left, right}, env), do: expr(left, env) in expr(right, env)
  def expr({:not_in, left, right}, env), do: expr(left, env) not in expr(right, env)
  def expr({:and, left, right}, env), do: expr(left, env) == true and expr(right, env) == true
  def expr({:or, left, right}, env), do: expr(left, env) == true or expr(right, env) == true
  def expr({:not, operand}, env), do: expr(operand, env) != true
  def expr(list, env) when is_list(list), do: Enum.map(list, &expr(&1, env))
end
