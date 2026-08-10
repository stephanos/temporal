package translate

type stdlibCompatibilityPolicy struct {
	targetGoVersion         string
	hooks                   map[packageSelector]packageSelector
	hooksByArch             map[string]map[packageSelector]packageSelector
	skippedPackages         map[string]bool
	keepAsmPackages         map[string]bool
	acceptedLinknames       map[packageSelector]packageSelector
	acceptedNoBodyLinknames map[packageSelector]bool
	noBodyLinknameTargets   map[packageSelector]packageSelector
	globalsDontTranslate    map[packageSelector]bool
}

var activeStdlibPolicy = &go126CompatibilityPolicy

var generatedStdlibHooksByArch = map[string]map[packageSelector]packageSelector{}
