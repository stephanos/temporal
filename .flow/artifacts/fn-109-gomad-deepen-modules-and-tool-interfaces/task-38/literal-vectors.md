# Digest controls

The literals below were independently calculated with Python's hashlib from
the explicit framed input bytes before changing production source. The test
expectations do not call the digest implementation or its helpers. BASE test
execution confirms preservation; the failing regression is the actual lint
command, not a fabricated behavioral failure.

Overlay replacement bodies are `package a\n` and `package z\n`, with original
keys `/fn109/digest/a.go` and `/fn109/digest/z.go`. The reversed JSON order uses
`/fn109/digest/zz/../a.go`, which sorts after z before cleaning and before z
after cleaning. Original files are fictitious and need not exist. Replacement
files live under distinct real temporary directories, outside the digest.
The framed stream is:

```text
/fn109/digest/a.go\x007b39baa38a2ec2b8d111bbbd8e448e80226477ab40105d9d2123d4dc18067438\n
/fn109/digest/z.go\x00fc852e86c6ea2bc13f6521e13cfb58dd977aa91a136e37ad3ff5acc8a81170cb\n
```

The module bodies are `module example.com/digest\n\ngo 1.27.1\n` and
`example.com/dependency v1.0.0 h1:fixture\n`. Their inner hashes are
`7e82ea43ae84f8f139d38214b3e7436c88fc855ba2416e718fcaadf26471c767`
and `abfd050464244826295a39cd6ea76c3288ca81e58af1f2d1f280c7b6d372e76b`.
Each outer frame is basename + NUL + lowercase inner hex + newline, or
basename + NUL + `absent` + newline. The empty file inner digest is the
standard empty SHA-256, distinct from absence. Argument order stays significant.

| Input | Literal SHA-256 (with `sha256:` prefix in tests) |
| --- | --- |
| Empty stream | e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855 |
| Multiple overlay replacements | 1db3af338bd4366b249e4f3ae4bf2af122071bad27645a84d6f601516d1d25f6 |
| Present go.mod, present go.sum | cace9d14953fa84d0beac296d7937707921775c4c283615f329c0c4f23fb5674 |
| Present go.mod, absent go.sum | b9aa172d3a27d4116f21ae146eaf64403e30c4b9e19abb775fb48464dfee4d6d |
| Present go.mod, empty go.sum | c1c810346073420fefb509e1463b85b7d724297bbc13fca78fa1e71b72434c23 |
| Present go.sum followed by go.mod | 195265fcebde348a5b2bc830009fbab082ee679517c2ed47cd439af3e42c77d1 |
| Same module content named module.txt | 512c10c73c8891cdbc87e147a83fad61c078da38cff9653a8d337404c598b9be |
