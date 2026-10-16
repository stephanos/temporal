//go:build gomad_native

package testfixture

import (
	"go.temporal.io/server/tools/gomad/internal/backends/native"
	nativebuild "go.temporal.io/server/tools/gomad/internal/backends/native/build"
)

type Provider = native.Provider
type Config = native.Config

var New = native.New
var Prepare = nativebuild.Prepare
var ReadToolchainIdentity = nativebuild.ReadToolchainIdentity
var ProjectBuildInfo = nativebuild.ProjectBuildInfo
