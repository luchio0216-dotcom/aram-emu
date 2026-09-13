package bootstrap

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// These controlled locks establish retry handling, not the provenance of any
// historical activation denial. Every archive, handle and marker is TEMP-only.
func TestActivationWindowsControlledHandles(t *testing.T) {
	for _, directoryHandle := range []bool{false, true} {
		for _, release := range []bool{false, true} {
			name := "file"
			if directoryHandle {
				name = "directory"
			}
			if release {
				name += "/release"
			} else {
				name += "/persistent"
			}
			t.Run(name, func(t *testing.T) {
				isolateConfig(t)
				archive := filepath.Join(t.TempDir(), "old.tar.gz")
				createProductArchive(t, archive, "tar.gz", map[string][]byte{productExecutableName(): []byte("old selected synthetic executable")})
				selected, err := Install(archive)
				if err != nil {
					t.Fatal(err)
				}
				markerPath, err := currentRuntimePath()
				if err != nil {
					t.Fatal(err)
				}
				marker, err := os.ReadFile(markerPath)
				if err != nil {
					t.Fatal(err)
				}
				root, err := runtimeDirectory()
				if err != nil {
					t.Fatal(err)
				}
				staging, err := os.MkdirTemp(root, ".install-control-")
				if err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(root, "controlled-target")
				payload := []byte("new synthetic executable")
				stagedFile := filepath.Join(staging, productExecutableName())
				if err = os.WriteFile(stagedFile, payload, 0600); err != nil {
					t.Fatal(err)
				}
				lockPath, flags := stagedFile, uint32(0)
				if directoryHandle {
					lockPath = staging
					flags = syscall.FILE_FLAG_BACKUP_SEMANTICS
				}
				path, err := syscall.UTF16PtrFromString(lockPath)
				if err != nil {
					t.Fatal(err)
				}
				handle, err := syscall.CreateFile(path, syscall.GENERIC_READ, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, flags, 0)
				if err != nil {
					t.Fatal(err)
				}
				open := true
				defer func() {
					if open {
						_ = syscall.CloseHandle(handle)
					}
				}()
				attempts, sleeps := 0, 0
				var last error
				err = retryActivation(staging, target, func(old, new string) error {
					attempts++
					last = os.Rename(old, new)
					if last != nil {
						var errno syscall.Errno
						if !errors.As(last, &errno) || !activationRetryable(last) {
							t.Fatalf("unexpected controlled denial: %v", last)
						}
						t.Logf("controlled attempt%d errno%d", attempts, errno)
					}
					return last
				}, func(d time.Duration) {
					want := (10 * time.Millisecond) << sleeps
					if d != want {
						t.Fatalf("sleep=%v want%v", d, want)
					}
					sleeps++
					if release && open {
						if e := syscall.CloseHandle(handle); e != nil {
							t.Fatal(e)
						}
						open = false
					}
				}, activationRetryable)
				if release {
					if err != nil || attempts != 2 || sleeps != 1 {
						t.Fatalf("release err=%v attempts=%d sleeps=%d", err, attempts, sleeps)
					}
					got, e := os.ReadFile(filepath.Join(target, productExecutableName()))
					if e != nil || !bytes.Equal(got, payload) {
						t.Fatal("activated bytes", e)
					}
					if _, e = os.Stat(staging); !errors.Is(e, os.ErrNotExist) {
						t.Fatal("staging not renamed", e)
					}
				} else {
					if err == nil || err != last || attempts != 6 || sleeps != 5 {
						t.Fatalf("persistent err=%v attempts=%d sleeps=%d", err, attempts, sleeps)
					}
					got, e := os.ReadFile(stagedFile)
					if e != nil || !bytes.Equal(got, payload) {
						t.Fatal("failed activation changed staging", e)
					}
					if _, e = os.Stat(target); !errors.Is(e, os.ErrNotExist) {
						t.Fatal("failed activation created target", e)
					}
					// Exercise the production wiring with the controlled handle still held.
					if e := activateRuntime(staging, target); !activationRetryable(e) {
						t.Fatalf("production sustained lock: %v", e)
					}
				}
				gotMarker, e := os.ReadFile(markerPath)
				if e != nil || !bytes.Equal(gotMarker, marker) {
					t.Fatal("activation changed current marker", e)
				}
				gotSelected, e := CurrentExecutable()
				if e != nil || !samePath(gotSelected, selected) {
					t.Fatal("activation changed selection", e)
				}
				if open {
					if e := syscall.CloseHandle(handle); e != nil {
						t.Fatal(e)
					}
					open = false
				}
			})
		}
	}
}
