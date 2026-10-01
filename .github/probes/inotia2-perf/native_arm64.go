//go:build linux && arm64

package main

import (
	"github.com/mirusu400/aram-core/application"
	"github.com/mirusu400/aram-core/cpu"
	"github.com/mirusu400/aram-core/cpu/interpreter"
)

func init() { application.RegisterCPUBackend("native", func() cpu.Backend { return interpreter.NewNativeJIT() }) }
