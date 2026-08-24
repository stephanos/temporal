package monitor

import (
	"io"

	"go.temporal.io/server/tools/umpire3/protocol/catalog"
	"go.temporal.io/server/tools/umpire3/protocol/internal/codec"
)

const DefaultDecodeLimit = codec.DefaultDecodeLimit

type ObservationID = catalog.ObservationID
type PropertyID = catalog.PropertyID
type EvidenceID = catalog.EvidenceID
type PropertyDeclaration = catalog.PropertyDeclaration

func DefaultCatalog() (catalog.Catalog, error) {
	return catalog.DefaultCatalog()
}

func decodeStrictJSON(reader io.Reader, limit int64, kind string, destination any) error {
	return codec.DecodeStrictJSON(reader, limit, kind, destination)
}

func validHash(value string) bool {
	return codec.ValidHash(value)
}
