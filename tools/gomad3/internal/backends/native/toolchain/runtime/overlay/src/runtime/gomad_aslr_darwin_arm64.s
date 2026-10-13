// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin && arm64

#include "textflag.h"

TEXT runtime·gomadPosixSpawnattrInit_trampoline(SB),NOSPLIT,$0
	MOVD	0(R0), R0	// arg 1 attr
	BL	libc_posix_spawnattr_init(SB)
	RET

TEXT runtime·gomadPosixSpawnattrSetflags_trampoline(SB),NOSPLIT,$0
	MOVH	8(R0), R1	// arg 2 flags
	MOVD	0(R0), R0	// arg 1 attr
	BL	libc_posix_spawnattr_setflags(SB)
	RET

TEXT runtime·gomadPosixSpawn_trampoline(SB),NOSPLIT,$0
	MOVD	8(R0), R1	// arg 2 path
	MOVD	16(R0), R2	// arg 3 file actions
	MOVD	24(R0), R3	// arg 4 attributes
	MOVD	32(R0), R4	// arg 5 argv
	MOVD	40(R0), R5	// arg 6 envp
	MOVD	0(R0), R0	// arg 1 pid
	BL	libc_posix_spawn(SB)
	RET

TEXT runtime·gomadImageSlide_trampoline(SB),NOSPLIT,$0
	MOVD	R0, R19		// R19 is callee-save
	MOVW	0(R19), R0	// arg 1 image index
	BL	libc__dyld_get_image_vmaddr_slide(SB)
	MOVD	R0, 8(R19)	// return value
	RET
