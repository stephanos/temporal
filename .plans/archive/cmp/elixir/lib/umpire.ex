defmodule Umpire do
  @moduledoc """
  The Umpire model layer, in Elixir.

  A Model is a module that `use`s `Umpire.Model` and declares, top to bottom, its vocabulary
  (`entity`, `domain`, `defstate`, `action`, `observation`), its machines (`defmachine`), and what
  they promise (`defproperty`, `defscenario`, `deflimits`, `defquery`, `defset`, `defcompose`).

  Every `defmachine` produces two things from one source:

    * **IR** (`Umpire.IR`): plain structs in which patterns, guards, updates and predicates are
      data, not closures. The table, the coverage check, the refinement and the search all read
      the IR. `mix umpire.ir` writes it as canonical JSON, one `<model>.ir.json` per Model, which
      is what a Go core or a Quint exporter would consume.
    * **Elixir functions**, one per step, in a module named after the machine
      (`StandaloneActivity.ActivityProtocol.attempt_result/2`). The patterns and guards are the
      author's own, and the bodies are lowered from the same IR, so the Elixir 1.20 type checker
      sees every step body and reports what it infers as compile-time warnings.

  What the language cannot check, the macros do, in `@before_compile`: that every domain is
  finite, that each step's `case` covers every (phase, input) combination with no dead arm, that
  step bodies stay inside the expression subset (`Umpire.Lower`), and that the refinement holds
  (`Umpire.Refinement`). Queries run in `mix test` (`Umpire.Search`).

  The pipeline, per Model module:

      expansion        each macro checks its own shape (keys, heads, arity); no cross references
      body evaluation  each declaration is lowered to IR and stored in an accumulating attribute
      @before_compile  names resolved, tables built, coverage and refinement checked,
                       `__umpire__/1` defined with the finished IR
      mix test         queries searched, pins asserted, IR interpreter and compiled functions
                       compared row by row
  """

  @typedoc "An action class: an action name and one assignment of its finite inputs."
  @type class :: {atom(), [term()]}

  @doc "The finished IR of a Model module."
  @spec ir(module()) :: Umpire.IR.Model.t()
  def ir(model), do: model.__umpire__(:ir)
end

defmodule Umpire.Step do
  @moduledoc """
  `{outcome, state, facts}`: what one enabled action does.

  A step function returns `[]` when its action is not enabled and more than one step when the
  action is nondeterministic. Every step function the macros emit returns a list of these.
  """

  @enforce_keys [:outcome, :state, :facts]
  defstruct @enforce_keys

  @type t :: %__MODULE__{outcome: atom(), state: struct(), facts: [atom() | tuple()]}
end

defmodule Umpire.Diagnostic do
  @moduledoc """
  Where a declaration came from, and the `CompileError`s that point back at it.

  Every IR node carries the `file` and `line` of the macro call that produced it, because most
  checks run in `@before_compile`, long after the caller's `__CALLER__` is gone. A `CompileError`
  raised with that location is reported by `mix compile` exactly as a syntax error on that line.
  """

  defstruct [:file, :line]

  @type t :: %__MODULE__{file: String.t(), line: pos_integer()}

  @spec loc(Macro.Env.t()) :: t()
  def loc(%Macro.Env{file: file, line: line}), do: %__MODULE__{file: file, line: line}

  @spec at(t(), keyword()) :: t()
  def at(%__MODULE__{} = loc, meta), do: %{loc | line: Keyword.get(meta, :line, loc.line)}

  @spec raise!(t(), String.t()) :: no_return()
  def raise!(%__MODULE__{file: file, line: line}, message) do
    raise CompileError, file: file, line: line, description: message
  end

  @doc "A keyword list whose keys are all in `allowed`."
  def keys!(opts, allowed, loc, what) when is_list(opts) do
    case Keyword.keys(opts) -- allowed do
      [] -> opts
      unknown -> raise!(loc, "#{what} has unknown keys #{inspect(unknown)}; allowed: #{inspect(allowed)}")
    end
  end

  def keys!(other, _allowed, loc, what),
    do: raise!(loc, "#{what} expects a keyword list, got: #{Macro.to_string(other)}")

  def atom!(name, _loc, _what) when is_atom(name) and name not in [nil, true, false], do: name

  def atom!(other, loc, what),
    do: raise!(loc, "#{what} must be an atom such as :completes, got: #{Macro.to_string(other)}")

  @doc "The short name of a single-segment alias such as `ProtocolState`."
  def alias!({:__aliases__, _, [short]}, _loc, _what) when is_atom(short), do: short

  def alias!(other, loc, what),
    do: raise!(loc, "#{what} must be a one-segment alias such as ProtocolState, got: #{Macro.to_string(other)}")

  @doc "The candidate closest to `name`, if any is close enough to suggest."
  @spec suggest(atom(), [atom()]) :: String.t()
  def suggest(name, candidates) do
    candidates
    |> Enum.map(&{String.jaro_distance(Atom.to_string(name), Atom.to_string(&1)), &1})
    |> Enum.max(fn -> {0.0, nil} end)
    |> case do
      {distance, best} when distance > 0.8 -> "\n    did you mean #{inspect(best)}?"
      _ -> ""
    end
  end
end

defmodule Umpire.Names do
  @moduledoc """
  The one place spec names become Elixir identifiers.

  Spec names are atoms and stay verbatim in the IR. Only two things are derived:

    * a machine's module: `:activityProtocol` becomes `<Model>.ActivityProtocol`;
    * a step, map or property function: `:attemptResult` becomes `attempt_result`.

  Two machines both step on `:attemptResult` without clashing because each function lives in its
  machine's module. `module!/2` rejects two machine names that camelize alike.
  """

  @spec module(module(), atom()) :: module()
  def module(model, name), do: Module.concat(model, name |> Atom.to_string() |> Macro.camelize())

  @spec fun(atom()) :: atom()
  def fun(name), do: name |> Atom.to_string() |> Macro.underscore() |> String.to_atom()

  @doc "A stable string for a class or state, used in row keys and error messages."
  @spec key(term()) :: String.t()
  def key({action, inputs}) when is_atom(action) and is_list(inputs),
    do: Enum.map_join([action | Enum.flat_map(inputs, &parts/1)], "-", &to_string/1)

  def key(%module{} = state) do
    module.__umpire_state__().fields
    |> Enum.flat_map(fn {field, _type} -> parts(Map.fetch!(state, field)) end)
    |> Enum.join("-")
  end

  defp parts(%_{} = member_state), do: [key(member_state)]
  defp parts(value) when is_tuple(value), do: value |> Tuple.to_list() |> Enum.flat_map(&parts/1)
  defp parts(value), do: [value]
end
