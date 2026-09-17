package integration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	aramcore "github.com/mirusu400/aram-core/core"
)

// saveDataMachine is the optional core capability that carries a title's
// writable storage across relaunches, the way handset flash survives an app
// exit. A machine that lacks it (no persistent storage) is simply not persisted.
type saveDataMachine interface {
	ExportSaveData() ([]byte, error)
	ImportSaveData([]byte) error
}

// currentInputHash returns the loaded title's SHA-256 identity, the key for its
// state slots and save-data file.
func (backend *Backend) currentInputHash() string {
	backend.mu.RLock()
	defer backend.mu.RUnlock()
	return backend.input.SHA256
}

// saveDataFrom reaches the save-data capability through the cheat wrapper.
func saveDataFrom(machine aramcore.Machine) (saveDataMachine, bool) {
	if machine == nil {
		return nil, false
	}
	if capability, ok := unwrapMachine(machine).(saveDataMachine); ok {
		return capability, true
	}
	capability, ok := machine.(saveDataMachine)
	return capability, ok
}

// saveDataFileFor returns the per-title save-data path, keyed by input SHA-256
// so each title keeps its own flash image alongside its state slots.
func (backend *Backend) saveDataFileFor(hash string) (string, error) {
	if hash == "" {
		return "", errors.New("loaded input has no SHA-256 identity")
	}
	backend.mu.RLock()
	root := backend.stateRoot
	backend.mu.RUnlock()
	if root == "" {
		configRoot, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(configRoot, "ARAM", "states")
	}
	directory := filepath.Join(root, hash)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(directory, "savedata.bin"), nil
}

// restoreSaveData loads a title's persisted writable storage into a freshly
// opened machine before it starts, so the guest's first read observes its
// saves. A missing file (first launch) is not an error.
func (backend *Backend) restoreSaveData(machine aramcore.Machine, hash string) error {
	capability, ok := saveDataFrom(machine)
	if !ok {
		return nil
	}
	path, err := backend.saveDataFileFor(hash)
	if err != nil {
		return fmt.Errorf("resolve game save path: %w", err)
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read game save data: %w", err)
	}
	if err := capability.ImportSaveData(data); err != nil {
		return fmt.Errorf("import game save data: %w", err)
	}
	return nil
}

// persistSaveData writes a title's writable storage to its per-title file. An
// empty export writes nothing, so a title that never saved leaves no file.
func (backend *Backend) persistSaveData(machine aramcore.Machine, hash string) error {
	capability, ok := saveDataFrom(machine)
	if !ok {
		return nil
	}
	data, err := capability.ExportSaveData()
	if err != nil {
		return fmt.Errorf("export game save data: %w", err)
	}
	return backend.writeSaveData(hash, data)
}

func (backend *Backend) writeSaveData(hash string, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	path, err := backend.saveDataFileFor(hash)
	if err != nil {
		return fmt.Errorf("resolve game save path: %w", err)
	}
	temporary := path + ".tmp"
	file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create temporary game save: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = file.Close()
			_ = os.Remove(temporary)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write temporary game save: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync temporary game save: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary game save: %w", err)
	}
	if err := replaceFileCrashSafely(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("replace game save: %w", err)
	}
	committed = true
	return nil
}

// FlushSaveData snapshots the loaded title's writable storage without stopping
// it. Mobile hosts call this during an Activity pause because Android may kill
// the process after it enters the background without giving the Go shell a
// normal close path.
func (backend *Backend) FlushSaveData() error {
	backend.operationMu.Lock()
	defer backend.operationMu.Unlock()

	machine := backend.currentMachine()
	if machine == nil {
		return nil
	}
	return backend.persistSaveData(machine, backend.currentInputHash())
}
