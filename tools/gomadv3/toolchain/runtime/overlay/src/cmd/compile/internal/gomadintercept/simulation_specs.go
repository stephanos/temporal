// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadintercept

var simulationSpecs = []spec{
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeDomainAvailable", Hook: "gomadInterceptRuntimeDomainAvailable", DeclarationSHA256: "sha256:e1f2f00d0f21e11d688551c6d77f5bc9abea677cf0805fea7125cf02723ef090"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeDomainBegin", Hook: "gomadInterceptRuntimeDomainBegin", DeclarationSHA256: "sha256:841134e65475799cf130d4127beacad19c09b69d09df02b964435ae7027b13e8"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeDomainRegister", Hook: "gomadInterceptRuntimeDomainRegister", DeclarationSHA256: "sha256:a4d0f95dc6fd33aa78092805005475dd7ad18c272d6a586b16ea727f368a8a5e"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeDomainEnter", Hook: "gomadInterceptRuntimeDomainEnter", DeclarationSHA256: "sha256:d11ef41916402a6082de54e8cd41346e72ec5bf68a4daab01145d7618a783a1d"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeDomainLeave", Hook: "gomadInterceptRuntimeDomainLeave", DeclarationSHA256: "sha256:21ec316a2b18610e9e946903c7d54cc0e7be105df3526162c0c224d28344608d"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeDomainRevoke", Hook: "gomadInterceptRuntimeDomainRevoke", DeclarationSHA256: "sha256:aa9b9a4a8617f833d1f99db1be300ca1d9bf4919fb22274a7514e3c5d38aa8c9"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeDomainFinish", Hook: "gomadInterceptRuntimeDomainFinish", DeclarationSHA256: "sha256:0521b81eda5b875ee982b88f1266528ce9b9a3236b104d9e072bb35dd71dc807"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeNetworkBegin", Hook: "gomadInterceptRuntimeNetworkBegin", DeclarationSHA256: "sha256:c6a522160ea85057c83e2116d092adf2e957e37976365246df048fd8318ce90c"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeNetworkPartition", Hook: "gomadInterceptRuntimeNetworkPartition", DeclarationSHA256: "sha256:8f16017f6bcef0f6f39408501258d1ff4510b6eb375eb818fde758214241a1a5"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeNetworkHeal", Hook: "gomadInterceptRuntimeNetworkHeal", DeclarationSHA256: "sha256:9ad06c2e24cf20da1ad412695d9e1a7f8c5016c9627b5c9901a6c4dbff2ef664"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeNetworkDelay", Hook: "gomadInterceptRuntimeNetworkDelay", DeclarationSHA256: "sha256:e4b66ced616c6d837f443f8c304ee5b87777cb9f1fba7084f15696b9fb6dc1e5"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeNetworkGroup", Hook: "gomadInterceptRuntimeNetworkGroup", DeclarationSHA256: "sha256:9bba4570fcf666141742ad497f651896c60e88b119e0e93476221e4a32a2193c"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeNetworkRevoke", Hook: "gomadInterceptRuntimeNetworkRevoke", DeclarationSHA256: "sha256:47f39b0df0792ffbe01080b262cde159a915f3bbc0f97c1ee4a609356ac1b385"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeNetworkFinish", Hook: "gomadInterceptRuntimeNetworkFinish", DeclarationSHA256: "sha256:a2afe4b31bf1c8bdf412b2eba66e8730ce4f984730d10321175001afe647a1c8"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeVolumeBegin", Hook: "gomadInterceptRuntimeVolumeBegin", DeclarationSHA256: "sha256:97809a6fb0eec7ed67fffdc1a049f660cc0beb1d1db1847c50e4eb312e3e6513"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeVolumeRegister", Hook: "gomadInterceptRuntimeVolumeRegister", DeclarationSHA256: "sha256:61c076c5b0fade884a566c8993eb17411f7e0164678fa98feb41a0e33feae09d"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeVolumeRevoke", Hook: "gomadInterceptRuntimeVolumeRevoke", DeclarationSHA256: "sha256:7215ead99ea32f1321d37ed780c1007560dc68144709f1118f2e590d20ce0a6c"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeVolumeEnumerate", Hook: "gomadInterceptRuntimeVolumeEnumerate", DeclarationSHA256: "sha256:f61866815ad0de62e4d17e70d87fb2acd7813c5da023405c357e7f9a9337891c"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeVolumeFinish", Hook: "gomadInterceptRuntimeVolumeFinish", DeclarationSHA256: "sha256:a7424c3b2987a3d4b93fcf2f93466bb159168e9bc4847b4791de077e93764c89"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "processBackendAvailable", Hook: "gomadInterceptProcessBackendAvailable", DeclarationSHA256: "sha256:fef91e7103793ab52b6f63015db3118d25f5149ef70c030fd944a10fec0ad26a"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "processBackendRole", Hook: "gomadInterceptProcessBackendRole", DeclarationSHA256: "sha256:4458fa6f941ac453f5957439b1ad95ccb6c735dd4f4461acc8660ce17ad07742"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "processBackendBootstrap", Hook: "gomadInterceptProcessBackendBootstrap", DeclarationSHA256: "sha256:d714168b794a8ef4ca1e00c8eb042b403a70cc68d53f41990acabb50eb03b155"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "processBackendExchange", Hook: "gomadInterceptProcessBackendExchange", DeclarationSHA256: "sha256:d431eb27d03a2ab04f1c37ec0219eacbcc2f166c8710b100d923a40590608e91"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "processBackendWaitStop", Hook: "gomadInterceptProcessBackendWaitStop", DeclarationSHA256: "sha256:ac35f227ddb1b96cde44425ea9c893463171f08dda0cf81054c34f4bf76d259d"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "processBackendServeModel", Hook: "gomadInterceptProcessBackendServeModel", DeclarationSHA256: "sha256:4c72e912ee5bfd5e328c7ecf3455121b9238d443f8cc20dbf40e5a2d23846f58"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeProcessNetworkOperation", Hook: "gomadInterceptRuntimeProcessNetworkOperation", DeclarationSHA256: "sha256:f7c753d1fdfccacf3dc01c9e8a9424b847d7f85916d3c3f28b4deb728d5287d4"},
	{PackagePath: "go.temporal.io/server/tools/gomadv3sim", Function: "runtimeProcessVolumeOperation", Hook: "gomadInterceptRuntimeProcessVolumeOperation", DeclarationSHA256: "sha256:3b11980475a86a133f768929fa60512a29ccb03488b9c21abb9e739a64cfe55a"},
}
