defmodule Umpire.Case do
  @moduledoc """
  ExUnit support for Model tests.

      defmodule PinsTest do
        use Umpire.Case
        ...
      end

  Provides `assert_query/3`, which runs a Query and on failure prints the search's own account
  (headline, statistics, the trace); `assert_agrees/2`, which checks that the compiled step
  functions and the IR interpreter give the same steps on every row; and `static_assert/1`, a
  pin decided when the test file compiles, before any test runs.
  """

  alias Umpire.{Domain, Eval, Search}

  defmacro __using__(opts) do
    quote do
      use ExUnit.Case, unquote(Keyword.put_new(opts, :async, true))
      import Umpire.Case
    end
  end

  @doc "Run `query` of `model` and assert its outcome, printing the trace when it differs."
  def assert_query(model, query, expected) do
    result = Search.run(model, query)

    unless result.outcome == expected do
      ExUnit.Assertions.flunk("expected #{inspect(expected)}, got #{inspect(result.outcome)}\n" <> Search.format(result, model.__umpire__(:ir)))
    end

    result
  end

  @doc """
  The emitted functions and `Umpire.Eval` agree on every state and every class of `machine`.
  This is what keeps the two readings of one source from drifting apart: the BEAM's `case` and
  this library's matcher.
  """
  def assert_agrees(model, machine_name) do
    machine = Enum.find(model.__umpire__(:ir).machines, &(&1.name == machine_name))
    actions = Map.new(model.__umpire__(:ir).actions, &{&1.name, &1})

    for state <- machine.state.values(),
        {action, inputs} <- Umpire.Table.classes(machine, actions) do
      step = Map.fetch!(machine.steps, action)
      compiled = apply(machine.module, step.fun, [state | inputs])
      {:ok, _clause, interpreted} = Eval.step(step, state, inputs, machine)

      unless Enum.sort(compiled) == Enum.sort(interpreted) do
        ExUnit.Assertions.flunk("""
        #{inspect(machine.module)}.#{step.fun} and the IR disagree on #{Umpire.Names.key(state)} / #{Umpire.Names.key({action, inputs})}
          compiled:    #{inspect(compiled)}
          interpreted: #{inspect(interpreted)}\
        """)
      end
    end

    :ok
  end

  @doc """
  A pin decided at the test file's compile time: `static_assert ProtocolState.size() == 288`.
  The expression is evaluated while ExUnit compiles the `.exs`, so a failure stops the run before
  any test starts and points at the pin's line. The Models are compiled by then, which is what
  lets the expression call them.
  """
  defmacro static_assert(expr) do
    {value, _binding} = Code.eval_quoted(expr, [], __CALLER__)

    unless value == true do
      raise CompileError,
        file: __CALLER__.file,
        line: __CALLER__.line,
        description: "static_assert failed: #{Macro.to_string(expr)}"
    end

    :ok
  end

  @doc "Every member of a domain module, for pins that range over one."
  def values(domain), do: Domain.values(domain)
end
