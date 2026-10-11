package ir

import "go.temporal.io/server/tools/umpire/ir/paths"

const FeatureDirectory = paths.FeatureDirectory

func ModelPath(root, name string) string        { return paths.ModelPath(root, name) }
func ModelName(path string) string              { return paths.ModelName(path) }
func CasePath(root, model, query string) string { return paths.CasePath(root, model, query) }
func FeatureRoot(root string) string            { return paths.FeatureRoot(root) }
