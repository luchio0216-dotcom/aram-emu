package integration

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"testing"

	"github.com/mirusu400/aram-core/application"
	"github.com/mirusu400/aram-frontend/frontend"
)

func TestLegacyRecognitionErrorsPreserveInputIdentity(t *testing.T) {
	mif := make([]byte, 64)
	for offset, value := range map[int]uint32{0: 0x10011, 8: 32, 12: 8, 16: 40, 20: 1, 24: 48, 28: 16} {
		binary.LittleEndian.PutUint32(mif[offset:], value)
	}
	brew := syntheticZIP(t, map[string][]byte{"app.mif": mif, "app.mod": []byte("opaque synthetic module")})
	// Synthetic header and undecoded body. These bytes claim no GVM instruction
	// semantics, matching the core's deliberately recognition-only contract.
	sgs := append([]byte{1, 0xff, 0x0c, 0xff, 0xff, 1, 0, 0, 0, 0}, []byte("Synthetic\x00\x00\x01\x02\x03\x04")...)
	for _, test := range []struct {
		name, format, profile string
		data                  []byte
	}{
		{"synthetic.zip", "brew-package", "brew-container-v1/unknown/generic", brew},
		{"synthetic.sgs", "gnex-sgs", "gvm-container-v1/skt/generic", sgs},
	} {
		t.Run(test.format, func(t *testing.T) {
			backend := NewBackend(nil)
			t.Cleanup(func() { _ = backend.Close() })
			info, err := backend.Open(context.Background(), frontend.OpenRequest{DisplayName: test.name, Data: test.data})
			var backendErr *frontend.BackendError
			var unsupported *application.UnsupportedPlatformError
			if !errors.As(err, &backendErr) || backendErr.Kind != frontend.FailureUnsupportedProfile ||
				!errors.As(err, &unsupported) || !errors.Is(err, application.ErrUnsupportedSource) {
				t.Fatalf("recognition must preserve typed unsupported execution: %v", err)
			}
			if info.Format != test.format || info.ProfileID != test.profile || info.DisplayName != test.name ||
				info.Size != int64(len(test.data)) || info.SHA256 != fmt.Sprintf("%x", sha256.Sum256(test.data)) {
				t.Fatalf("recognized identity = %+v", info)
			}
			if backend.State() != frontend.StateEmpty || backend.Supports(frontend.CommandStart) {
				t.Fatal("recognition-only package became executable")
			}
		})
	}
}

func TestExplicitGVMDiagnosticProfileReachesServiceBoundary(t *testing.T) {
	const entry, ds, ps, dm = 54, 128, 132, 136
	sgs := make([]byte, dm)
	sgs[0], sgs[2], sgs[5], sgs[10] = 2, 12, 1, 'G'
	for offset, value := range map[int]uint16{
		0x1c: entry,
		0x2c: ds,
		0x2e: ps,
		0x30: dm,
		0x32: dm,
	} {
		binary.LittleEndian.PutUint16(sgs[offset:], value)
	}
	copy(sgs[entry:], []byte{0x06, 0, 10, 0x06, 0x12, 0x34, 0x9a, 0xff})
	copy(sgs[ds:], []byte{1, 2, 1, 0})
	copy(sgs[ps:], []byte{1, 2, 3, 4})

	backend := NewBackend(nil)
	t.Cleanup(func() { _ = backend.Close() })
	info, err := backend.Open(context.Background(), frontend.OpenRequest{
		DisplayName: "diagnostic.sgs",
		Data:        sgs,
		ProfileID:   application.GVMKernelProfileID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if info.Format != "gnex-sgs" || info.ProfileID != application.GVMKernelProfileID ||
		backend.State() != frontend.StateReady {
		t.Fatalf("open identity=%+v state=%s", info, backend.State())
	}
	if err := backend.Execute(context.Background(), frontend.CommandStart); err != nil {
		t.Fatal(err)
	}
	diagnostics := backend.Diagnostics()
	if diagnostics.Execution == nil || diagnostics.Execution.Reason != "service-boundary" ||
		diagnostics.Execution.Instructions != 2 || diagnostics.Execution.PC != 61 ||
		diagnostics.Execution.Error != "" || backend.State() != frontend.StateStopped {
		t.Fatalf("diagnostics=%+v state=%s", diagnostics.Execution, backend.State())
	}
	if diagnostics.GVM == nil || diagnostics.GVM.Boundary != "timer-request" ||
		diagnostics.GVM.Interval != 10 || diagnostics.GVM.Selector != 0x1234 {
		t.Fatalf("GVM diagnostics=%+v", diagnostics.GVM)
	}
	if frame := backend.VideoFrame(); frame.Image != nil || frame.Sequence != 0 {
		t.Fatalf("diagnostic profile manufactured a video frame: %+v", frame)
	}
	for _, command := range []frontend.BackendCommand{
		frontend.CommandStart,
		frontend.CommandPauseResume,
		frontend.CommandFrame,
		frontend.CommandLoadState,
		frontend.CommandSaveState,
	} {
		if capability := backend.Capability(command); capability.Supported || capability.Reason == "" {
			t.Fatalf("%s capability=%+v", command, capability)
		}
	}
	if capability := backend.Capability(frontend.CommandReset); !capability.Supported {
		t.Fatalf("reset capability=%+v", capability)
	}
	if err := backend.QueueInput(frontend.InputEvent{Control: "ok", Pressed: true}); !errors.Is(err, application.ErrGVMInputUnavailable) {
		t.Fatalf("input boundary: %v", err)
	}
}

func TestJ2MESaveStateRejectsAnotherResolvedProfile(t *testing.T) {
	ctx := context.Background()
	backend := NewBackend(nil)
	t.Cleanup(func() { _ = backend.Close() })
	if err := backend.ConfigureStateRoot(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	request := frontend.OpenRequest{DisplayName: "synthetic.jar", Data: syntheticJ2ME(t, false)}
	if _, err := backend.Open(ctx, request); err != nil {
		t.Fatal(err)
	}
	for _, command := range []frontend.BackendCommand{frontend.CommandStart, frontend.CommandSaveState} {
		if err := backend.Execute(ctx, command); err != nil {
			t.Fatal(err)
		}
	}
	request.ProfileID = "j2me-1.0/lgt/generic"
	if _, err := backend.Open(ctx, request); err != nil {
		t.Fatal(err)
	}
	if err := backend.Execute(ctx, frontend.CommandLoadState); err == nil {
		t.Fatal("generic Java state was accepted under an LGT profile")
	}
	if backend.State() != frontend.StateReady || backend.Diagnostics().Input.ProfileID != request.ProfileID {
		t.Fatal("rejected state mutated the current Java profile or lifecycle")
	}
}
