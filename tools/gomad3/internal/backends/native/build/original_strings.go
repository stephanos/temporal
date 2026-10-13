package build

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad3/internal/backends/target"
)

type originalString struct {
	field string
	value string
}

func validateOriginalStrings(values ...originalString) error {
	for _, value := range values {
		if !utf8.ValidString(value.value) {
			return fmt.Errorf("%s must be valid UTF-8", value.field)
		}
	}
	return nil
}

func encodePreparedTargetIdentity(identity preparedTargetIdentity) ([]byte, error) {
	if err := validatePreparedTargetIdentityStrings(identity); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(identity)
	if err != nil {
		return nil, fmt.Errorf("encode prepared target identity: %w", err)
	}
	return encoded, nil
}

func validatePreparedTargetIdentityStrings(identity preparedTargetIdentity) error {
	values := []originalString{
		{"prepared target identity schema", identity.Schema},
		{"prepared target Go version", identity.GoVersion},
		{"prepared target build key", identity.BuildKey},
		{"prepared target GOOS", identity.TargetGOOS},
		{"prepared target GOARCH", identity.TargetGOARCH},
		{"prepared target binary Go version", identity.BinaryVersion},
		{"prepared target kind", string(identity.Kind)},
		{"prepared target package", identity.Package},
		{"prepared target capability mode", string(identity.CapabilityMode)},
		{"prepared target overlay hash", string(identity.Overlay)},
		{"prepared target modfile hash", string(identity.ModFile)},
		{"prepared target module files hash", string(identity.ModuleFiles)},
		{"prepared target closure hash", string(identity.Closure)},
	}
	for _, value := range identity.Environment {
		values = append(values, originalString{"prepared target environment", value})
	}
	for _, value := range identity.BuildTags {
		values = append(values, originalString{"prepared target build tag", value})
	}
	for _, source := range identity.Embedded {
		values = append(values,
			originalString{"prepared target embedded import path", source.ImportPath},
			originalString{"prepared target embedded test variant", source.ForTest},
			originalString{"prepared target embedded name", source.Name},
			originalString{"prepared target embedded hash", string(source.SHA256)},
		)
	}
	for _, module := range identity.Modules {
		values = append(values,
			originalString{"prepared target module path", module.Path},
			originalString{"prepared target module version", module.Version},
			originalString{"prepared target module Go version", module.GoVersion},
			originalString{"prepared target module go.mod hash", string(module.GoMod)},
		)
	}
	return validateOriginalStrings(values...)
}

func encodeCapabilityClosure(closure target.CapabilityClosure) ([]byte, error) {
	if err := validateCapabilityClosureStrings(closure); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(closure)
	if err != nil {
		return nil, fmt.Errorf("encode reviewed closure: %w", err)
	}
	return encoded, nil
}

func validateCapabilityClosureStrings(closure target.CapabilityClosure) error {
	values := []originalString{{"capability closure schema", closure.Schema}}
	for _, compatibility := range closure.Compatibility {
		values = append(values,
			originalString{"capability closure compatibility ID", compatibility.ID},
			originalString{"capability closure compatibility hash", compatibility.SHA256},
		)
	}
	seenModules := map[*target.CapabilityModule]struct{}{}
	for _, pkg := range closure.Packages {
		values = append(values,
			originalString{"capability closure package import path", pkg.ImportPath},
			originalString{"capability closure package test variant", pkg.ForTest},
			originalString{"capability closure package name", pkg.Name},
		)
		for _, imported := range pkg.Imports {
			values = append(values, originalString{"capability closure package import", imported})
		}
		appendCapabilityModuleStrings(&values, pkg.Module, seenModules)
		for _, source := range pkg.Sources {
			values = append(values,
				originalString{"capability closure source name", source.Name},
				originalString{"capability closure source hash", source.SHA256},
			)
			for _, directive := range source.LinknameDirectives {
				values = append(values, originalString{"capability closure linkname directive", directive})
			}
		}
		for _, source := range pkg.ForeignSources {
			values = append(values,
				originalString{"capability closure foreign source kind", source.Kind},
				originalString{"capability closure foreign source name", source.Name},
				originalString{"capability closure foreign source hash", source.SHA256},
			)
		}
	}
	return validateOriginalStrings(values...)
}

func appendCapabilityModuleStrings(values *[]originalString, module *target.CapabilityModule, seen map[*target.CapabilityModule]struct{}) {
	if module == nil {
		return
	}
	if _, ok := seen[module]; ok {
		return
	}
	seen[module] = struct{}{}
	*values = append(*values,
		originalString{"capability closure module path", module.Path},
		originalString{"capability closure module version", module.Version},
		originalString{"capability closure module sum", module.Sum},
	)
	appendCapabilityModuleStrings(values, module.Replacement, seen)
	if module.Adapter == nil {
		return
	}
	adapter := module.Adapter
	*values = append(*values,
		originalString{"capability closure adapter profile", adapter.ProfileName},
		originalString{"capability closure adapter profile hash", adapter.ProfileImplementationSHA256},
		originalString{"capability closure adapter module path", adapter.Adapter.Path},
		originalString{"capability closure adapter module version", adapter.Adapter.Version},
		originalString{"capability closure adapter module sum", adapter.Adapter.Sum},
		originalString{"capability closure adapter original source inventory hash", adapter.OriginalSourceInventorySHA256},
		originalString{"capability closure adapter replacement source inventory hash", adapter.ReplacementSourceInventorySHA256},
		originalString{"capability closure adapter prepared source set hash", adapter.PreparedSourceSetSHA256},
	)
}
