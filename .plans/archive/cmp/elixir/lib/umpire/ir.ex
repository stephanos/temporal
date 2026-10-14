defmodule Umpire.IR do
  @moduledoc """
  The Model as data.

  Everything a checker, a search or an exporter needs is here, and nothing here is a closure: a
  step is a list of clauses whose patterns, guards and bodies are the tagged tuples below. That is
  what lets one Model feed the native table and search in this library, a JSON export for a Go
  core, and a later compiler to Quint or TLA+.

  ## Expressions

      expr    :: {:lit, term}                     a literal: atom, integer, boolean, tuple, struct
               | {:var, atom}                     a variable bound by a pattern
               | {:input, atom}                   an input of the step's action, by field name
               | {:field, atom}                   a field of the step's state
               | {:succ, expr, max}               saturating successor; max is the field's bound
               | {:tuple, [expr]}                 a tuple built from expressions
               | {:path, index, [atom]}           a property argument and a field path, e.g.
                                                  {:path, 0, [:state, :worker, :phase]}
               | {:eq | :neq, expr, expr}
               | {:in | :not_in, expr, expr}      right side a {:lit, list} or a {:path, _, [:facts]}
               | {:and | :or, expr, expr}
               | {:not, expr}

      pattern :: :any | {:bind, atom} | {:lit, term} | {:tuple, [pattern]}

      body    :: :disabled                        []            not enabled here
               | :stay                            stay()        accepted, same state, no facts
               | :not_found                       not_found()   :notFound, same state, no facts
               | {:moves, expr, [expr], [{atom, expr}]}
                                                  moves(phase, facts, updates)
               | {:state, module, [{atom, expr}]} %ProductState{...}, in a map body only

  A step's `subjects` are what its `case` matches on (`[{:field, :phase}, {:input, :result}]`),
  and a clause pattern is matched against the tuple of their values.
  """

  alias Umpire.Diagnostic

  @type expr :: tuple()
  @type pattern :: :any | tuple()
  @type body :: atom() | tuple()

  defmodule Entity do
    @moduledoc false
    defstruct [:name, :key, refer: [], loc: nil]
  end

  defmodule Domain do
    @moduledoc """
    A finite type. `members` are atoms or `{constructor, [{field, type}]}`, where a type is
    `:boolean`, `{:range, lo, hi}` or another domain's module.
    """
    defstruct [:name, :module, :members, :loc]
  end

  defmodule StateType do
    @moduledoc "A state struct: its module and its fields in declaration order, each of a finite type."
    defstruct [:name, :module, :fields, :loc]
  end

  defmodule Action do
    @moduledoc false
    defstruct [:name, :party, :creates, :on, :schema, :results, input: [], examples: %{}, loc: nil]
  end

  defmodule Observation do
    @moduledoc false
    defstruct [:name, :on, :read, :loc]
  end

  defmodule Clause do
    @moduledoc "One `pattern [when guard] -> body` arm, with the line it was written on."
    defstruct [:pattern, :guard, :body, :loc]
  end

  defmodule StepFn do
    @moduledoc """
    One `defstep`. `inputs` pairs each declared input field with the variable the head binds it
    to (`[result: :result]`, `[scheduleToClose: :schedule_to_close]`).
    """
    defstruct [:action, :fun, :state_var, inputs: [], subjects: [], clauses: [], loc: nil]
  end

  defmodule Abstraction do
    @moduledoc "One `defmap`: a total function from this machine's states to another machine's."
    defstruct [:name, :fun, :state_var, subjects: [], clauses: [], loc: nil]
  end

  defmodule Machine do
    @moduledoc """
    One `defmachine`. `refines` is `{target, map_name, rule}` or nil; `rule` is `:strict` (the
    Lean checker's rule, the default) or `:mapped_states` (the rule SPEC.md states).
    """
    defstruct [
      :name,
      :module,
      :for,
      :state,
      :outcome,
      :facts,
      :refines,
      starts: [],
      ends: [],
      timers: [],
      unobservable: [],
      evidence: [],
      steps: %{},
      maps: %{},
      loc: nil
    ]
  end

  defmodule Restriction do
    @moduledoc "`defmachine name, from: {Model, machine}, only: [...]`: a machine with fewer actions."
    defstruct [:name, :from, :only, :loc]
  end

  defmodule ClassRef do
    @moduledoc """
    An action class as a scenario or a `when:` names it. `member` is set for a composition's
    qualified action (`activity.backoff`); `inputs` is `:any` for a bare `when:` name.
    """
    defstruct [:member, :action, inputs: [], loc: nil]
  end

  defmodule Property do
    @moduledoc """
    `kind` is `:same_step` (one argument, requires `when`) or `:transition` (two arguments).
    `holds` is an expression over `{:path, 0 | 1, ...}`.
    """
    defstruct [:name, :machine, :kind, :when, :fun, :holds, loc: nil]
  end

  defmodule Scenario do
    @moduledoc false
    defstruct [:name, :model, :starts, actions: [], loc: nil]
  end

  defmodule Limits do
    @moduledoc false
    defstruct [:name, :steps, :actions, :search, :loc]
  end

  defmodule Query do
    @moduledoc """
    `kind` is `:find` or `:verify`. `require_firing` makes a verify whose claim never fired an
    error rather than a pass; it defaults to true (the Lean framework defaults it to false).
    """
    defstruct [:name, :kind, :property, :scenario, :limits, require_firing: true, loc: nil]
  end

  defmodule TestSet do
    @moduledoc "A `defset`. Named `TestSet` so that `Set` stays free for readers."
    defstruct [:name, :purpose, :repeat, :machine, :budget, bind: [], queries: [], cover: [], loc: nil]
  end

  defmodule Composition do
    @moduledoc """
    `sync` is `[{name, [{member, action}, {member, action}]}]`: the two member actions fire as one
    class named `name`. Unsynchronized member actions stay executable on their own, qualified by
    member (`{:activity, :backoff}`).
    """
    defstruct [:name, :for, :state, members: [], sync: [], starts: [], ends: [], loc: nil]
  end

  defmodule Model do
    @moduledoc "One Model module, complete. The tables are included so the IR alone can be checked."
    defstruct [
      :name,
      entities: [],
      domains: [],
      states: [],
      actions: [],
      observations: [],
      machines: [],
      restrictions: [],
      properties: [],
      scenarios: [],
      limits: [],
      queries: [],
      sets: [],
      compositions: [],
      tables: %{}
    ]

    @type t :: %__MODULE__{}
  end

  @doc """
  Canonical JSON for a Model: object keys sorted, atoms as strings, tagged tuples as
  `{"op": tag, "args": [...]}`, value tuples as `{"ctor": name, "args": [...]}`, maps keyed by
  values as sorted `{"entries": [{"key", "value"}]}`, locations dropped. The Model holds no floats, which is where RFC 8785 is otherwise subtle; a golden test
  pins the bytes.
  """
  @spec to_json(term()) :: iodata()
  def to_json(term), do: term |> plain() |> encode()

  @ops ~w(lit var input field succ tuple path eq neq in not_in and or not bind moves state)a

  defp plain(%Diagnostic{}), do: nil
  defp plain(%Range{first: lo, last: hi}), do: %{"range" => [lo, hi]}
  defp plain(%_{} = struct), do: struct |> Map.from_struct() |> Map.delete(:loc) |> plain()
  # An object when every key is a name; otherwise (the `examples:` maps are keyed by input values
  # such as `{:handlerError, false}`) a list of key/value entries, sorted by the encoded key.
  defp plain(map) when is_map(map) do
    if Enum.all?(Map.keys(map), &(is_atom(&1) or is_binary(&1))) do
      Map.new(map, fn {k, v} -> {to_string(k), plain(v)} end)
    else
      entries = Enum.map(map, fn {k, v} -> %{"key" => plain(k), "value" => plain(v)} end)
      %{"entries" => Enum.sort_by(entries, &IO.iodata_to_binary(encode(&1["key"])))}
    end
  end
  defp plain(list) when is_list(list), do: Enum.map(list, &plain/1)
  defp plain({op, _} = t) when op in @ops, do: tagged(t)
  defp plain({op, _, _} = t) when op in @ops, do: tagged(t)
  defp plain({op, _, _, _} = t) when op in @ops, do: tagged(t)
  defp plain(t) when is_tuple(t), do: plain_ctor(Tuple.to_list(t))
  defp plain(atom) when is_atom(atom) and atom not in [nil, true, false], do: Atom.to_string(atom)
  defp plain(other), do: other

  defp tagged(t), do: %{"op" => Atom.to_string(elem(t, 0)), "args" => t |> Tuple.to_list() |> tl() |> plain()}
  defp plain_ctor([ctor | args]), do: %{"ctor" => plain(ctor), "args" => plain(args)}

  defp encode(map) when is_map(map) do
    pairs = map |> Enum.sort_by(&elem(&1, 0)) |> Enum.map(fn {k, v} -> [JSON.encode!(k), ?:, encode(v)] end)
    [?{, Enum.intersperse(pairs, ?,), ?}]
  end

  defp encode(list) when is_list(list), do: [?[, list |> Enum.map(&encode/1) |> Enum.intersperse(?,), ?]]
  defp encode(scalar), do: JSON.encode!(scalar)
end
