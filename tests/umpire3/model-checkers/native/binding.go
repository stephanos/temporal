package native

import (
	"fmt"

	"go.temporal.io/server/tests/umpire3/protocol"
)

func BindingSource(view protocol.FirstOrderView) ([]byte, error) {
	if err := view.Validate(); err != nil {
		return nil, err
	}
	if view.Target != protocol.TargetIDNexusCancellation || view.Variant != "sound" {
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
