package integration

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"strings"
)

// These inputs have the same Inotia 2 save-slot schema. The ARAM variant only
// changes offline startup authentication and the blessed-seal rare selection.
// Keep migration directional and hash-scoped: no other title/version is
// permitted to bypass the ordinary WFS input identity check.
const (
	inotia2ARAMInputHash = "0902cae187fa958a42eac1b84ac59da5b35c6a80bc6dd6fec1a1aa71e13d9399"
	inotia2RareInputHash = "3ed920a9143c0f8971f20b8fd87b8ac9582e441f9276c9edefe70af6c6277b39"
	inotia2OriginalInputHash = "974e0df9ab1e51751efdef21a0fe324ad79917f41cd049fcfaf7c46c14f9e324"
)

func compatibleLegacyWFSIdentity(current, identity string) bool {
	if strings.EqualFold(current, identity) {
		return true
	}
	return strings.EqualFold(current, inotia2ARAMInputHash) &&
		(strings.EqualFold(identity, inotia2RareInputHash) ||
			strings.EqualFold(identity, inotia2OriginalInputHash))
}

// prepareLegacyWFSFiles retains the destination package's startup certificate
// when importing Inotia 2 saves into the authentication-fixed ARAM package.
// cert.c2s belongs to that executable's subscriber identity, not to character
// progress. Read it from the loaded package even on a first launch, rather than
// trusting a potentially stale certificate in the old emulator's backup.
// All save slots and other files remain byte-for-byte unchanged.
func (backend *Backend) prepareLegacyWFSFiles(current, identity string, files map[string][]byte) (map[string][]byte, error) {
	if !compatibleLegacyWFSIdentity(current, identity) {
		return nil, fmt.Errorf(
			"this legacy WFS save belongs to a different title (%s…), not the loaded one",
			shortSaveHash(identity),
		)
	}
	if !strings.EqualFold(current, inotia2ARAMInputHash) {
		return files, nil
	}
	backend.mu.RLock()
	source := backend.source
	backend.mu.RUnlock()
	if source.ReaderAt == nil || source.Size <= 0 {
		return nil, errors.New("legacy WFS save: destination package is unavailable for authentication migration")
	}
	archive, err := zip.NewReader(source.ReaderAt, source.Size)
	if err != nil {
		return nil, fmt.Errorf("legacy WFS save: read destination package: %w", err)
	}
	var certificate []byte
	for _, file := range archive.File {
		name := strings.ToLower(file.Name)
		if name != "p/cert.c2s" && !strings.HasSuffix(name, "/p/cert.c2s") {
			continue
		}
		if certificate != nil || file.UncompressedSize64 == 0 || file.UncompressedSize64 > 4096 {
			return nil, errors.New("legacy WFS save: destination package has an invalid authentication file")
		}
		reader, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("legacy WFS save: open destination authentication file: %w", err)
		}
		certificate, err = io.ReadAll(io.LimitReader(reader, 4097))
		closeErr := reader.Close()
		if err != nil {
			return nil, fmt.Errorf("legacy WFS save: read destination authentication file: %w", err)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("legacy WFS save: close destination authentication file: %w", closeErr)
		}
		if len(certificate) == 0 || len(certificate) > 4096 {
			return nil, errors.New("legacy WFS save: destination authentication file exceeds size limit")
		}
	}
	if certificate == nil {
		return nil, errors.New("legacy WFS save: destination package is missing its authentication file")
	}
	prepared := make(map[string][]byte, len(files)+1)
	for name, contents := range files {
		prepared[name] = contents
	}
	prepared["/cert.c2s"] = certificate
	return prepared, nil
}
