package experiment

import (
	"io"

	"go.temporal.io/server/tools/umpire3/protocol/catalog"
	"go.temporal.io/server/tools/umpire3/protocol/internal/codec"
)

const FormatVersion = catalog.FormatVersion

type Catalog = catalog.Catalog
type CapabilityID = catalog.CapabilityID
type PropertyDeclaration = catalog.PropertyDeclaration
type ProjectionDeclaration = catalog.ProjectionDeclaration
type ActionDeclaration = catalog.ActionDeclaration
type ParameterDeclaration = catalog.ParameterDeclaration
type FaultDeclaration = catalog.FaultDeclaration

func DefaultCatalog() (catalog.Catalog, error) {
	return catalog.DefaultCatalog()
}

func DefaultComposition() (catalog.Composition, error) {
	return catalog.DefaultComposition()
}

func decodeStrictJSON(reader io.Reader, limit int64, kind string, destination any) error {
	return codec.DecodeStrictJSON(reader, limit, kind, destination)
}
