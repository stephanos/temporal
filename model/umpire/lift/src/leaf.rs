use crate::{
    Type,
    tree::{Op, Unary},
};

pub(crate) fn scalar_type(name: &str) -> Option<Type> {
    let (min, max) = match name {
        "bool" => return Some(Type::Bool),
        "i8" => (i128::from(i8::MIN), i128::from(i8::MAX)),
        "i16" => (i128::from(i16::MIN), i128::from(i16::MAX)),
        "i32" => (i128::from(i32::MIN), i128::from(i32::MAX)),
        "i64" => (i128::from(i64::MIN), i128::from(i64::MAX)),
        "u8" => (0, i128::from(u8::MAX)),
        "u16" => (0, i128::from(u16::MAX)),
        "u32" => (0, i128::from(u32::MAX)),
        "u64" => (0, i128::from(u64::MAX)),
        _ => return None,
    };
    Some(Type::Int { min, max })
}
pub(crate) fn binary_op(op: &syn::BinOp) -> Option<Op> {
    Some(match op {
        syn::BinOp::Add(_) => Op::Add,
        syn::BinOp::Sub(_) => Op::Sub,
        syn::BinOp::Mul(_) => Op::Mul,
        syn::BinOp::Div(_) => Op::Div,
        syn::BinOp::Rem(_) => Op::Rem,
        syn::BinOp::And(_) => Op::And,
        syn::BinOp::Or(_) => Op::Or,
        syn::BinOp::Eq(_) => Op::Eq,
        syn::BinOp::Ne(_) => Op::Ne,
        syn::BinOp::Lt(_) => Op::Lt,
        syn::BinOp::Le(_) => Op::Le,
        syn::BinOp::Gt(_) => Op::Gt,
        syn::BinOp::Ge(_) => Op::Ge,
        _ => return None,
    })
}
pub(crate) fn unary_op(op: &syn::UnOp) -> Option<Unary> {
    match op {
        syn::UnOp::Not(_) => Some(Unary::Not),
        syn::UnOp::Neg(_) => Some(Unary::Neg),
        _ => None,
    }
}
