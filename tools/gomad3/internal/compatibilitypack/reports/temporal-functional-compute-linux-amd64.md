# Compatibility Pack Review: temporal-functional-compute-linux-amd64

Review SHA-256: `sha256:6b65f3b97ea639c174557d2227b35df1db8c79d70ac63380089b0ab0b0e50080`

Owner: `temporal-server`

Reviewed at: `2026-09-27T00:00:00Z`

Justification: Admits the exact amd64 assembly and runtime linknames of the deterministic hashing, compression, cryptographic, CPU-feature, and reflection packages the local Temporal frontend functional workload reaches on linux/amd64; guarded capability mode keeps the forbidden imports it also reaches behind the runtime guard.

Target: `go-test ./tests/gomadfunctional`

Target module: `go.temporal.io/server`

Test arguments: `-test.run ^TestFrontendSystemInfo$ -test.count=1`

Build tags: `disable_grpc_modules,test_dep`

Platform: `linux/amd64`

Workload: `frontend-system-info`

## Activation

- `filippo.io/edwards25519@v1.2.0` (`h1:crnVqOiS4jqYleHd9vaKZ+HKtHfllngJIiOpNpoJsjo=`), replacement `none`
- `github.com/cespare/xxhash/v2@v2.3.0` (`h1:UL815xU9SqsFlibzuggzjXhog7bL6oX9BbNZnL2UFvs=`), replacement `none`
- `github.com/dgryski/go-farm@v0.0.0-20240924180020-3414d57e47da` (`h1:aIftn67I1fkbMa512G+w+Pxci9hJPB8oMnkcP3iZF38=`), replacement `none`
- `github.com/golang/snappy@v1.0.0` (`h1:Oy607GVXHs7RtbggtPBnr2RmDArIsAefDwvrdWvRhGs=`), replacement `none`
- `github.com/klauspost/compress@v1.18.5` (`h1:/h1gH5Ce+VWNLSWqPzOVn6XBO+vJbCNGvjoaGBFW2IE=`), replacement `none`
- `github.com/modern-go/reflect2@v1.0.3-0.20250322232337-35a7c28c31ee` (`h1:W5t00kpgFdJifH4BDsTlE89Zl93FEloxaWZfGcifgq8=`), replacement `none`
- `github.com/twmb/murmur3@v1.1.8` (`h1:8Yt9taO/WN3l08xErzjeschgZU2QSrwm1kclYq+0aRg=`), replacement `none`
- `golang.org/x/crypto@v0.55.0` (`h1:+KWHjbgOaAQ66dh/YlkZKHlz9ZUlq61AFirAR9ntP8M=`), replacement `none`
- `golang.org/x/sys@v0.47.0` (`h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs=`), replacement `none`

## Reviewed packages

### `filippo.io/edwards25519/field`

Module: `filippo.io/edwards25519@v1.2.0` (`h1:crnVqOiS4jqYleHd9vaKZ+HKtHfllngJIiOpNpoJsjo=`), replacement `none`

Source set: `sha256:b0df6389e24ec1331f2d68fc32f3bdcfd91aa1d523bc0cc53dfdb603f47f186a`

Go sources:

- `fe.go`: `sha256:2ba45341ad3167343c18812b92d0fcddfad17e159e0770622a30fd190c565f06`
- `fe_amd64.go`: `sha256:1d1ea6a61ee950e19ce37b8be7b9250e962a922fb768bfc64cfbca63952af4d1`
- `fe_extra.go`: `sha256:8fda2b521aaa3e7506fd4556f82d115068222e5843a0822ec75376bfa7216024`
- `fe_generic.go`: `sha256:75d213650f76b7c5501202bf5cae1f0cdc1f84fbaa8bed5f93c0109189e01dba`

Foreign sources:

- `assembly:fe_amd64.s`: `sha256:affcfc732685da135acea57e6c0a09d893ad21547128b67c2e9583c23654f7c7`

Requested facts:

- `foreign:assembly:fe_amd64.s`: **allow** — **security-sensitive**

### `github.com/cespare/xxhash/v2`

Module: `github.com/cespare/xxhash/v2@v2.3.0` (`h1:UL815xU9SqsFlibzuggzjXhog7bL6oX9BbNZnL2UFvs=`), replacement `none`

Source set: `sha256:e73acc9b838ec3a638af489d7c54e3e457000cc0f841b1ac24f03970b3a16e7a`

Go sources:

- `xxhash.go`: `sha256:cc024316c7e49696f5705195951e49a8d24b612e2f95bec41ee4cd71990b78f9`
- `xxhash_asm.go`: `sha256:f5a64edc8b76317c95879329a0f3b358773fe3b529b8b88206012c9379145fc7`
- `xxhash_unsafe.go`: `sha256:b164ad04d24b0d1f5fbde666ae3806f4f33a23044359f63162aed343bcc97eb3`

Foreign sources:

- `assembly:xxhash_amd64.s`: `sha256:580c39fa974ecc91035f33cc258a4141cf52bc767d2f5283fb5b8609e4a856db`

Requested facts:

- `foreign:assembly:xxhash_amd64.s`: **allow** — **security-sensitive**

### `github.com/dgryski/go-farm`

Module: `github.com/dgryski/go-farm@v0.0.0-20240924180020-3414d57e47da` (`h1:aIftn67I1fkbMa512G+w+Pxci9hJPB8oMnkcP3iZF38=`), replacement `none`

Source set: `sha256:2bd79d3ec5b1e272184a8112daac728444465422d35d51f87f235ebd1444b2ac`

Go sources:

- `basics.go`: `sha256:095589f7dd715c6beb4c0eb244e5d72807ad051096e2d1d83f0c1d6ddf0463fa`
- `farmhashcc.go`: `sha256:8054715758c3ed2aefafa1dc71a1d645e331a8ceeb6c1f35f95800b1e29e378e`
- `farmhashmk.go`: `sha256:ff3b1c45588debb557b62b2cdb8ba999aaf25494849fe380adb00513fa5e1992`
- `farmhashna.go`: `sha256:743e608953a3d03a9af6eb69bbe06e20bb738e2eb87c9cf2b18abe76aa35baa4`
- `farmhashuo.go`: `sha256:9313d44a13fadb53c66b79b5b64235eca6541cc304fbddb4af0adb377b162db7`
- `farmhashxo.go`: `sha256:0589d68bb6aeb515a222fa4954774493f65fb3f3dad4b3eed815ab737ee8d9ac`
- `fp_stub.go`: `sha256:a2eb47362cef014ce32ae7ff15862a28b4ac69c6e0059288d8f13deed17175db`

Foreign sources:

- `assembly:fp_amd64.s`: `sha256:b83c6aae11827af118b25101122a67701d2276227153ac6f9a9963b0d329f2c6`

Requested facts:

- `foreign:assembly:fp_amd64.s`: **allow** — **security-sensitive**

### `github.com/golang/snappy`

Module: `github.com/golang/snappy@v1.0.0` (`h1:Oy607GVXHs7RtbggtPBnr2RmDArIsAefDwvrdWvRhGs=`), replacement `none`

Source set: `sha256:5320354d500850b5b3d83f60a6a6c7afe7ee5c7a7d041d8954de14482316e1b8`

Go sources:

- `decode.go`: `sha256:eebff83e4ab463713bb79b4d9f35c0212e72b9e1e02b18fb1632b412d4c8c192`
- `decode_asm.go`: `sha256:37ffc5a5ac8a0c376dc497891901010529b6bb510a8b9b5512f94eda3497274b`
- `encode.go`: `sha256:b4e357ac92d94c339a523d5c81834eb171f44047a5350376475e7424e1fbbe6e`
- `encode_asm.go`: `sha256:992413050d507073a011886d44c2d157d56e0dc949bda4704995a907edabca7b`
- `snappy.go`: `sha256:6fb3bc0c2c735aa29e587a64139267fb9cb3e1f947c88c2182f2998ebb2d3e5e`

Foreign sources:

- `assembly:decode_amd64.s`: `sha256:dac5b5604d2c92976cc13b11036d746b66f56164c41af2df80e362f3f4381680`
- `assembly:encode_amd64.s`: `sha256:b4868d32d151bf5094bdc9be29f1e35253ab79d8006c8dbf08fe5f0eda96cfc2`

Requested facts:

- `foreign:assembly:decode_amd64.s`: **allow** — **security-sensitive**
- `foreign:assembly:encode_amd64.s`: **allow** — **security-sensitive**

### `github.com/klauspost/compress/huff0`

Module: `github.com/klauspost/compress@v1.18.5` (`h1:/h1gH5Ce+VWNLSWqPzOVn6XBO+vJbCNGvjoaGBFW2IE=`), replacement `none`

Source set: `sha256:ddbda3050007a840305877dfc1d6a6300756df0ec5c591fd53316ca538ad08c9`

Go sources:

- `bitreader.go`: `sha256:dd01a3bc6e42f96a7516199fd2a6457e02aa531a2cedcc07b725dc7d37403b43`
- `bitwriter.go`: `sha256:da46bbc76279fbe6e5790647e0344c483444021b7650a28daa0c7dd7f31e3632`
- `compress.go`: `sha256:6cd9068c12f5dbedf23676310d9e81f8f4388a77c8e49bba27796f4f895ab78b`
- `decompress.go`: `sha256:1215c8e8259106f825450dc797550952e15aa8786b91630d5bf47664d35ec8df`
- `decompress_amd64.go`: `sha256:9aeafaec125e956d9a109a8f6a4093d6961902ed9657b05986534cf695be3faa`
- `huff0.go`: `sha256:699d9d7d9a84630dc55fd6e93b63a320aca9c6f16afe5dad15e04812e366bac7`

Foreign sources:

- `assembly:decompress_amd64.s`: `sha256:1be4b028f6a98b957cf7557629bbc3aaead44fccf0c60f796fdee608b8d7ee9e`

Requested facts:

- `foreign:assembly:decompress_amd64.s`: **allow** — **security-sensitive**

### `github.com/klauspost/compress/internal/cpuinfo`

Module: `github.com/klauspost/compress@v1.18.5` (`h1:/h1gH5Ce+VWNLSWqPzOVn6XBO+vJbCNGvjoaGBFW2IE=`), replacement `none`

Source set: `sha256:2499c2de444a2a5d9c574a1b700146554e859a32f718e945122fb19856179de3`

Go sources:

- `cpuinfo.go`: `sha256:289da93be4624cb43edccb6126032f3dce2870f3bb8712c6aaba20515d08d48d`
- `cpuinfo_amd64.go`: `sha256:78f2c3d5aee5c4187e48b5b8378ef9e4fdacf22593f429fc20326d0b2df9d7f1`

Foreign sources:

- `assembly:cpuinfo_amd64.s`: `sha256:9ea28b6d2e9e2210f12ef7d44af4716a3dc148c4e214f23f001271fd100547b9`

Requested facts:

- `foreign:assembly:cpuinfo_amd64.s`: **allow** — **security-sensitive**

### `github.com/klauspost/compress/zstd`

Module: `github.com/klauspost/compress@v1.18.5` (`h1:/h1gH5Ce+VWNLSWqPzOVn6XBO+vJbCNGvjoaGBFW2IE=`), replacement `none`

Source set: `sha256:eb4ca4be6d4d41918b780415554ff54c9ca218cbbf7178c47474397fb4d0a01d`

Go sources:

- `bitreader.go`: `sha256:f752b8860ccd02cef0d803f6ae513418554cd2675ce9db3b7d1a076b76d5cbc2`
- `bitwriter.go`: `sha256:d63c34ee81f4785ab08c9fd415e3e1da027f015007fc72bf9906e43c921e6718`
- `blockdec.go`: `sha256:b2e6c37759403e6b59f2ce9be65cab6e0a1d6629d8a79f8105ba1c64a563a76c`
- `blockenc.go`: `sha256:2c1bd142cc17d4fa7719855f9742b94132ccfa70a5f6602a219ceb0b487356a1`
- `blocktype_string.go`: `sha256:d7225622f9b2112c6a41f6df10265025786dbcd857d16c60b1757822ab6f3a8a`
- `bytebuf.go`: `sha256:ed79412400a6739a8060a4d04d60261d755e1b77a95b07f4b62761df07291ed9`
- `bytereader.go`: `sha256:455fe0da7298381a420501ba67bdcee2d2b82c60ed701bd5d2f3a02804c6ecd6`
- `decodeheader.go`: `sha256:18a618ead4628fb685e7da85c37af5e65517ed2d17c1c5029a352610f4929a06`
- `decoder.go`: `sha256:14b850eb2880c4103fd09cd693237af4f315894c610fc6bf51852865ee14b582`
- `decoder_options.go`: `sha256:acf7c4e6ac5b9cf1fa53cbed9c32eb40f2df1b7e8333851e8df1e543fd2239fa`
- `dict.go`: `sha256:c0742de26d9fcd7a26c472d2b2b823cf19bb73737883e7f4e469228c7d35ac83`
- `enc_base.go`: `sha256:3bcb79516bf469f077a4fd0486d2ece01f7bb2c6959b9ec3c4e3eaeb8e90289a`
- `enc_best.go`: `sha256:679b28e776926c3c70121163b0ed647d4fb7330f8804e161c906a605c741ef1a`
- `enc_better.go`: `sha256:f017c68af35219933def0324001fe0d5b4afbf13f9bb11bb89ebd60c5bf7839f`
- `enc_dfast.go`: `sha256:c69d71f76e4e6d40b75eb4d9e71fd10149819ad18da70f9a51ea3231c2a4d85a`
- `enc_fast.go`: `sha256:ea12ee70f9150942a4f5185b9825edfcfd9aa62c2d8cb1f0a9319f555bc5d237`
- `encoder.go`: `sha256:c101d114affe0b2e50260db6fc8adf893c02346340dcec0f534aef328042db21`
- `encoder_options.go`: `sha256:78cac0ea887986a2a500be17b85b38f2a6cb2331a9fb65ace1d0ca70abff91ac`
- `framedec.go`: `sha256:5ae64aea3e2c632d6148e0f0801d92290da1cd070b983d8298bfb505c7940cdf`
- `frameenc.go`: `sha256:826c75e733aaa50ef8cecb2223b4fad30c666cf1012cc0919fed64449067c9b2`
- `fse_decoder.go`: `sha256:005f20afa6bd1c58091acd7a7ea5c9500610d4a41bf7846e4ba9c59a5326be58`
- `fse_decoder_amd64.go`: `sha256:9a31305570cb5c98b698407be3b1b818fe4657ef0cf9118f0cf44959233f25d6`
- `fse_encoder.go`: `sha256:3925129f27af18f90955ac6fc57169d639605d421a69c76b31b4d85d4c815dff`
- `fse_predefined.go`: `sha256:3406d164c2738fdbbed4498da44e97d10f46995b70414649a238f041e7631ce6`
- `hash.go`: `sha256:4fdbadddb62eadcdcebe5f4a6d1ee08509069066677a3ca201d3eacb2894f5f2`
- `history.go`: `sha256:a3a1ac1c2c990ca03efcfb8185d0a9a1e68a99a016ccca8b1bbabcc7fda1fecb`
- `matchlen_amd64.go`: `sha256:682396138a4c9f624240e6d3276e9bc552803bd7db7ebd4c051ebc272fd65c6a`
- `seqdec.go`: `sha256:b2ea74a7161842bb0084474025f26b5375d2acbba2e5b7b25234bcfaf7efc4d7`
- `seqdec_amd64.go`: `sha256:a54d38ce2d9948da48d56378c6ac344e86d8f3bdc61c020407a39aa84ce330d6`
- `seqenc.go`: `sha256:a43251ae4dd47efc79873833648dfc5328ee5ab729e1a49bed9c627309f9a385`
- `simple_go124.go`: `sha256:c2adaeafb451e4ee7032d9fda084f627f6ad22c797b033a2ea398929726d583a`
- `snappy.go`: `sha256:f0b8b75c040e2981c2e085f2887f2f5095ce7ce7c929994329e6d5c5d860dc70`
- `zip.go`: `sha256:a76e95d6493c4dbac23adb33635ac105236d96da93089e485b3a39d4896e7696`
- `zstd.go`: `sha256:bcd7958d0d3a7d32cf7700e20db538886db0ba1ba14d64f6e955b3c4667c29ac`

Foreign sources:

- `assembly:fse_decoder_amd64.s`: `sha256:c4a1dfa1b108aa32f587b676523d09858c5ee5df9753d0067af5aee303e5df4b`
- `assembly:matchlen_amd64.s`: `sha256:f6983c33f36ef09e255c1d029158fa31adfaf6cbc56d264d3ff6431097cfbc0f`
- `assembly:seqdec_amd64.s`: `sha256:b3c4b4f8cc3224e2243824c767d6e6f2c61ca637a553104a581e69c4bdb7173f`

Requested facts:

- `foreign:assembly:fse_decoder_amd64.s`: **allow** — **security-sensitive**
- `foreign:assembly:matchlen_amd64.s`: **allow** — **security-sensitive**
- `foreign:assembly:seqdec_amd64.s`: **allow** — **security-sensitive**

### `github.com/klauspost/compress/zstd/internal/xxhash`

Module: `github.com/klauspost/compress@v1.18.5` (`h1:/h1gH5Ce+VWNLSWqPzOVn6XBO+vJbCNGvjoaGBFW2IE=`), replacement `none`

Source set: `sha256:cc4226894a26d1be5e7ef38eb0c9b3b722ec3b5920335e1555bb76eeee66f389`

Go sources:

- `xxhash.go`: `sha256:83344ca444865877a307d2980068f883716736e9a5b8fca36d13e5557ee319c1`
- `xxhash_asm.go`: `sha256:51742c9f72a6460f70d4a9dab6285074e7e59a874a40019f9af1821db34d3e23`
- `xxhash_safe.go`: `sha256:5a12c499074f3428854b32094344f11c8622d8e1548710d6c4e9f9ce365cd19a`

Foreign sources:

- `assembly:xxhash_amd64.s`: `sha256:3796a9c399d49392c3ad83af1452099d24c1e6fb993941d8bd1735879f5edbc8`

Requested facts:

- `foreign:assembly:xxhash_amd64.s`: **allow** — **security-sensitive**

### `github.com/modern-go/reflect2`

Module: `github.com/modern-go/reflect2@v1.0.3-0.20250322232337-35a7c28c31ee` (`h1:W5t00kpgFdJifH4BDsTlE89Zl93FEloxaWZfGcifgq8=`), replacement `none`

Source set: `sha256:9c0e723899306512615447aeb61c41214117f04c16d1a51562ad607019409941`

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

- `assembly:reflect2_amd64.s`: `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
- `assembly:relfect2_mips64x.s`: `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
- `assembly:relfect2_mipsx.s`: `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
- `assembly:relfect2_ppc64x.s`: `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`

Requested facts:

- `foreign:assembly:reflect2_amd64.s`: **allow** — **security-sensitive**
- `foreign:assembly:relfect2_mips64x.s`: **allow** — **security-sensitive**
- `foreign:assembly:relfect2_mipsx.s`: **allow** — **security-sensitive**
- `foreign:assembly:relfect2_ppc64x.s`: **allow** — **security-sensitive**
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

### `github.com/twmb/murmur3`

Module: `github.com/twmb/murmur3@v1.1.8` (`h1:8Yt9taO/WN3l08xErzjeschgZU2QSrwm1kclYq+0aRg=`), replacement `none`

Source set: `sha256:c1273c4f8afe53393aaf14442b69e08f1a4cb552564314f3da12a88a3e6f523c`

Go sources:

- `murmur.go`: `sha256:0afcdbfffdf0ff3dbb0fd2da67842bdcb5189b52e5e5cb3f887769507d4bd241`
- `murmur128.go`: `sha256:01af3a50a07d9b6a170f6e54493ce5b9e8d77c0e5570ddab31e8edfce0086191`
- `murmur128_decl.go`: `sha256:27babf62312ce06c71ca295926b6e99f6bedf240d142a8c80488c113e155a808`
- `murmur32.go`: `sha256:2ce01d3d167987841916dbb8f81d6fddd32f99c91ec8cf4520f99f711951c942`
- `murmur32_gen.go`: `sha256:3c549370f306e0d79fe1213fb69bab37a5ae82c86f77d3aa36bfccae67ff093f`
- `murmur64.go`: `sha256:f6cd8d41bb891d1a5564a0262c8ced574ddb1cb70b4749b7a934973f8fe9294e`

Foreign sources:

- `assembly:murmur128_amd64.s`: `sha256:eb772378e8b77d8e10f05360c473f3b9a632464aede83ca18b18aa6452a1c08c`

Requested facts:

- `foreign:assembly:murmur128_amd64.s`: **allow** — **security-sensitive**

### `golang.org/x/crypto/chacha20poly1305`

Module: `golang.org/x/crypto@v0.55.0` (`h1:+KWHjbgOaAQ66dh/YlkZKHlz9ZUlq61AFirAR9ntP8M=`), replacement `none`

Source set: `sha256:578255ef6549d34ff6c9b7524449aa4fc2d3d6d2fa0c8efd4e8162917755c550`

Go sources:

- `chacha20poly1305.go`: `sha256:08400b984074331e97c7e2a1d9701f14cb8f9919b3eb3e0d1bb8e3549961cac2`
- `chacha20poly1305_amd64.go`: `sha256:bae40d20a6077f0f5570b28086cdc4cb6e51aaa757e54b32ba99770f8b234869`
- `chacha20poly1305_generic.go`: `sha256:f3e04c8517e163f94ec2bb6e7b634ef239432ed7422b1d8695cfc446c8551181`
- `fips140only_go1.26.go`: `sha256:60368cae6d4d630509309819d7f397532c6f7812978c12983bca375e4ae810aa`
- `xchacha20poly1305.go`: `sha256:7325ab1f70b6ab4a241ca0237f050202ddb35225a690c8146006e51a6ceeb685`

Foreign sources:

- `assembly:chacha20poly1305_amd64.s`: `sha256:4146e78a518ac6030aff76b7ced4b007d50757c63acca363403f180134093af8`

Requested facts:

- `foreign:assembly:chacha20poly1305_amd64.s`: **allow** — **security-sensitive**
- `import:golang.org/x/sys/cpu`: **deny** — **security-sensitive**

### `golang.org/x/crypto/internal/poly1305`

Module: `golang.org/x/crypto@v0.55.0` (`h1:+KWHjbgOaAQ66dh/YlkZKHlz9ZUlq61AFirAR9ntP8M=`), replacement `none`

Source set: `sha256:330c7c1958e7696192bdb5ea7727c333fd6d071d214d19f3c5fa386d9350b31b`

Go sources:

- `poly1305.go`: `sha256:c4b20ab7330e47ee155a775f69ae3b0e432c75d74ad005cf764228a69ddf6da6`
- `sum_asm.go`: `sha256:39e3031ebe8eccf3ac0324c1d8d21fbbac8b74e9bd48737b0c042c28b4676d43`
- `sum_generic.go`: `sha256:b0094a2895d5bda42dcaaf57c0b31fc914c3b6c6aa6237aab0100a6d78346933`

Foreign sources:

- `assembly:sum_amd64.s`: `sha256:f8959555c2e70f460ba88bca1f37705d6c570c0f99f37650a907e9391a960446`

Requested facts:

- `foreign:assembly:sum_amd64.s`: **allow** — **security-sensitive**

### `golang.org/x/sys/cpu`

Module: `golang.org/x/sys@v0.47.0` (`h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs=`), replacement `none`

Source set: `sha256:5bb3e9321d057a387573e1f4ea64b5094be8d8bde19b832fa47944719bd428d9`

Go sources:

- `byteorder.go`: `sha256:825146fd4557b1cbd8161fa28bb4be8820089848d695316edeecb7fd5a551f8a`
- `cpu.go`: `sha256:b56854737e8d803d582232f9790f7dda2d8ff69e5982f6cdbdd2a150be540922`
- `cpu_gc_x86.go`: `sha256:2eec0a58e170f14b054a9d0c006fede8cd5c075c5d7cde1fbab688b4b793588e`
- `cpu_linux_noinit.go`: `sha256:620abd97199a2230f59165a2bc58c3ba780b6801ec0246cfb7319553dd795226`
- `cpu_other_x86.go`: `sha256:6f447bdafdc6f75593b29977c4f43a39f743043d4f286c542a8b93fa7fad3bf9`
- `cpu_x86.go`: `sha256:6a9cdf0eca762b62d0e49e1f3bc9b5c68f23fecbdb888d12ef27bc32bc3893b9`
- `endian_little.go`: `sha256:c6bc70c372d9e1fe86fcf295f406b17bf04bf8d1af25c2456f58520cdaef3be9`
- `hwcap_linux.go`: `sha256:4101df793fddf76dfae477f917928008bc4a797446cd4ad44cde6e906c3f8713`
- `parse.go`: `sha256:97b269ea4f0b6d4071a9eed8a74f05055965c307bbab9090d9002bb01f7365a9`
- `runtime_auxv.go`: `sha256:d898ace395866bed261d403c8cd0ea6eab6d6d77f52042204957e389c38938cf`
- `runtime_auxv_go121.go`: `sha256:6eee9d1a593dce53d22545dc5d5f6ed9127a43e6568ef5f5521920946510d445`

Foreign sources:

- `assembly:cpu_gc_x86.s`: `sha256:74ac7fc7ef9c56c3306238cf031ea8ef7c0312a7116cb9fba6d07f0b1382df80`

Requested facts:

- `foreign:assembly:cpu_gc_x86.s`: **allow** — **security-sensitive**
- `linkname:runtime_auxv_go121.go`: **allow** — **security-sensitive**
  - source `sha256:6eee9d1a593dce53d22545dc5d5f6ed9127a43e6568ef5f5521920946510d445`
  - directive `runtime_getAuxv runtime.getAuxv`

