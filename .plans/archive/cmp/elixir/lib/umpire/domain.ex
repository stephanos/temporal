defmodule Umpire.Domain do
  @moduledoc """
  Finite types and their enumeration.

  A domain is declared with `domain Name, members` and becomes the module `<Model>.Name` with
  `values/0`, `size/0`, a `@type t` for Dialyzer and docs, and `__umpire_domain__/0` for the IR.
  A state is declared with `defstate Name, field: type, ...` and becomes a struct module with the
  same functions over the product of its fields.

  A member is an atom, or a constructor with finite fields, written `{:failed, retryable:
  :boolean}` and enumerated as `{:failed, false}`, `{:failed, true}`. A field type is `:boolean`,
  a `Range` such as `0..@attemptBound`, or another domain's module. Nothing else is accepted,
  which is the whole of the finiteness check: every type is built from finitely many atoms,
  booleans and bounded integers, so every state space is finite by construction.

  Enumeration order is declaration order, and for a product the first field varies slowest, so
  the first state with a given phase has every other field at its first value. `start/2` relies
  on that: `starts: [:unstarted]` means the first state whose phase is `:unstarted`.
  """

  alias Umpire.{Diagnostic, IR}

  @type type :: :boolean | Range.t() | module()

  @doc "Validate a domain's members at module-body evaluation, when every referenced module exists."
  @spec declare!(module(), list(), Diagnostic.t()) :: IR.Domain.t()
  def declare!(module, members, loc) do
    members =
      Enum.map(members, fn
        atom when is_atom(atom) and atom not in [nil, true, false] ->
          atom

        {ctor, fields} when is_atom(ctor) and is_list(fields) ->
          {ctor, Enum.map(fields, fn {field, type} -> {field, type!(type, loc, "#{inspect(ctor)}.#{field}")} end)}

        other ->
          Diagnostic.raise!(loc, "domain member #{inspect(other)} is neither an atom nor {constructor, field: type}")
      end)

    duplicates = members |> Enum.map(&ctor/1) |> then(&(&1 -- Enum.uniq(&1)))
    if duplicates != [], do: Diagnostic.raise!(loc, "domain #{inspect(module)} repeats #{inspect(duplicates)}")

    %IR.Domain{name: module |> Module.split() |> List.last(), module: module, members: members, loc: loc}
  end

  @doc "Validate a state's fields; the struct module itself is created by `Umpire.Model`."
  @spec declare_state!(module(), keyword(), Diagnostic.t()) :: IR.StateType.t()
  def declare_state!(module, fields, loc) do
    fields = Enum.map(fields, fn {field, type} -> {field, type!(type, loc, "field #{field}")} end)
    %IR.StateType{name: module |> Module.split() |> List.last(), module: module, fields: fields, loc: loc}
  end

  defp type!(:boolean, _loc, _what), do: :boolean
  defp type!(%Range{first: lo, last: hi, step: 1} = range, _loc, _what) when lo <= hi, do: range

  defp type!(module, loc, what) when is_atom(module) do
    # A domain of another Model may still be compiling; this waits for it like a remote call does.
    Code.ensure_compiled(module)

    cond do
      function_exported?(module, :__umpire_domain__, 0) ->
        module

      function_exported?(module, :__umpire_state__, 0) ->
        module

      true ->
        Diagnostic.raise!(loc, """
        #{what} has type #{inspect(module)}, which is not a finite type.
          A field type is :boolean, a range such as 0..2, or a module declared with domain/2 or defstate/2.\
        """)
    end
  end

  defp type!(other, loc, what),
    do: Diagnostic.raise!(loc, "#{what} has type #{inspect(other)}, which is not a finite type")

  @doc "Every member of a type, in declaration order."
  @spec values(type() | IR.Domain.t() | IR.StateType.t()) :: [term()]
  def values(:boolean), do: [false, true]
  def values(%Range{} = range), do: Enum.to_list(range)

  def values(%IR.Domain{members: members}) do
    Enum.flat_map(members, fn
      atom when is_atom(atom) ->
        [atom]

      {ctor, fields} ->
        for assignment <- product(Enum.map(fields, fn {_field, type} -> values(type) end)),
            do: List.to_tuple([ctor | assignment])
    end)
  end

  # Built as maps tagged with `__struct__` rather than with `struct!/2`: the states are
  # enumerated before their module exists, so that they can be compiled into it as data.
  def values(%IR.StateType{module: module, fields: fields}) do
    for assignment <- product(Enum.map(fields, fn {_field, type} -> values(type) end)),
        do: Map.new([{:__struct__, module} | Enum.zip(Keyword.keys(fields), assignment)])
  end

  def values(module) when is_atom(module), do: module.values()

  @doc "The cartesian product of lists, first list slowest."
  @spec product([[term()]]) :: [[term()]]
  def product([]), do: [[]]
  def product([values | rest]), do: for(value <- values, tail <- product(rest), do: [value | tail])

  @doc "The first state whose phase is `phase`."
  @spec start(module(), atom()) :: struct()
  def start(state_module, phase), do: Enum.find(state_module.values(), &(&1.phase == phase))

  @doc "A member's constructor name: the atom itself, or the tag of a constructor tuple."
  def ctor({ctor, _fields}), do: ctor
  def ctor(value) when is_tuple(value), do: elem(value, 0)
  def ctor(atom) when is_atom(atom), do: atom

  @doc "Whether `value` is a member of `type`."
  @spec member?(type(), term()) :: boolean()
  def member?(type, value), do: value in values(type)

  @doc "The typespec union for a domain's validated members, for `@type t` in its module."
  @spec typespec(IR.Domain.t()) :: Macro.t()
  def typespec(%IR.Domain{members: []}), do: quote(do: none())

  def typespec(%IR.Domain{members: members}) do
    members
    |> Enum.map(fn
      atom when is_atom(atom) -> atom
      {ctor, fields} -> {:{}, [], [ctor | Enum.map(fields, fn {_f, t} -> field_spec(t) end)]}
    end)
    |> Enum.reverse()
    |> Enum.reduce(fn member, union -> {:|, [], [member, union]} end)
  end

  @doc "The struct typespec for a validated state."
  @spec state_typespec(IR.StateType.t()) :: Macro.t()
  def state_typespec(%IR.StateType{fields: fields}) do
    quote do: %__MODULE__{unquote_splicing(Enum.map(fields, fn {f, t} -> {f, field_spec(t)} end))}
  end

  defp field_spec(:boolean), do: quote(do: boolean())
  defp field_spec(%Range{first: lo, last: hi}), do: quote(do: unquote(lo)..unquote(hi))
  defp field_spec(module) when is_atom(module), do: quote(do: unquote(module).t())
end
