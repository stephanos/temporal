# Extension tool controls

This test-only package retains repository Make freshness, embedded-guest refusal and historical retired-name policy controls. Reusable native commands live in `umpire/dev/xtask`; guest compilation and identities live in `umpire/dev/guest-driver`. The tests bootstrap current local native tools through the framework script before invoking them.
