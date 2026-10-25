# System workspace tool bridge

`cargo xtask` from the outer Model workspace forwards to the independent framework
CLI at `umpire/dev/xtask`. Guest commands execute the native framework CLI directly: the five
framework guest packages compile in `umpire/`, while other packages retain the outer
workspace. Output and target paths resolve before changing directories. Compiler
wrapping and cache admission stay in the dedicated framework guest driver.

This package also retains repository Make freshness, embedded-guest refusal and
historical retired-name controls. They bootstrap current local native tools through
the framework script before invoking them.
