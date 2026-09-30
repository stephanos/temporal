defmodule Umpire.Lower do
  @moduledoc """
  From the author's AST to IR, and from IR back to Elixir.

  This module is the expression subset. A step or map body is one `case` over subjects (state
  fields and the action's inputs) or a single step constructor. Inside it:

    * patterns: literals, `_`, variables, tuples of patterns;
    * guards: `==`, `!=`, `in`, `not in`, `and`, `or`, `not`, over bound variables, subjects,
      literals, literal lists and `@attributes`;
    * bodies: `[]`, `stay()`, `not_found()`, `moves(phase, facts, updates)` whose arguments are
      literals, bound variables, inputs, state fields, `succ(state.field)`, tuples of those, and
      `@attributes`; in a `defmap`, a struct literal of the target state;
    * property predicates: the same operators over `step.state...`, `step.facts`,
      `step.outcome` (or `before`/`next` for a transition claim).

  Anything else is a `CompileError` at its line: any function call (local, remote, anonymous),
  `send`, `receive`, `if`, `cond`, `with`, pipes, comprehensions, string interpolation, variable
  rebinding. The subset is what makes a body data: it can be enumerated for coverage, compared by
  the refinement, exported as JSON, and compiled to another checker, none of which a closure
  allows. The price is that a Model says in several arms what ordinary Elixir would say with `++`.

  The emitted function keeps the author's patterns and guards verbatim (with `@attributes`
  replaced by their values, since an attribute of the Model is not visible inside a machine's
  module) and replaces each body by the `%Umpire.Step{}` list its IR stands for. Line metadata is
  kept, so a type checker warning points at the author's line.
  """

  alias Umpire.{Diagnostic, Domain, IR}

  @constructors [:moves, :stay, :not_found]

  ## Shapes, at expansion

  @doc false
  def members_shape!(members, loc, what) when is_list(members) do
    Enum.each(members, fn
      atom when is_atom(atom) -> :ok
      {ctor, fields} when is_atom(ctor) and is_list(fields) -> keyword!(fields, loc, "#{what} #{inspect(ctor)}")
      other -> Diagnostic.raise!(loc, "#{what}: #{Macro.to_string(other)} is neither an atom nor {constructor, field: type}")
    end)
  end

  def members_shape!(other, loc, what),
    do: Diagnostic.raise!(loc, "#{what} expects a list of members, got: #{Macro.to_string(other)}")

  @doc false
  def keyword!(fields, loc, what) do
    unless is_list(fields) and Enum.all?(fields, &match?({key, _} when is_atom(key), &1)) do
      Diagnostic.raise!(loc, "#{what} expects field: type pairs, got: #{Macro.to_string(fields)}")
    end

    fields
  end

  @doc "A `def`-like head: `attemptResult(state, result)`. Returns `{name, state_var, input_vars}`."
  def head_shape!({name, _meta, [state | inputs]} = head, loc, what) when is_atom(name) do
    vars = Enum.map([state | inputs], &var_name!(&1, loc, "#{what} #{Macro.to_string(head)}"))
    {name, hd(vars), tl(vars)}
  end

  def head_shape!(head, loc, what),
    do: Diagnostic.raise!(loc, "#{what} expects a head such as attemptResult(state, result), got: #{Macro.to_string(head)}")

  defp var_name!({name, _meta, context}, _loc, _what) when is_atom(name) and is_atom(context), do: name

  defp var_name!(other, loc, what),
    do: Diagnostic.raise!(loc, "#{what}: every argument is a plain variable, got: #{Macro.to_string(other)}")

  @doc "`fn step -> ... end` or `fn before, next -> ... end`, one clause, plain variables."
  def fn_shape!({:fn, _, [{:->, _, [params, body]}]}, loc) do
    Enum.each(params, &var_name!(&1, loc, "holds:"))
    {params, body}
  end

  def fn_shape!(other, loc),
    do: Diagnostic.raise!(loc, "holds: expects fn step -> ... end with one clause, got: #{Macro.to_string(other)}")

  @doc """
  An action class as a scenario or `when:` spells it. Bare `backoff` is the action with no input
  (or, under `when:`, any class of it); `attemptResult({:failed, true})` one class;
  `activity.backoff` a composition member's action.
  """
  def class_ref!(ast, loc, opts) do
    {member, action, args, meta} =
      case ast do
        {{:., meta, [{member, _, ctx}, action]}, _, args} when is_atom(member) and is_atom(ctx) -> {member, action, args, meta}
        {action, meta, ctx} when is_atom(action) and is_atom(ctx) -> {nil, action, nil, meta}
        {action, meta, args} when is_atom(action) and is_list(args) -> {nil, action, args, meta}
        other -> Diagnostic.raise!(loc, "#{Macro.to_string(other)} is not an action class such as attemptResult(:completed)")
      end

    inputs =
      case args do
        nil -> if opts[:any_inputs], do: :any, else: []
        [] -> []
        args -> Enum.map(args, &literal!(&1, Diagnostic.at(loc, meta)))
      end

    %IR.ClassRef{member: member, action: action, inputs: inputs, loc: Diagnostic.at(loc, meta)}
  end

  @doc false
  def literal!(ast, loc) do
    if Macro.quoted_literal?(ast) and not match?({:__aliases__, _, _}, ast) do
      {value, []} = Code.eval_quoted(ast)
      value
    else
      Diagnostic.raise!(loc, "a class input is a literal such as :unset or {:failed, true}, got: #{Macro.to_string(ast)}")
    end
  end

  @doc "`[workerStop: activity.workerStop || worker.workerStop, ...]`"
  def sync!(pairs, members, loc) do
    Enum.map(keyword!(pairs, loc, "sync:"), fn
      {name, {:||, meta, [left, right]}} ->
        {name, Enum.map([left, right], &member_action!(&1, members, Diagnostic.at(loc, meta)))}

      {name, other} ->
        Diagnostic.raise!(loc, "sync #{name}: expects member.action || member.action, got: #{Macro.to_string(other)}")
    end)
  end

  defp member_action!({{:., _, [{member, _, ctx}, action]}, _, []}, members, loc) when is_atom(ctx) do
    if member in members, do: {member, action}, else: Diagnostic.raise!(loc, "sync names #{member}, which is not a member (#{inspect(members)})")
  end

  defp member_action!(other, _members, loc),
    do: Diagnostic.raise!(loc, "sync expects member.action, got: #{Macro.to_string(other)}")

  @doc false
  def header_shape!({:@, _, [{name, _, ctx}]}, _loc) when is_atom(name) and is_atom(ctx), do: :ok

  def header_shape!(value, loc) do
    unless Macro.quoted_literal?(value), do: Diagnostic.raise!(loc, "a machine header takes literals or an @attribute, got: #{Macro.to_string(value)}")
  end

  @doc "Resolve aliases where the author wrote them, so an evaluated body can name a module."
  def expand_aliases(ast, env) do
    Macro.prewalk(ast, fn
      {:__aliases__, _, _} = alias_ast -> Macro.expand(alias_ast, env)
      other -> other
    end)
  end

  ## Values, at module-body evaluation

  @doc "A header value: a literal, or an `@attribute` of this module or of its Model."
  def resolve!({:@, _, [{name, _, ctx}]}, module, loc) when is_atom(ctx), do: attribute!(module, name, loc)
  def resolve!(literal, _module, _loc), do: literal

  @doc "An attribute of `module`, else of the Model that `module` belongs to."
  def attribute!(module, name, loc) do
    owners = [module, Module.get_attribute(module, :umpire_model)] |> Enum.reject(&is_nil/1)

    case Enum.find(owners, &Module.has_attribute?(&1, name)) do
      nil -> Diagnostic.raise!(loc, "@#{name} is not set in #{inspect(module)} or in its Model; set it above its first use")
      owner -> Module.get_attribute(owner, name)
    end
  end

  @doc "An `action` declaration, validated against the parties and the domains."
  def action!(name, opts, loc) do
    party = Keyword.get(opts, :party)

    cond do
      party == :system ->
        Diagnostic.raise!(loc, "action #{inspect(name)}: :system is reserved for timers, which a machine declares under timers")

      party not in Umpire.Model.parties() ->
        Diagnostic.raise!(loc, "action #{inspect(name)}: party #{inspect(party)} is not one of #{inspect(Umpire.Model.parties())}")

      true ->
        :ok
    end

    input = Enum.map(Keyword.get(opts, :input, []), fn {field, type} -> {field, domain!(type, loc)} end)

    examples = Keyword.get(opts, :examples, %{})

    for {class, _realization} <- examples do
      classes = input |> Enum.map(fn {_f, t} -> Domain.values(t) end) |> Domain.product() |> Enum.map(&unwrap/1)

      unless class in classes do
        Diagnostic.raise!(loc, "action #{inspect(name)}: example #{inspect(class)} is not a class of its input #{inspect(input)}")
      end
    end

    %IR.Action{
      name: name,
      party: party,
      creates: opts[:creates],
      on: opts[:on],
      schema: opts[:schema],
      results: opts[:results] && domain!(opts[:results], loc),
      input: input,
      examples: examples,
      loc: loc
    }
  end

  defp unwrap([single]), do: single
  defp unwrap(many), do: List.to_tuple(many)

  defp domain!(:boolean, _loc), do: :boolean

  defp domain!(module, loc) do
    if is_atom(module) and match?({:module, _}, Code.ensure_compiled(module)) and function_exported?(module, :__umpire_domain__, 0),
      do: module,
      else: Diagnostic.raise!(loc, "#{inspect(module)} is not a domain; declare it with domain/2")
  end

  ## Steps and maps

  defmodule Scope do
    @moduledoc false
    # What a body may refer to: the state variable, the inputs by variable name, the variables
    # the current pattern bound, and the machine it belongs to.
    # `roots` is used by property predicates only: argument name to position.
    defstruct [
      :module,
      :machine,
      :state,
      :state_var,
      :loc,
      inputs: %{},
      input_types: %{},
      bound: MapSet.new(),
      roots: %{},
      what: "defstep"
    ]
  end

  @doc "Lower one `defstep`. Returns the IR and the parts of the function to emit."
  def step!(module, action, {state_var, input_vars}, body, loc) do
    machine = Module.get_attribute(module, :umpire_name)
    state = Keyword.fetch!(Module.get_attribute(module, :umpire_header), :state).__umpire_state__()
    what = "defstep #{action}/#{length(input_vars) + 1} in #{inspect(machine)}"
    {inputs, input_types} = inputs!(module, action, input_vars, loc, what)

    scope = %Scope{
      module: module,
      machine: machine,
      state: state,
      state_var: state_var,
      loc: loc,
      what: what,
      inputs: Map.new(inputs, fn {field, var} -> {var, field} end),
      input_types: input_types
    }

    {subjects, clauses} = lower_case!(body, scope, &body!/2)

    step = %IR.StepFn{
      action: action,
      fun: Umpire.Names.fun(action),
      state_var: state_var,
      inputs: inputs,
      subjects: subjects,
      clauses: clauses,
      loc: loc
    }

    {step, emit(step, body, scope, &emit_body/2)}
  end

  @doc "Lower one `defmap`."
  def map!(module, name, state_var, body, loc) do
    machine = Module.get_attribute(module, :umpire_name)
    state = Keyword.fetch!(Module.get_attribute(module, :umpire_header), :state).__umpire_state__()
    scope = %Scope{module: module, machine: machine, state: state, state_var: state_var, loc: loc, what: "defmap #{name}"}
    {subjects, clauses} = lower_case!(body, scope, &map_body!/2)
    abstraction = %IR.Abstraction{name: name, fun: Umpire.Names.fun(name), state_var: state_var, subjects: subjects, clauses: clauses, loc: loc}
    {abstraction, emit(abstraction, body, scope, &emit_map_body/2)}
  end

  # The action's declared inputs, paired with the head's variables. A timer has none.
  defp inputs!(module, action, input_vars, loc, what) do
    model = Module.get_attribute(module, :umpire_model)
    actions = model |> Module.get_attribute(:umpire_actions) |> Map.new(&{&1.name, &1})
    timers = Module.get_attribute(module, :umpire_timers) || []

    declared =
      cond do
        Map.has_key?(actions, action) ->
          actions[action].input

        action in timers ->
          []

        true ->
          Diagnostic.raise!(loc, """
          #{what} names no declared action and no timer of this machine
              actions: #{actions |> Map.keys() |> Enum.sort() |> Enum.map_join(", ", &inspect/1)}
              timers:  #{if timers == [], do: "(none declared above this step)", else: Enum.map_join(timers, ", ", &inspect/1)}\
          #{Diagnostic.suggest(action, Map.keys(actions) ++ timers)}\
          """)
      end

    if length(declared) != length(input_vars) do
      Diagnostic.raise!(loc, "#{what} binds #{length(input_vars)} inputs; #{inspect(action)} declares #{length(declared)}: #{inspect(declared)}")
    end

    inputs =
      Enum.zip_with(declared, input_vars, fn {field, _type}, var ->
        if var in [field, Umpire.Names.fun(field)] or String.starts_with?(Atom.to_string(var), "_"),
          do: {field, var},
          else: Diagnostic.raise!(loc, "#{what}: argument #{var} binds input #{field}; name it #{Umpire.Names.fun(field)} or #{field}")
      end)

    {inputs, Map.new(declared)}
  end

  defp lower_case!({:case, _meta, [subject, [do: arms]]}, scope, lower_body) do
    subjects = subjects!(subject, scope)
    {subjects, Enum.map(arms, &clause!(&1, length(subjects), scope, lower_body))}
  end

  defp lower_case!(body, scope, lower_body),
    do: {[], [%IR.Clause{pattern: :any, guard: {:lit, true}, body: lower_body.(body, scope), loc: scope.loc}]}

  defp subjects!({left, right}, scope), do: [subject!(left, scope), subject!(right, scope)]
  defp subjects!({:{}, _, elements}, scope), do: Enum.map(elements, &subject!(&1, scope))
  defp subjects!(single, scope), do: [subject!(single, scope)]

  defp subject!(ast, scope) do
    case expr!(ast, scope) do
      {:field, _} = field -> field
      {:input, _} = input -> input
      _ -> Diagnostic.raise!(scope.loc, "#{scope.what}: a case matches on state fields and inputs, got: #{Macro.to_string(ast)}")
    end
  end

  defp clause!({:->, meta, [[head], body]}, arity, scope, lower_body) do
    loc = Diagnostic.at(scope.loc, meta)
    scope = %{scope | loc: loc}

    {pattern_ast, guard_ast} =
      case head do
        {:when, _, [pattern, guard]} -> {pattern, guard}
        pattern -> {pattern, true}
      end

    pattern = pattern!(pattern_ast, scope)

    unless arity == 1 or match?({:tuple, elements} when length(elements) == arity, pattern) or pattern in [:any] or match?({:bind, _}, pattern) do
      Diagnostic.raise!(loc, "#{scope.what}: the case matches #{arity} subjects, so each pattern is a #{arity}-tuple")
    end

    rebinding!(pattern, scope)
    scope = %{scope | bound: bindings(pattern)}
    %IR.Clause{pattern: pattern, guard: expr!(guard_ast, scope), body: lower_body.(body, scope), loc: loc}
  end

  defp pattern!({:_, _, ctx}, _scope) when is_atom(ctx), do: :any

  defp pattern!({name, _, ctx}, _scope) when is_atom(name) and is_atom(ctx) do
    if String.starts_with?(Atom.to_string(name), "_"), do: :any, else: {:bind, name}
  end

  defp pattern!({left, right}, scope), do: {:tuple, [pattern!(left, scope), pattern!(right, scope)]}
  defp pattern!({:{}, _, elements}, scope), do: {:tuple, Enum.map(elements, &pattern!(&1, scope))}
  defp pattern!(literal, _scope) when is_atom(literal) or is_integer(literal), do: {:lit, literal}

  defp pattern!(other, scope),
    do: Diagnostic.raise!(scope.loc, "#{scope.what}: #{Macro.to_string(other)} is outside the pattern subset (literals, _, variables, tuples)")

  # No rebinding: a name bound twice in one pattern would be an equality test on the BEAM and an
  # overwrite in `Umpire.Eval`, and a name that shadows the state or an input hides it.
  defp rebinding!(pattern, scope) do
    names = binding_list(pattern)
    taken = [scope.state_var | Map.keys(scope.inputs)]

    case {names -- Enum.uniq(names), Enum.filter(names, &(&1 in taken))} do
      {[], []} -> :ok
      {[twice | _], _} -> Diagnostic.raise!(scope.loc, "#{scope.what}: #{twice} is bound twice in one pattern; use a guard to compare")
      {[], [shadow | _]} -> Diagnostic.raise!(scope.loc, "#{scope.what}: the pattern rebinds #{shadow}, which the head already binds")
    end
  end

  defp binding_list({:bind, name}), do: [name]
  defp binding_list({:tuple, patterns}), do: Enum.flat_map(patterns, &binding_list/1)
  defp binding_list(_), do: []

  defp bindings({:bind, name}), do: MapSet.new([name])
  defp bindings({:tuple, patterns}), do: patterns |> Enum.map(&bindings/1) |> Enum.reduce(MapSet.new(), &MapSet.union/2)
  defp bindings(_), do: MapSet.new()

  # A step body: [], stay(), not_found() or moves(...). Nothing else.
  defp body!([], _scope), do: :disabled
  defp body!({:stay, _, []}, _scope), do: :stay
  defp body!({:not_found, _, []}, _scope), do: :not_found
  defp body!({:moves, meta, [phase, facts]}, scope), do: body!({:moves, meta, [phase, facts, []]}, scope)

  defp body!({:moves, _, [phase, facts, updates]}, scope) when is_list(facts) and is_list(updates) do
    fields = Keyword.keys(scope.state.fields)

    for {field, _} <- updates, field == :phase or field not in fields do
      Diagnostic.raise!(scope.loc, "#{scope.what}: moves/3 updates #{field}, which is not a field of #{inspect(scope.state.module)} other than phase")
    end

    {:moves, expr!(phase, scope), Enum.map(facts, &expr!(&1, scope)), Enum.map(updates, fn {f, v} -> {f, expr!(v, scope)} end)}
  end

  defp body!(other, scope) do
    Diagnostic.raise!(scope.loc, """
    #{scope.what}: #{Macro.to_string(other)} is not a step body.
      A body is [], stay(), not_found(), moves(phase, facts) or moves(phase, facts, field: value).\
    """)
  end

  # A map body: a struct literal of some state module, fields from the expression subset.
  defp map_body!({:%, _, [module, {:%{}, _, fields}]}, scope) when is_atom(module),
    do: {:state, module, Enum.map(fields, fn {f, v} -> {f, expr!(v, scope)} end)}

  defp map_body!(other, scope),
    do: Diagnostic.raise!(scope.loc, "#{scope.what}: a map body is a state struct literal, got: #{Macro.to_string(other)}")

  @doc false
  # The expression subset shared by guards, bodies and (with `roots`) property predicates.
  def expr!(ast, scope)

  def expr!(literal, _scope) when is_atom(literal) or is_integer(literal), do: {:lit, literal}
  def expr!({:@, _, [{name, _, ctx}]}, scope) when is_atom(ctx), do: {:lit, attribute!(scope.module, name, scope.loc)}
  def expr!(list, scope) when is_list(list), do: literal_or(list, Enum.map(list, &expr!(&1, scope)), scope)
  def expr!({left, right}, scope), do: literal_or({left, right}, {:tuple, [expr!(left, scope), expr!(right, scope)]}, scope)

  def expr!({op, _, [left, right]}, scope) when op in [:==, :!=, :and, :or, :in] do
    {%{==: :eq, !=: :neq, and: :and, or: :or, in: :in}[op], expr!(left, scope), expr!(right, scope)}
  end

  def expr!({:not, _, [{:in, _, [left, right]}]}, scope), do: {:not_in, expr!(left, scope), expr!(right, scope)}
  def expr!({:not, _, [operand]}, scope), do: {:not, expr!(operand, scope)}

  def expr!({:succ, _, [{{:., _, [{var, _, _}, field]}, _, []}]}, %Scope{state_var: var} = scope) do
    case Keyword.get(scope.state.fields, field) do
      %Range{last: max} -> {:succ, {:field, field}, max}
      other -> Diagnostic.raise!(scope.loc, "#{scope.what}: succ/1 needs a range field; #{field} has type #{inspect(other)}")
    end
  end

  def expr!({{:., _, [{var, _, ctx}, field]}, _, []}, %Scope{state_var: var} = scope) when is_atom(ctx) do
    if Keyword.has_key?(scope.state.fields, field),
      do: {:field, field},
      else: Diagnostic.raise!(scope.loc, "#{scope.what}: #{inspect(scope.state.module)} has no field #{field}" <> Diagnostic.suggest(field, Keyword.keys(scope.state.fields)))
  end

  def expr!({name, _, ctx} = var, scope) when is_atom(name) and is_atom(ctx) do
    cond do
      MapSet.member?(scope.bound, name) -> {:var, name}
      Map.has_key?(scope.inputs, name) -> {:input, Map.fetch!(scope.inputs, name)}
      Map.has_key?(scope.roots, name) -> {:path, scope.roots[name], []}
      true -> Diagnostic.raise!(scope.loc, "#{scope.what}: #{Macro.to_string(var)} is not bound by the pattern, the head, or the claim")
    end
  end

  def expr!({{:., _, [_, _]}, _, []} = access, %Scope{roots: roots} = scope) do
    case unwind(access, []) do
      {root, path} when is_map_key(roots, root) -> {:path, roots[root], path}
      _ -> outside!(access, scope)
    end
  end

  def expr!(other, scope), do: outside!(other, scope)

  defp unwind({{:., _, [inner, field]}, _, []}, path), do: unwind(inner, [field | path])
  defp unwind({root, _, ctx}, path) when is_atom(root) and is_atom(ctx), do: {root, path}
  defp unwind(_, _), do: :error

  # A tuple or list of literals is one literal; one with a variable in it is built at run time.
  defp literal_or(ast, lowered, _scope) do
    if Macro.quoted_literal?(ast), do: {:lit, elem(Code.eval_quoted(ast), 0)}, else: lowered
  end

  defp outside!(ast, scope) do
    what =
      case ast do
        # Aliases were expanded at the macro, so a remote call's module is an atom here.
        {{:., _, [mod, fun]}, _, args} when is_atom(mod) and is_atom(fun) -> "the call #{inspect(mod)}.#{fun}/#{length(args)}"
        {fun, _, args} when is_atom(fun) and is_list(args) and fun in @constructors -> "#{fun}/#{length(args)} outside a body position"
        {fun, _, args} when is_atom(fun) and is_list(args) -> "#{fun}/#{length(args)}"
        _ -> Macro.to_string(ast)
      end

    Diagnostic.raise!(scope.loc, """
    #{scope.what}: #{what} is outside the Umpire expression subset.
      Allowed: literals, bound variables, inputs, state fields, succ/1, tuples, lists, @attributes,
      ==, !=, in, not in, and, or, not. No function calls, sends or side effects.\
    """)
  end

  ## Properties

  @doc "Lower a `holds:` predicate. Paths are checked against the machine in `Umpire.Check`."
  def property!(module, name, machine, trigger, params, body, loc) do
    roots = params |> Enum.map(&elem(&1, 0)) |> Enum.with_index() |> Map.new()
    scope = %Scope{module: module, loc: loc, roots: roots, what: "defproperty #{inspect(name)}"}

    %IR.Property{
      name: name,
      machine: machine,
      kind: if(trigger, do: :same_step, else: :transition),
      when: trigger,
      fun: Umpire.Names.fun(name),
      holds: expr!(body, scope),
      loc: loc
    }
  end

  ## Emission

  # What an emitted body refers to: the state variable, and each input field's variable.
  defmodule Out do
    @moduledoc false
    defstruct [:state, inputs: %{}]
  end

  # The emitted function: the head pins the state struct and each input's domain, so the type
  # checker knows both; the body is the author's case with each arm's body lowered.
  defp emit(ir, body_ast, scope, emit_body) do
    inputs = Map.get(ir, :inputs, [])
    out = %Out{state: Macro.var(scope.state_var, nil), inputs: Map.new(inputs, fn {field, var} -> {field, Macro.var(var, nil)} end)}

    guard =
      inputs
      |> Enum.map(fn {field, _var} ->
        values = Domain.values(Map.fetch!(scope.input_types, field))
        quote(do: unquote(out.inputs[field]) in unquote(Macro.escape(values)))
      end)
      |> Enum.reduce(true, fn check, acc -> if acc == true, do: check, else: quote(do: unquote(acc) and unquote(check)) end)

    body =
      case body_ast do
        {:case, meta, [subject, [do: arms]]} ->
          arms =
            Enum.zip_with(arms, ir.clauses, fn {:->, arm_meta, [head, _body]}, clause ->
              {:->, arm_meta, [substitute(head, scope), emit_body.(clause.body, out)]}
            end)

          {:case, meta, [subject, [do: arms]]}

        _single ->
          emit_body.(hd(ir.clauses).body, out)
      end

    %{
      params: [quote(do: %unquote(scope.state.module){} = unquote(out.state)) | Enum.map(inputs, fn {f, _} -> out.inputs[f] end)],
      guard: guard,
      body: body
    }
  end

  defp substitute(ast, scope) do
    Macro.prewalk(ast, fn
      {:@, _, [{name, _, ctx}]} when is_atom(ctx) -> Macro.escape(attribute!(scope.module, name, scope.loc))
      other -> other
    end)
  end

  defp emit_body(:disabled, _out), do: []
  defp emit_body(:stay, out), do: quote(do: [%Umpire.Step{outcome: :accepted, state: unquote(out.state), facts: []}])
  defp emit_body(:not_found, out), do: quote(do: [%Umpire.Step{outcome: :notFound, state: unquote(out.state), facts: []}])

  defp emit_body({:moves, phase, facts, updates}, out) do
    fields = [{:phase, emit_expr(phase, out)} | Enum.map(updates, fn {f, v} -> {f, emit_expr(v, out)} end)]
    facts = Enum.map(facts, &emit_expr(&1, out))

    quote do
      [%Umpire.Step{outcome: :accepted, state: %{unquote(out.state) | unquote_splicing(fields)}, facts: unquote(facts)}]
    end
  end

  defp emit_map_body({:state, module, fields}, out),
    do: quote(do: %unquote(module){unquote_splicing(Enum.map(fields, fn {f, v} -> {f, emit_expr(v, out)} end))})

  defp emit_expr({:lit, value}, _out), do: Macro.escape(value)
  defp emit_expr({:var, name}, _out), do: Macro.var(name, nil)
  defp emit_expr({:input, field}, out), do: Map.fetch!(out.inputs, field)
  defp emit_expr({:field, field}, out), do: quote(do: unquote(out.state).unquote(field))
  defp emit_expr({:succ, inner, max}, out), do: quote(do: min(unquote(emit_expr(inner, out)) + 1, unquote(max)))
  defp emit_expr({:tuple, elements}, out), do: {:{}, [], Enum.map(elements, &emit_expr(&1, out))}
end
