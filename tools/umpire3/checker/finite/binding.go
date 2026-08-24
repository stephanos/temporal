package finite

import (
	"fmt"

	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
)

func BindingSource(view protocolchecker.FirstOrderView) ([]byte, error) {
	if err := view.Validate(); err != nil {
		return nil, err
	}
	if view.Target != protocolcatalog.TargetIDNexusCancellation || view.Variant != "sound" {
		return nil, fmt.Errorf("native certificate checker does not support %q/%q", view.Target, view.Variant)
	}
	viewDigest, err := firstOrderViewDigest(view)
	if err != nil {
		return nil, err
	}
	return fmt.Appendf(nil, `namespace Umpire3.Generated.NexusCertificateBinding

def semanticHash : String :=
  %q

def viewDigest : String :=
  %q

end Umpire3.Generated.NexusCertificateBinding
`, view.SemanticHash, viewDigest), nil
}
