defmodule Umpire.Check do
  @moduledoc """
  The checks that need more than one declaration, run from the two `@before_compile` hooks.

  `machine!/1` resolves one machine's names against its Model; `model!/1` checks the Model as a
  whole: that each registry has unique names, that properties read fields their machine has and
  name actions it steps on, that scenarios name classes their machine has, that queries are
  admissible (a product property may be read on a protocol scenario only through a declared
  refinement), that sets bind every party their queries' paths use, and that compositions
  synchronize actions their members have. Each failure is a `CompileError` at the declaration
  that caused it.
  """

  alias Umpire.{Compose, Diagnostic, Domain, IR, Table}

  ## One machine

  @spec machine!(module()) :: IR.Machine.t()
  def machine!(module) do
    get = &Module.get_attribute(module, &1)
    header = get.(:umpire_header)
    loc = get.(:umpire_loc)
    model = get.(:umpire_model)
    actions = model |> Module.get_attribute(:umpire_actions) |> Map.new(&{&1.name, &1})
    steps = get.(:umpire_steps) |> Enum.reverse()
    maps = get.(:umpire_maps) |> Enum.reverse()

    machine = %IR.Machine{
      name: get.(:umpire_name),
      module: module,
      for: header[:for],
      state: header[:state],
      outcome: header[:outcome],
      facts: header[:facts],
      refines: get.(:umpire_refines),
      starts: get.(:umpire_starts) || [],
      ends: get.(:umpire_ends) || [],
      timers: get.(:umpire_timers) || [],
      unobservable: get.(:umpire_unobservable) || [],
      evidence: get.(:umpire_evidence) || [],
      steps: unique!(steps, & &1.action, "defstep"),
      maps: unique!(maps, & &1.name, "defmap"),
      loc: loc
    }

    for key <- [:state, :outcome, :facts], machine |> Map.fetch!(key) |> is_nil() do
      Diagnostic.raise!(loc, "defmachine #{inspect(machine.name)} needs #{key}:")
    end

    entities = model |> Module.get_attribute(:umpire_entities) |> Enum.map(& &1.name)
    subset!([machine.for], entities, loc, "for", "an entity this Model declares")

    phases = phases(machine.state)
    subset!(machine.starts, phases, loc, "starts", "a phase of #{inspect(machine.state)}")
    subset!(machine.ends, phases, loc, "ends", "a phase of #{inspect(machine.state)}")
    subset!(machine.unobservable, machine.timers, loc, "unobservable", "a timer of this machine")

    for timer <- machine.timers, not Map.has_key?(machine.steps, timer) do
      Diagnostic.raise!(loc, "timer #{inspect(timer)} of #{inspect(machine.name)} has no defstep")
    end

    for {action, step} <- machine.steps, a <- List.wrap(actions[action]), a.on || a.creates, (a.on || a.creates) != machine.for do
      Diagnostic.raise!(step.loc, "defstep #{action} in #{inspect(machine.name)}: #{inspect(action)} is an action on #{inspect(a.on || a.creates)}, and this machine is for #{inspect(machine.for)}")
    end

    ctors = machine.facts.__umpire_domain__().members |> Enum.map(&Domain.ctor/1)
    subset!(Keyword.keys(machine.evidence), ctors, loc, "evidence", "a fact constructor of #{inspect(machine.facts)}")

    with {_target, map, _rule} <- machine.refines, false <- Map.has_key?(machine.maps, map) do
      Diagnostic.raise!(loc, "refines names map #{inspect(map)}, and this machine has no defmap #{map}")
    end

    machine
  end

  defp phases(state_module) do
    case Keyword.fetch(state_module.__umpire_state__().fields, :phase) do
      {:ok, type} -> Domain.values(type)
      :error -> []
    end
  end

  ## The whole Model

  @spec model!(module()) :: IR.Model.t()
  def model!(module) do
    get = &(module |> Module.get_attribute(&1) |> Enum.reverse())
    machine_modules = get.(:umpire_machines)
    tables = Map.new(machine_modules, &{&1.__umpire_machine__().name, &1.table()})
    tables = Enum.reduce(get.(:umpire_restrictions), tables, &restrict!/2)
    compositions = get.(:umpire_compositions)
    tables = Enum.reduce(compositions, tables, &Map.put(&2, &1.name, compose!(&1, &2)))

    model = %IR.Model{
      name: module,
      entities: get.(:umpire_entities),
      domains: get.(:umpire_domains),
      states: get.(:umpire_states),
      actions: get.(:umpire_actions),
      observations: get.(:umpire_observations),
      machines: Enum.map(machine_modules, & &1.__umpire_machine__()),
      restrictions: get.(:umpire_restrictions),
      properties: get.(:umpire_properties),
      scenarios: get.(:umpire_scenarios),
      limits: get.(:umpire_limits),
      queries: get.(:umpire_queries),
      sets: get.(:umpire_sets),
      compositions: compositions,
      tables: tables
    }

    for key <- [:entities, :actions, :observations, :properties, :scenarios, :limits, :queries, :sets],
        do: unique!(Map.fetch!(model, key), & &1.name, key)

    Enum.each(model.properties, &property!(&1, model))
    Enum.each(model.scenarios, &scenario!(&1, model))
    Enum.each(model.queries, &query!(&1, model))
    Enum.each(model.sets, &set!(&1, model))
    model
  end

  defp restrict!(%IR.Restriction{from: {from_model, from_machine}} = r, tables) do
    source = from_model.__umpire__({:table, from_machine})
    actions = source.classes |> Enum.map(&elem(&1, 0)) |> Enum.uniq()
    subset!(r.only, actions, r.loc, "only", "an action of #{inspect(from_machine)}")
    Map.put(tables, r.name, Table.restrict(source, r.name, r.only))
  end

  defp compose!(%IR.Composition{} = c, tables) do
    for {member, machine} <- c.members, not Map.has_key?(tables, machine) do
      Diagnostic.raise!(c.loc, "defcompose #{inspect(c.name)}: member #{member} names #{inspect(machine)}, which is not a machine of this Model")
    end

    for {name, pairs} <- c.sync, {member, action} <- pairs,
        not Enum.any?(tables[c.members[member]].classes, &(elem(&1, 0) == action)) do
      Diagnostic.raise!(c.loc, "defcompose #{inspect(c.name)}: sync #{name} names #{member}.#{action}, and #{inspect(c.members[member])} has no action #{inspect(action)}")
    end

    Compose.table!(c, tables)
  end

  defp property!(%IR.Property{} = p, model) do
    table = table!(model, p.machine, p.loc, "defproperty #{inspect(p.name)}")
    if p.when, do: class!(p.when, table, p.loc, "defproperty #{inspect(p.name)} when:")
    sample = hd(table.states)
    walk_paths(p.holds, fn path -> path!(path, sample, p) end)
  end

  defp walk_paths({:path, _index, path}, fun), do: fun.(path)
  defp walk_paths(tuple, fun) when is_tuple(tuple), do: tuple |> Tuple.to_list() |> Enum.each(&walk_paths(&1, fun))
  defp walk_paths(list, fun) when is_list(list), do: Enum.each(list, &walk_paths(&1, fun))
  defp walk_paths(_other, _fun), do: :ok

  defp path!([root | rest], sample, p) when root in [:state, :facts, :outcome] do
    if root == :state do
      Enum.reduce(rest, sample, fn field, value ->
        if is_map(value) and Map.has_key?(value, field),
          do: Map.fetch!(value, field),
          else: Diagnostic.raise!(p.loc, "defproperty #{inspect(p.name)} reads .#{field}, which #{inspect(p.machine)}'s state does not have")
      end)
    end
  end

  defp path!(path, _sample, p),
    do: Diagnostic.raise!(p.loc, "defproperty #{inspect(p.name)} reads #{inspect(path)}; a step has state, facts and outcome")

  defp scenario!(%IR.Scenario{} = s, model) do
    table = table!(model, s.model, s.loc, "defscenario #{inspect(s.name)}")
    Enum.each(s.actions, &class!(&1, table, &1.loc, "defscenario #{inspect(s.name)}"))
  end

  defp query!(%IR.Query{} = q, model) do
    what = "defquery #{inspect(q.name)}"
    p = find!(model.properties, q.property, q.loc, what, "property")
    s = find!(model.scenarios, q.scenario, q.loc, what, "scenario")
    l = find!(model.limits, q.limits, q.loc, what, "limits")

    if q.kind == :find and p.kind == :transition,
      do: Diagnostic.raise!(q.loc, "#{what}: find: needs a same-step claim; #{inspect(p.name)} is a transition claim, which is verified, never realized")

    # A product property on a protocol path is read through the map; the claim's action must be
    # one the path's machine steps on, or the claim could never fire.
    _view = Umpire.Search.view!(p.machine, s.model, model, q.loc)
    if p.when, do: class!(p.when, Map.fetch!(model.tables, s.model), q.loc, "#{what}: #{inspect(p.name)} when: on #{inspect(s.model)}")

    admitted = admitted(l)

    if length(s.actions) > admitted,
      do: Diagnostic.raise!(q.loc, "#{what}: #{inspect(s.name)} has #{length(s.actions)} actions; limits #{inspect(l.name)} admit #{admitted}")
  end

  # `deflimits` already required positive integers; the guard tells the type checker so.
  defp admitted(%IR.Limits{actions: actions, steps: steps}) when is_integer(actions) and is_integer(steps),
    do: min(actions, steps)

  defp set!(%IR.TestSet{purpose: :exploratory} = set, model) do
    table!(model, set.machine, set.loc, "defset #{inspect(set.name)}")
    find!(model.limits, set.budget, set.loc, "defset #{inspect(set.name)}", "limits")
    subset!(set.cover, [:rows, :results, :classMembers], set.loc, "cover", "one of :rows, :results, :classMembers")
  end

  defp set!(%IR.TestSet{} = set, model) do
    what = "defset #{inspect(set.name)}"
    queries = Enum.map(set.queries, &find!(model.queries, &1, set.loc, what, "query"))

    for q <- queries, q.kind != :find,
        do: Diagnostic.raise!(set.loc, "#{what}: #{inspect(q.name)} is a verify query, which realizes nothing and belongs outside a set")

    parties =
      for q <- queries,
          ref <- find!(model.scenarios, q.scenario, set.loc, what, "scenario").actions,
          # Timers are the reserved party :system's and are in no action registry.
          action <- Enum.filter(model.actions, &(&1.name == ref.action)),
          into: MapSet.new(),
          do: action.party

    for party <- parties, not Keyword.has_key?(set.bind, party),
        do: Diagnostic.raise!(set.loc, "#{what}: its queries' paths use party #{inspect(party)}, which bind: leaves unbound")

    if set.purpose == :canary and not Enum.any?(set.bind, &match?({_, :observed}, &1)),
      do: Diagnostic.raise!(set.loc, "#{what}: a canary observes at least one party")
  end

  ## Helpers

  defp table!(model, name, loc, what) do
    Map.get(model.tables, name) ||
      Diagnostic.raise!(loc, "#{what} names #{inspect(name)}, which is not a machine or composition of this Model" <> Diagnostic.suggest(name, Map.keys(model.tables)))
  end

  # A scenario or `when:` class against a table's classes. Composition classes are `{name, inputs}`
  # for a sync line and `{{member, action}, inputs}` for a member's own action.
  defp class!(%IR.ClassRef{} = ref, table, loc, what) do
    action = if ref.member, do: {ref.member, ref.action}, else: ref.action

    found =
      Enum.any?(table.classes, fn {a, inputs} ->
        a == action and (ref.inputs == :any or inputs == ref.inputs)
      end)

    unless found do
      names = table.classes |> Enum.map(&elem(&1, 0)) |> Enum.uniq()

      Diagnostic.raise!(loc, """
      #{what}: #{Umpire.Names.key({ref.action, List.wrap(if ref.inputs == :any, do: [], else: ref.inputs)})} is not a class of #{inspect(table.machine)}
          its actions: #{Enum.map_join(names, ", ", &inspect/1)}\
      #{if is_atom(action), do: Diagnostic.suggest(action, Enum.filter(names, &is_atom/1)), else: ""}\
      """)
    end
  end

  defp find!(list, name, loc, what, kind) do
    Enum.find(list, &(&1.name == name)) ||
      Diagnostic.raise!(loc, "#{what} names #{kind} #{inspect(name)}, which is not declared" <> Diagnostic.suggest(name, Enum.map(list, & &1.name)))
  end

  defp subset!(values, allowed, loc, key, what) do
    case Enum.reject(List.wrap(values), &(&1 in allowed)) do
      [] -> :ok
      [bad | _] -> Diagnostic.raise!(loc, "#{key}: #{inspect(bad)} is not #{what}" <> Diagnostic.suggest(bad, allowed))
    end
  end

  defp unique!(items, name_of, kind) do
    Enum.reduce(items, %{}, fn item, seen ->
      name = name_of.(item)
      if Map.has_key?(seen, name), do: Diagnostic.raise!(item.loc, "#{kind} #{inspect(name)} is declared twice")
      Map.put(seen, name, item)
    end)
  end
end
