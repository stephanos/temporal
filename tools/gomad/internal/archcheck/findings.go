package archcheck

type Platform struct{ OS, Arch string }

type Finding struct{ Category, Path, Detail string }
