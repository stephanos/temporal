use umpire_dsl::{Path, path};

fn main() {
    let _: Path<u8> = path![true, User::Send { amount: 0.., .. }];
    let _: Path<u8> = path![true, User::Poll =>];
    let _: Path<u8> = path![true, (User::Poll |)];
    let _: Path<u8> = path![true, [User::Poll; 0..]];
    let _: Path<u8> = path![true, unordered![]];
    let _: Path<u8> = path![true, ..!(), false];
    let _: Path<u8> = path![true, User::**];
    let _: Path<u8> = path![true, User::];
}
