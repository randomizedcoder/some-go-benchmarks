//go:build amd64

#include "textflag.h"

// func rdtsc() uint64
//
// RDTSC reads the Time Stamp Counter into EDX:EAX.
// We combine them into a 64-bit value in AX and return it.
TEXT ·rdtsc(SB), NOSPLIT, $0-8
	RDTSC
	SHLQ	$32, DX
	ORQ	DX, AX
	MOVQ	AX, ret+0(FP)
	RET
