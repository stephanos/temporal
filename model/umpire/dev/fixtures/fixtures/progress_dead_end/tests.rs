use crate::progress_dead_end::{ProgressDeadEnd, kettle};

umpire_testing::fixture!(ProgressDeadEnd(kettle::world()), 16);
