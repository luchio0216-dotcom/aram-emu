package bootstrap

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func TestActivationRetrySequence(t *testing.T) {
	denied := &os.LinkError{Op: "rename", Old: "staging", New: "target", Err: syscall.Errno(5)}
	sharing := fmt.Errorf("wrapped: %w", syscall.Errno(32))
	terminal := errors.New("terminal")
	tests := []struct {
		name    string
		results []error
		retry   bool
		want    error
		sleeps  []time.Duration
	}{
		{"success", []error{nil}, true, nil, nil},
		{"denied_then_success", []error{denied, nil}, true, nil, []time.Duration{10 * time.Millisecond}},
		{"sharing_then_success", []error{sharing, nil}, true, nil, []time.Duration{10 * time.Millisecond}},
		{"sixth_success", []error{denied, sharing, denied, sharing, denied, nil}, true, nil, []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond, 80 * time.Millisecond, 160 * time.Millisecond}},
		{"exhaustion", []error{denied, denied, denied, denied, denied, sharing}, true, sharing, []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond, 80 * time.Millisecond, 160 * time.Millisecond}},
		{"terminal", []error{terminal}, true, terminal, nil},
		{"becomes_terminal", []error{denied, terminal}, true, terminal, []time.Duration{10 * time.Millisecond}},
		{"other_platform", []error{denied}, false, denied, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			var sleeps []time.Duration
			err := retryActivation("staging", "target", func(old, new string) error {
				if old != "staging" || new != "target" {
					t.Fatal("paths changed")
				}
				if calls >= len(tt.results) {
					t.Fatal("too many attempts")
				}
				result := tt.results[calls]
				calls++
				return result
			}, func(d time.Duration) { sleeps = append(sleeps, d) }, func(err error) bool {
				return tt.retry && (errors.Is(err, syscall.Errno(5)) || errors.Is(err, syscall.Errno(32)))
			})
			if err != tt.want || calls != len(tt.results) || !reflect.DeepEqual(sleeps, tt.sleeps) {
				t.Fatalf("err=%v calls=%d sleeps=%v", err, calls, sleeps)
			}
		})
	}
}

func TestActivationPlatformClassification(t *testing.T) {
	for _, code := range []syscall.Errno{2, 5, 6, 17, 32, 80, 183} {
		err := &os.LinkError{Op: "rename", Err: code}
		want := runtime.GOOS == "windows" && (code == 5 || code == 32)
		if got := activationRetryable(err); got != want {
			t.Fatalf("errno%d retry=%v want%v", code, got, want)
		}
	}
	if activationRetryable(nil) || activationRetryable(errors.New("Access is denied.")) {
		t.Fatal("classified by text or nil")
	}
}

func TestActivationInstallContentAndMarker(t *testing.T) {
	for _, format := range []string{"zip", "tar.gz"} {
		t.Run(format, func(t *testing.T) {
			isolateConfig(t)
			var previous string
			for version := 0; version < 2; version++ {
				archive := filepath.Join(t.TempDir(), "aram."+format)
				payload := []byte(fmt.Sprintf("synthetic executable version%d", version))
				build := []byte(fmt.Sprintf("build%d", version))
				createProductArchive(t, archive, format, map[string][]byte{productExecutableName(): payload, "BUILD-INFO.txt": build})
				executable, err := Install(archive)
				if err != nil {
					t.Fatal(err)
				}
				for name, want := range map[string][]byte{productExecutableName(): payload, "BUILD-INFO.txt": build} {
					got, err := os.ReadFile(filepath.Join(filepath.Dir(executable), name))
					if err != nil || !bytes.Equal(got, want) {
						t.Fatalf("installed %s: %q err=%v", name, got, err)
					}
				}
				marker, err := readCurrentRuntime()
				if err != nil || marker.ArchiveSHA256 != digestOf(t, archive) || !samePath(marker.Executable, executable) {
					t.Fatalf("marker=%+v err=%v", marker, err)
				}
				selected, err := CurrentExecutable()
				if err != nil || !samePath(selected, executable) {
					t.Fatal("selected runtime", err)
				}
				if filepath.Base(filepath.Dir(executable)) != marker.ArchiveSHA256[:16] {
					t.Fatal("not content addressed")
				}
				again, err := Install(archive)
				if err != nil || !samePath(again, executable) {
					t.Fatal("repeat install", err)
				}
				if previous != "" {
					got, err := os.ReadFile(previous)
					if err != nil || !bytes.Equal(got, []byte("synthetic executable version0")) {
						t.Fatal("previous runtime changed", err)
					}
				}
				previous = executable
			}
		})
	}
}
