//go:build !gomadv3_toolchain

package gomadv3sim

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeDomainBridgeFingerprints(t *testing.T) {
	files := token.NewFileSet()
	want := map[string]string{
		"runtimeDomainAvailable":         "sha256:e1f2f00d0f21e11d688551c6d77f5bc9abea677cf0805fea7125cf02723ef090",
		"runtimeDomainBegin":             "sha256:841134e65475799cf130d4127beacad19c09b69d09df02b964435ae7027b13e8",
		"runtimeDomainRegister":          "sha256:a4d0f95dc6fd33aa78092805005475dd7ad18c272d6a586b16ea727f368a8a5e",
		"runtimeDomainEnter":             "sha256:d11ef41916402a6082de54e8cd41346e72ec5bf68a4daab01145d7618a783a1d",
		"runtimeDomainLeave":             "sha256:21ec316a2b18610e9e946903c7d54cc0e7be105df3526162c0c224d28344608d",
		"runtimeDomainRevoke":            "sha256:aa9b9a4a8617f833d1f99db1be300ca1d9bf4919fb22274a7514e3c5d38aa8c9",
		"runtimeDomainFinish":            "sha256:0521b81eda5b875ee982b88f1266528ce9b9a3236b104d9e072bb35dd71dc807",
		"runtimeNetworkBegin":            "sha256:c6a522160ea85057c83e2116d092adf2e957e37976365246df048fd8318ce90c",
		"runtimeNetworkPartition":        "sha256:8f16017f6bcef0f6f39408501258d1ff4510b6eb375eb818fde758214241a1a5",
		"runtimeNetworkHeal":             "sha256:9ad06c2e24cf20da1ad412695d9e1a7f8c5016c9627b5c9901a6c4dbff2ef664",
		"runtimeNetworkDelay":            "sha256:e4b66ced616c6d837f443f8c304ee5b87777cb9f1fba7084f15696b9fb6dc1e5",
		"runtimeNetworkGroup":            "sha256:9bba4570fcf666141742ad497f651896c60e88b119e0e93476221e4a32a2193c",
		"runtimeNetworkRevoke":           "sha256:47f39b0df0792ffbe01080b262cde159a915f3bbc0f97c1ee4a609356ac1b385",
		"runtimeNetworkFinish":           "sha256:a2afe4b31bf1c8bdf412b2eba66e8730ce4f984730d10321175001afe647a1c8",
		"runtimeVolumeBegin":             "sha256:97809a6fb0eec7ed67fffdc1a049f660cc0beb1d1db1847c50e4eb312e3e6513",
		"runtimeVolumeRegister":          "sha256:61c076c5b0fade884a566c8993eb17411f7e0164678fa98feb41a0e33feae09d",
		"runtimeVolumeRevoke":            "sha256:7215ead99ea32f1321d37ed780c1007560dc68144709f1118f2e590d20ce0a6c",
		"runtimeVolumeEnumerate":         "sha256:f61866815ad0de62e4d17e70d87fb2acd7813c5da023405c357e7f9a9337891c",
		"runtimeVolumeFinish":            "sha256:a7424c3b2987a3d4b93fcf2f93466bb159168e9bc4847b4791de077e93764c89",
		"processBackendAvailable":        "sha256:fef91e7103793ab52b6f63015db3118d25f5149ef70c030fd944a10fec0ad26a",
		"processBackendRole":             "sha256:4458fa6f941ac453f5957439b1ad95ccb6c735dd4f4461acc8660ce17ad07742",
		"processBackendBootstrap":        "sha256:d714168b794a8ef4ca1e00c8eb042b403a70cc68d53f41990acabb50eb03b155",
		"processBackendExchange":         "sha256:d431eb27d03a2ab04f1c37ec0219eacbcc2f166c8710b100d923a40590608e91",
		"processBackendWaitStop":         "sha256:ac35f227ddb1b96cde44425ea9c893463171f08dda0cf81054c34f4bf76d259d",
		"processBackendServeModel":       "sha256:4c72e912ee5bfd5e328c7ecf3455121b9238d443f8cc20dbf40e5a2d23846f58",
		"runtimeProcessNetworkOperation": "sha256:f7c753d1fdfccacf3dc01c9e8a9424b847d7f85916d3c3f28b4deb728d5287d4",
	}
	actual := make(map[string]string, len(want))
	for _, name := range []string{"runtime_domain.go", "runtime_network.go", "runtime_process.go", "runtime_process_model.go", "runtime_volume.go"} {
		source, err := os.ReadFile(name)
		require.NoError(t, err)
		parsed, err := parser.ParseFile(files, name, source, parser.ParseComments)
		require.NoError(t, err)
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if _, ok := want[function.Name.Name]; !ok {
				continue
			}
			var formatted bytes.Buffer
			require.NoError(t, format.Node(&formatted, files, function))
			digest := sha256.Sum256(formatted.Bytes())
			actual[function.Name.Name] = fmt.Sprintf("sha256:%x", digest)
		}
	}
	require.Equal(t, want, actual)
}
