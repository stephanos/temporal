# Compatibility Pack Review: reflect2-go126

Review SHA-256: `sha256:203be98d0a33919bcebbb484507fce282107d9f13c420ad0fafd98ca3801dc61`

Owner: `temporal-server`

Reviewed at: `2026-08-15T00:00:00Z`

Justification: Preserves the exact reviewed reflect2 runtime-link boundary required by the Temporal representative workload.

Target: `go-test ./temporal`

Target module: `go.temporal.io/server`

Test arguments: `-test.run ^TestNewServerWithOTEL$`

Build tags: `test_dep`

Platform: `darwin/arm64`

Workload: `temporal-representative`

## Activation

- `github.com/modern-go/reflect2@v1.0.3-0.20250322232337-35a7c28c31ee` (`h1:W5t00kpgFdJifH4BDsTlE89Zl93FEloxaWZfGcifgq8=`), replacement `none`

## Reviewed packages

### `github.com/modern-go/reflect2`

Module: `github.com/modern-go/reflect2@v1.0.3-0.20250322232337-35a7c28c31ee` (`h1:W5t00kpgFdJifH4BDsTlE89Zl93FEloxaWZfGcifgq8=`), replacement `none`

Source set: `sha256:a848d53c66893da2ba62d7d9523ad2ce65c84bd6ba3bbaaebdcb79edb0e2deb7`

Go sources:

- `go_above_118.go`: `sha256:b41d841d561da73b0ab54f9f2830d7f9437561b831faad1fa22f738ea99ad805`
- `go_above_19.go`: `sha256:422e740515d8517cdc4d412e0fe0bf3d42f86909302ce3cf2df66a8800fd021f`
- `reflect2.go`: `sha256:23df966bbd3419c6ad2eddb10eec1c0d6ccbd337912625a823b00689cceb1c76`
- `reflect2_kind.go`: `sha256:7d5ac0c71ac5fba79d2b96ff1387e53ab4b1770501a9199c2a555bccfd2f1c8a`
- `safe_field.go`: `sha256:3295dc8e033a764f3797b65c97c5b9f6deaaba8adebafe7e8f067a383ccf34de`
- `safe_map.go`: `sha256:19e7c56513a6133a54c7314b1d2b272e289ea701e192f85d8dca3cf764a045ae`
- `safe_slice.go`: `sha256:5e7acc8d9c21ce3384c218b046bdd2a21422fa1681fc1912b4f4cdc5cfffc856`
- `safe_struct.go`: `sha256:2a06f38bf1093f94a3dc482432e03fa07ba0e85cdfbb8f2d1f0bddc68a5a74aa`
- `safe_type.go`: `sha256:b7528634290f8a731c18233bad53535aceb77d942b25f96813872f79246081ac`
- `type_map.go`: `sha256:4fabf996f68479b1b4ab68741558fe85074c97ec4d576cd113c0383cf56286db`
- `unsafe_array.go`: `sha256:02014c8f507943e69abc22b219dbb8dd60b2d35fa2c787a5ca8a3a532de1690a`
- `unsafe_eface.go`: `sha256:e00a1d58505e0c7c2afc8bddb5cbb01070173d53df10441cc21f8db22c6b148f`
- `unsafe_field.go`: `sha256:a9147bb01f44f670c93e82e4d6c084735a36a09a293dc5eb14d85b0a9d4c0cfe`
- `unsafe_iface.go`: `sha256:d21952ed67758fc50df9164d5b9f44bb542a64bc960b0359951c3b53fc7e3cf7`
- `unsafe_link.go`: `sha256:f2ac5514fc2dc286e08f9c3655c32ec5821e262df610004161cb10dd27c08cde`
- `unsafe_map.go`: `sha256:e67735062461c806294fb78b20d6949cd6c1601d43f4f77150ccdbb6e689b32b`
- `unsafe_ptr.go`: `sha256:5dd7c5c463663d4ab8bd96954e4d9d561d886f44ff82e7b89e6cb006ef53bb36`
- `unsafe_slice.go`: `sha256:810e20cb269ebfb6cf98cdf729712e7941a3afe73db6c1450de84aa688c04aaf`
- `unsafe_struct.go`: `sha256:806e182f0cfa7c371f332c24b8f77f8edbce2d22fe15372463674d1eafb18823`
- `unsafe_type.go`: `sha256:aebd12151ce6dbc450bcf63a37d8ec8e9d3bb90dca521cd817779187705e4624`

Foreign sources:

- `assembly:relfect2_arm64.s`: `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
- `assembly:relfect2_mips64x.s`: `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
- `assembly:relfect2_mipsx.s`: `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
- `assembly:relfect2_ppc64x.s`: `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`

Requested facts:

- `foreign:assembly:relfect2_arm64.s`: **deny** — **security-sensitive**
- `foreign:assembly:relfect2_mips64x.s`: **deny** — **security-sensitive**
- `foreign:assembly:relfect2_mipsx.s`: **deny** — **security-sensitive**
- `foreign:assembly:relfect2_ppc64x.s`: **deny** — **security-sensitive**
- `linkname:go_above_118.go`: **allow** — **security-sensitive**
  - source `sha256:b41d841d561da73b0ab54f9f2830d7f9437561b831faad1fa22f738ea99ad805`
  - directive `mapiterinit reflect.mapiterinit`
- `linkname:go_above_19.go`: **allow** — **security-sensitive**
  - source `sha256:422e740515d8517cdc4d412e0fe0bf3d42f86909302ce3cf2df66a8800fd021f`
  - directive `resolveTypeOff reflect.resolveTypeOff`
  - directive `makemap reflect.makemap`
- `linkname:type_map.go`: **allow** — **security-sensitive**
  - source `sha256:4fabf996f68479b1b4ab68741558fe85074c97ec4d576cd113c0383cf56286db`
  - directive `typelinks2 reflect.typelinks`
- `linkname:unsafe_link.go`: **allow** — **security-sensitive**
  - source `sha256:f2ac5514fc2dc286e08f9c3655c32ec5821e262df610004161cb10dd27c08cde`
  - directive `unsafe_New reflect.unsafe_New`
  - directive `typedmemmove reflect.typedmemmove`
  - directive `unsafe_NewArray reflect.unsafe_NewArray`
  - directive `typedslicecopy reflect.typedslicecopy`
  - directive `mapassign reflect.mapassign`
  - directive `mapaccess reflect.mapaccess`
  - directive `mapiternext reflect.mapiternext`
  - directive `ifaceE2I reflect.ifaceE2I`

