//! Emit an admitted Model (`umpire_ir::Lifted`) as a BTOR2 word-level transition system.
//!
//! Encoding: every state field is a bitvector state; a free input `act` chooses the catalog
//! action for the step (value `n` = no step); one-step history (`prev_*`, `prev_act`,
//! `has_step`) serves postconditions; evaluated arithmetic faults latch into `fault`.
//! Each property becomes one `bad`; the system is shared and one file is written per property.
use std::collections::BTreeMap;
use std::fmt::Write as _;
use umpire_ir::expr::{Expr, ExprKind, Op, Unary};
use umpire_ir::tree::{Branch, Start};
use umpire_ir::{Lifted, Type};

type Id = u64;

#[derive(Clone, Copy, Debug)]
pub struct Shape {
    pub width: u32,
    pub signed: bool,
}

/// A typed emitted value with its evaluation-fault flag.
#[derive(Clone, Copy)]
struct V {
    id: Id,
    shape: Shape,
    fault: Id,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Kind {
    Invariant,
    Postcondition,
    Negative,
    Arithmetic,
}

#[derive(Clone, Debug)]
pub struct Property {
    pub name: String,
    pub kind: Kind,
    bad: Id,
}

/// Where a decoded witness finds the Model: state and input declaration indices.
#[derive(Clone, Debug)]
pub struct Layout {
    pub fields: Vec<(String, Id, Shape, Type)>,
    pub world: Vec<(String, Id, Shape, Type)>,
    pub act: Id,
    pub actions: Vec<String>,
}

pub struct System {
    lines: Vec<String>,
    last: Id,
    pub properties: Vec<Property>,
    pub layout: Layout,
}

impl System {
    /// The BTOR2 text for one property.
    pub fn file(&self, property: &Property) -> String {
        let mut text = self.lines.join("\n");
        let _ = writeln!(text, "\n{} bad {}", self.last + 1, property.bad);
        text
    }
    pub fn shared_lines(&self) -> usize {
        self.lines.len()
    }
}

#[derive(Default)]
struct Emitter {
    lines: Vec<String>,
    /// Declared states and inputs in order: witnesses address them by this index.
    states: Vec<Id>,
    inputs: Vec<Id>,
    next: Id,
    sorts: BTreeMap<u32, Id>,
    consts: BTreeMap<(u32, u128), Id>,
}

impl Emitter {
    fn line(&mut self, body: &str) -> Id {
        self.next += 1;
        let id = self.next;
        self.lines.push(format!("{id} {body}"));
        id
    }
    fn sort(&mut self, width: u32) -> Id {
        if let Some(id) = self.sorts.get(&width) {
            return *id;
        }
        let id = self.line(&format!("sort bitvec {width}"));
        self.sorts.insert(width, id);
        id
    }
    fn constant(&mut self, shape: Shape, value: i128) -> Id {
        let mask = if shape.width >= 128 { u128::MAX } else { (1u128 << shape.width) - 1 };
        let bits = (value as u128) & mask;
        if let Some(id) = self.consts.get(&(shape.width, bits)) {
            return *id;
        }
        let sort = self.sort(shape.width);
        let text = format!("{bits:0width$b}", width = shape.width as usize);
        let id = self.line(&format!("const {sort} {text}"));
        self.consts.insert((shape.width, bits), id);
        id
    }
    fn bool(&mut self, value: bool) -> Id {
        self.constant(BOOL, i128::from(value))
    }
    fn op1(&mut self, name: &str, width: u32, a: Id) -> Id {
        let sort = self.sort(width);
        self.line(&format!("{name} {sort} {a}"))
    }
    fn op2(&mut self, name: &str, width: u32, a: Id, b: Id) -> Id {
        let sort = self.sort(width);
        self.line(&format!("{name} {sort} {a} {b}"))
    }
    fn extend(&mut self, how: &str, a: Id, width: u32, extra: u32) -> Id {
        let sort = self.sort(width + extra);
        let name = if how == "ext_signed" { "sext" } else { "uext" };
        self.line(&format!("{name} {sort} {a} {extra}"))
    }
    fn ite(&mut self, width: u32, c: Id, a: Id, b: Id) -> Id {
        let sort = self.sort(width);
        self.line(&format!("ite {sort} {c} {a} {b}"))
    }
    fn and(&mut self, a: Id, b: Id) -> Id {
        self.op2("and", 1, a, b)
    }
    fn or(&mut self, a: Id, b: Id) -> Id {
        self.op2("or", 1, a, b)
    }
    fn not(&mut self, a: Id) -> Id {
        self.op1("not", 1, a)
    }
    fn all(&mut self, ids: &[Id]) -> Id {
        match ids {
            [] => self.bool(true),
            [first, rest @ ..] => rest.iter().fold(*first, |acc, id| self.and(acc, *id)),
        }
    }
    fn any(&mut self, ids: &[Id]) -> Id {
        match ids {
            [] => self.bool(false),
            [first, rest @ ..] => rest.iter().fold(*first, |acc, id| self.or(acc, *id)),
        }
    }
    fn state(&mut self, width: u32, name: &str) -> Id {
        let sort = self.sort(width);
        let id = self.line(&format!("state {sort} {name}"));
        self.states.push(id);
        id
    }
    fn input(&mut self, width: u32, name: &str) -> Id {
        let sort = self.sort(width);
        let id = self.line(&format!("input {sort} {name}"));
        self.inputs.push(id);
        id
    }
    fn init(&mut self, width: u32, state: Id, value: Id) {
        let sort = self.sort(width);
        self.line(&format!("init {sort} {state} {value}"));
    }
    fn next(&mut self, width: u32, state: Id, value: Id) {
        let sort = self.sort(width);
        self.line(&format!("next {sort} {state} {value}"));
    }
    fn constraint(&mut self, cond: Id) {
        self.line(&format!("constraint {cond}"));
    }
    fn comment(&mut self, text: &str) {
        self.lines.push(format!("; {text}"));
    }
}

const BOOL: Shape = Shape { width: 1, signed: false };

pub fn shape(ty: &Type) -> Result<Shape, String> {
    match ty {
        Type::Bool => Ok(BOOL),
        Type::Int { min, max } => {
            for width in [8u32, 16, 32, 64, 128] {
                let signed_min = -(1i128 << (width - 1));
                let signed_max = (1i128 << (width - 1)) - 1;
                if *min >= 0
                    && *max <= (if width >= 128 { i128::MAX } else { (1i128 << width) - 1 })
                    && *max > signed_max
                {
                    return Ok(Shape { width, signed: false });
                }
                if *min >= signed_min && *max <= signed_max {
                    return Ok(Shape { width, signed: true });
                }
            }
            Err(format!("integer range {min}..={max} has no bitvector width"))
        }
        Type::Enum { variants, .. } => {
            let width = (usize::BITS - variants.len().saturating_sub(1).leading_zeros()).max(1);
            Ok(Shape { width, signed: false })
        }
        other => Err(format!("type {other:?} is not word-level")),
    }
}

fn range(ty: &Type) -> Option<(i128, i128)> {
    match ty {
        Type::Int { min, max } => Some((*min, *max)),
        _ => None,
    }
}

fn enum_index(ty: &Type, variant: &str) -> Result<i128, String> {
    match ty {
        Type::Enum { variants, .. } => variants
            .iter()
            .position(|v| v == variant)
            .map(|i| i as i128)
            .ok_or_else(|| format!("unknown variant {variant} of {ty:?}")),
        _ => Err(format!("enum literal {variant} with non-enum type {ty:?}")),
    }
}

/// Which state each unprimed and primed variable reads.
struct Env<'a> {
    current: &'a BTreeMap<String, V>,
    before: Option<&'a BTreeMap<String, V>>,
    world: &'a BTreeMap<String, V>,
    locals: BTreeMap<String, V>,
}

impl Emitter {
    fn expr(&mut self, e: &Expr, env: &Env<'_>) -> Result<V, String> {
        let no_fault = self.bool(false);
        Ok(match &e.kind {
            ExprKind::Bool(b) => V { id: self.bool(*b), shape: BOOL, fault: no_fault },
            ExprKind::Int(v) => {
                let shape = shape(&e.ty)?;
                V { id: self.constant(shape, *v), shape, fault: no_fault }
            }
            ExprKind::Enum(variant) => {
                let shape = shape(&e.ty)?;
                let index = enum_index(&e.ty, variant)?;
                V { id: self.constant(shape, index), shape, fault: no_fault }
            }
            ExprKind::Var(name, primed) => {
                let map = if *primed { env.current } else { env.before.unwrap_or(env.current) };
                *map.get(name).ok_or_else(|| format!("unknown state field {name}"))?
            }
            ExprKind::World(name) => {
                *env.world.get(name).ok_or_else(|| format!("unknown World field {name}"))?
            }
            ExprKind::Local(name) => {
                *env.locals.get(name).ok_or_else(|| format!("unknown local {name}"))?
            }
            ExprKind::Unary(Unary::Not, inner) => {
                let v = self.expr(inner, env)?;
                V { id: self.not(v.id), shape: BOOL, fault: v.fault }
            }
            ExprKind::Unary(Unary::Neg, inner) => {
                let v = self.expr(inner, env)?;
                let id = self.op1("neg", v.shape.width, v.id);
                let min = self.constant(v.shape, -(1i128 << (v.shape.width - 1)));
                let overflow = self.op2("eq", 1, v.id, min);
                let fault = self.or(v.fault, overflow);
                V { id, shape: v.shape, fault }
            }
            ExprKind::Binary(op, left, right) => self.binary(*op, left, right, env)?,
            ExprKind::If(c, yes, no) => {
                let c = self.expr(c, env)?;
                let y = self.expr(yes, env)?;
                let n = self.expr(no, env)?;
                let id = self.ite(y.shape.width, c.id, y.id, n.id);
                let arm = self.ite(1, c.id, y.fault, n.fault);
                let fault = self.or(c.fault, arm);
                V { id, shape: y.shape, fault }
            }
            ExprKind::Check(_, inner) => self.expr(inner, env)?,
            ExprKind::Checks(checks) => {
                let mut ids = Vec::new();
                let mut faults = Vec::new();
                for check in checks {
                    let v = self.expr(check, env)?;
                    ids.push(v.id);
                    faults.push(v.fault);
                }
                let id = self.all(&ids);
                let fault = self.any(&faults);
                V { id, shape: BOOL, fault }
            }
            ExprKind::Unit => V { id: self.bool(true), shape: BOOL, fault: no_fault },
            other => return Err(format!("unsupported expression in the spike: {other:?}")),
        })
    }

    fn binary(&mut self, op: Op, left: &Expr, right: &Expr, env: &Env<'_>) -> Result<V, String> {
        let a = self.expr(left, env)?;
        let b = self.expr(right, env)?;
        let s = a.shape;
        let w = s.width;
        let both = self.or(a.fault, b.fault);
        Ok(match op {
            Op::And => {
                let id = self.and(a.id, b.id);
                let taken = self.and(a.id, b.fault);
                let fault = self.or(a.fault, taken);
                V { id, shape: BOOL, fault }
            }
            Op::Or => {
                let id = self.or(a.id, b.id);
                let na = self.not(a.id);
                let taken = self.and(na, b.fault);
                let fault = self.or(a.fault, taken);
                V { id, shape: BOOL, fault }
            }
            Op::Eq => V { id: self.op2("eq", 1, a.id, b.id), shape: BOOL, fault: both },
            Op::Ne => V { id: self.op2("neq", 1, a.id, b.id), shape: BOOL, fault: both },
            Op::Lt | Op::Le | Op::Gt | Op::Ge => {
                let name = match (op, s.signed) {
                    (Op::Lt, true) => "slt",
                    (Op::Le, true) => "slte",
                    (Op::Gt, true) => "sgt",
                    (Op::Ge, true) => "sgte",
                    (Op::Lt, false) => "ult",
                    (Op::Le, false) => "ulte",
                    (Op::Gt, false) => "ugt",
                    _ => "ugte",
                };
                V { id: self.op2(name, 1, a.id, b.id), shape: BOOL, fault: both }
            }
            Op::Add | Op::Sub | Op::Mul => {
                // rIC3's logic library has no saddo/ssubo/smulo: widen, operate, and compare
                // with the extended narrow result; a difference is a Rust overflow.
                let name = match op {
                    Op::Add => "add",
                    Op::Sub => "sub",
                    _ => "mul",
                };
                let id = self.op2(name, w, a.id, b.id);
                let extra = if op == Op::Mul { w } else { 1 };
                let wide = w + extra;
                let ext = if s.signed { "ext_signed" } else { "ext_unsigned" };
                let wa = self.extend(ext, a.id, w, extra);
                let wb = self.extend(ext, b.id, w, extra);
                let wr = self.op2(name, wide, wa, wb);
                let narrow = self.extend(ext, id, w, extra);
                let o = self.op2("neq", 1, wr, narrow);
                let fault = self.or(both, o);
                V { id, shape: s, fault }
            }
            Op::Div | Op::Rem => {
                let name = match (op, s.signed) {
                    (Op::Div, true) => "sdiv",
                    (Op::Rem, true) => "srem",
                    (Op::Div, false) => "udiv",
                    _ => "urem",
                };
                let id = self.op2(name, w, a.id, b.id);
                let zero = self.constant(s, 0);
                let by_zero = self.op2("eq", 1, b.id, zero);
                let mut fault = self.or(both, by_zero);
                if s.signed {
                    let min = self.constant(s, -(1i128 << (w - 1)));
                    let minus_one = self.constant(s, -1);
                    let is_min = self.op2("eq", 1, a.id, min);
                    let is_minus_one = self.op2("eq", 1, b.id, minus_one);
                    let o = self.and(is_min, is_minus_one);
                    fault = self.or(fault, o);
                }
                V { id, shape: s, fault }
            }
        })
    }
}

pub fn emit(model: &Lifted, negative: &[(String, Expr)]) -> Result<System, String> {
    let mut e = Emitter::default();
    e.comment(&format!("Umpire Model {} as a BTOR2 transition system (spike)", model.name));
    let zero1 = e.bool(false);
    let one1 = e.bool(true);

    // World constants: free states that never change, inside their declared domain.
    let mut world = BTreeMap::new();
    let mut world_layout = Vec::new();
    for field in &model.world_fields {
        let s = shape(&field.ty)?;
        let id = e.state(s.width, &format!("world_{}", field.name));
        e.next(s.width, id, id);
        domain(&mut e, id, s, &field.ty)?;
        world.insert(field.name.clone(), V { id, shape: s, fault: zero1 });
        world_layout.push((field.name.clone(), id, s, field.ty.clone()));
    }

    // State fields and the one-step history.
    let mut current = BTreeMap::new();
    let mut before = BTreeMap::new();
    let mut fields = Vec::new();
    let mut prev_ids = Vec::new();
    for field in &model.fields {
        let s = shape(&field.ty)?;
        let id = e.state(s.width, &field.name);
        let prev = e.state(s.width, &format!("prev_{}", field.name));
        domain(&mut e, id, s, &field.ty)?;
        current.insert(field.name.clone(), V { id, shape: s, fault: zero1 });
        before.insert(field.name.clone(), V { id: prev, shape: s, fault: zero1 });
        fields.push((field.name.clone(), id, s, field.ty.clone()));
        prev_ids.push((prev, id, s));
    }
    let started = e.state(1, "started");
    e.init(1, started, one1);
    e.next(1, started, zero1);
    let has_step = e.state(1, "has_step");
    e.init(1, has_step, zero1);
    let fault = e.state(1, "fault");
    e.init(1, fault, zero1);

    // Starts.
    let env = Env { current: &current, before: None, world: &world, locals: BTreeMap::new() };
    for field in &model.fields {
        let v = current[&field.name];
        let cond = match &field.start {
            Start::Value(value) => {
                let value = e.expr(value, &env)?;
                e.op2("eq", 1, v.id, value.id)
            }
            Start::Range(lo, hi) | Start::ExclusiveRange(lo, hi) => {
                let lo = e.expr(lo, &env)?;
                let hi = e.expr(hi, &env)?;
                let (ge, le, lt) =
                    if v.shape.signed { ("sgte", "slte", "slt") } else { ("ugte", "ulte", "ult") };
                let above = e.op2(ge, 1, v.id, lo.id);
                let below = e.op2(
                    if matches!(field.start, Start::Range(..)) { le } else { lt },
                    1,
                    v.id,
                    hi.id,
                );
                e.and(above, below)
            }
            Start::Computed => {
                return Err(format!("computed starts ({}) are outside the spike", field.name));
            }
        };
        let implied = e.op2("implies", 1, started, cond);
        e.constraint(implied);
    }

    // The chosen action.
    let n = model.actions.len();
    let act_shape = Shape { width: (usize::BITS - n.leading_zeros()).max(1), signed: false };
    let act = e.input(act_shape.width, "act");
    let stutter = e.constant(act_shape, n as i128);
    let in_range = e.op2("ulte", 1, act, stutter);
    e.constraint(in_range);
    let prev_act = e.state(act_shape.width, "prev_act");
    e.next(act_shape.width, prev_act, act);

    // Actions: first-match branches; an action is enabled when its taken branch changes the
    // state (or preserves a self-loop); evaluated faults latch regardless of the chosen action,
    // as the native checker evaluates every action at every reached state.
    let mut selected = Vec::new();
    let mut nexts: Vec<Vec<(Id, Id)>> = Vec::new(); // per action: (field id, next value)
    let mut all_faults = vec![fault];
    for (index, action) in model.actions.iter().enumerate() {
        e.comment(&format!("action {index}: {}", action.name));
        if !action.fields.is_empty() {
            return Err(format!("action inputs ({}) are outside the spike", action.name));
        }
        let mut earlier_guards = Vec::new();
        let mut enabled_terms = Vec::new();
        let mut field_next: Vec<(Id, Id)> = fields.iter().map(|(_, id, _, _)| (*id, *id)).collect();
        for branch in &action.branches {
            let (taken, updates, changed, bfault) =
                emit_branch(&mut e, branch, &current, &world, &earlier_guards, &fields)?;
            earlier_guards.push(branch.guard.clone());
            let preserve = e.bool(branch.preserve_self_loop);
            let edge = e.or(changed, preserve);
            let enabled = e.and(taken, edge);
            enabled_terms.push(enabled);
            let taken_fault = e.and(taken, bfault);
            all_faults.push(taken_fault);
            for (slot, (fid, next)) in field_next.iter_mut().enumerate() {
                if let Some(update) = updates[slot] {
                    let width = fields[slot].2.width;
                    *next = e.ite(width, taken, update, *next);
                }
                let _ = fid;
            }
        }
        let enabled = e.any(&enabled_terms);
        let this = e.constant(act_shape, index as i128);
        let chosen = e.op2("eq", 1, act, this);
        let sel = e.and(chosen, enabled);
        selected.push(sel);
        nexts.push(field_next);
    }
    let stepped = e.any(&selected);
    e.next(1, has_step, stepped);
    for (slot, (name, id, s, _)) in fields.iter().enumerate() {
        let mut value = *id;
        for (action, sel) in selected.iter().enumerate().rev() {
            value = e.ite(s.width, *sel, nexts[action][slot].1, value);
        }
        e.next(s.width, *id, value);
        let _ = name;
    }
    for (prev, id, s) in &prev_ids {
        e.next(s.width, *prev, *id);
    }

    // Properties.
    let mut properties = Vec::new();
    let mut claim_faults = Vec::new();
    let invariant_env =
        Env { current: &current, before: None, world: &world, locals: BTreeMap::new() };
    for (kind, claims) in [(Kind::Invariant, &model.invariants), (Kind::Negative, &model.negative)]
    {
        for claim in claims {
            let v = e.expr(&claim.expr, &invariant_env)?;
            claim_faults.push(v.fault);
            let bad = e.not(v.id);
            properties.push(Property {
                name: claim.name.clone(),
                kind,
                bad: pending_bad(&mut e, bad),
            });
        }
    }
    for (name, expr) in negative {
        let v = e.expr(expr, &invariant_env)?;
        claim_faults.push(v.fault);
        let bad = e.not(v.id);
        properties.push(Property {
            name: name.clone(),
            kind: Kind::Negative,
            bad: pending_bad(&mut e, bad),
        });
    }
    let post_env =
        Env { current: &current, before: Some(&before), world: &world, locals: BTreeMap::new() };
    for post in &model.postconditions {
        let v = e.expr(&post.expr, &post_env)?;
        let mut cond = has_step;
        if let Some(after) = &post.after {
            let index = model
                .actions
                .iter()
                .position(|a| &a.name == after)
                .ok_or_else(|| format!("unknown after action {after}"))?;
            let this = e.constant(act_shape, index as i128);
            let is_after = e.op2("eq", 1, prev_act, this);
            cond = e.and(cond, is_after);
        }
        let guarded_fault = e.and(has_step, v.fault);
        claim_faults.push(guarded_fault);
        let failed = e.not(v.id);
        let bad = e.and(cond, failed);
        properties.push(Property {
            name: post.name.clone(),
            kind: Kind::Postcondition,
            bad: pending_bad(&mut e, bad),
        });
    }
    all_faults.extend(claim_faults);
    let next_fault = e.any(&all_faults);
    e.next(1, fault, next_fault);
    properties.push(Property {
        name: "arithmetic".into(),
        kind: Kind::Arithmetic,
        bad: pending_bad(&mut e, fault),
    });

    let index = |ids: &[Id], id: Id| {
        ids.iter()
            .position(|x| *x == id)
            .map(|i| i as Id)
            .ok_or_else(|| format!("node {id} is not a declared state or input"))
    };
    let mut indexed = Vec::new();
    for (name, id, s, ty) in fields {
        indexed.push((name, index(&e.states, id)?, s, ty));
    }
    let mut indexed_world = Vec::new();
    for (name, id, s, ty) in world_layout {
        indexed_world.push((name, index(&e.states, id)?, s, ty));
    }
    let act = index(&e.inputs, act)?;
    Ok(System {
        last: e.next,
        lines: e.lines,
        properties,
        layout: Layout {
            fields: indexed,
            world: indexed_world,
            act,
            actions: model.actions.iter().map(|a| a.name.clone()).collect(),
        },
    })
}

/// The `bad` condition; its line is written per property after the shared text.
fn pending_bad(_: &mut Emitter, cond: Id) -> Id {
    cond
}

fn domain(e: &mut Emitter, id: Id, s: Shape, ty: &Type) -> Result<(), String> {
    match ty {
        Type::Int { .. } => {
            let (min, max) = range(ty).ok_or("int")?;
            let lo = e.constant(s, min);
            let hi = e.constant(s, max);
            let (ge, le) = if s.signed { ("sgte", "slte") } else { ("ugte", "ulte") };
            let above = e.op2(ge, 1, id, lo);
            let below = e.op2(le, 1, id, hi);
            let inside = e.and(above, below);
            e.constraint(inside);
        }
        Type::Enum { variants, .. } => {
            let count = e.constant(s, variants.len() as i128);
            let inside = e.op2("ult", 1, id, count);
            e.constraint(inside);
        }
        Type::Bool => {}
        other => return Err(format!("field type {other:?} is not word-level")),
    }
    Ok(())
}

type BranchOut = (Id, Vec<Option<Id>>, Id, Id);

fn emit_branch(
    e: &mut Emitter,
    branch: &Branch,
    current: &BTreeMap<String, V>,
    world: &BTreeMap<String, V>,
    earlier: &[Expr],
    fields: &[(String, Id, Shape, Type)],
) -> Result<BranchOut, String> {
    let mut env = Env { current, before: None, world, locals: BTreeMap::new() };
    let mut faults = Vec::new();
    for (name, value) in &branch.definitions {
        let v = e.expr(value, &env)?;
        faults.push(v.fault);
        env.locals.insert(name.clone(), v);
    }
    let guard = e.expr(&branch.guard, &env)?;
    let mut not_earlier = Vec::new();
    for g in earlier {
        let g = e.expr(g, &env)?;
        not_earlier.push(e.not(g.id));
    }
    let earlier_failed = e.all(&not_earlier);
    let taken = e.and(guard.id, earlier_failed);
    let mut updates = Vec::new();
    let mut changed = Vec::new();
    let mut body_faults = Vec::new();
    for (name, id, _, _) in fields {
        match branch.updates.get(name) {
            Some(update) => {
                let v = e.expr(update, &env)?;
                body_faults.push(v.fault);
                let differs = e.op2("neq", 1, v.id, *id);
                changed.push(differs);
                updates.push(Some(v.id));
            }
            None => updates.push(None),
        }
    }
    for check in &branch.checks {
        let v = e.expr(check, &env)?;
        body_faults.push(v.fault);
    }
    if branch.outcome.kind != ExprKind::Unit {
        return Err("non-unit outcomes are outside the spike".into());
    }
    let changed = e.any(&changed);
    let body_fault = e.any(&body_faults);
    let taken_body = e.and(taken, body_fault);
    let defs = e.any(&faults);
    let fault = [guard.fault, taken_body, defs];
    let fault = e.any(&fault);
    Ok((taken, updates, changed, fault))
}

/// Sketch of the text for a report: how many lines the shared system has.
pub fn describe(system: &System) -> String {
    let mut out = String::new();
    let _ = writeln!(out, "shared BTOR2 lines: {}", system.shared_lines());
    let _ = writeln!(out, "properties: {}", system.properties.len());
    out
}
