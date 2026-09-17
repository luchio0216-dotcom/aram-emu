package integration

import (
	"testing"

	"github.com/mirusu400/aram-core/application"
)

func TestBREWDiagnosticsShapeMatchesCoreFrameStats(t *testing.T) {
	stats := application.BREWFrameStats{PresentCount: 3, FrameValid: true}
	diagnostics := BREWDiagnostics{
		PresentCount: stats.PresentCount,
		FrameValid:   stats.FrameValid,
	}
	if diagnostics.PresentCount != 3 || !diagnostics.FrameValid {
		t.Fatalf("BREW diagnostics = %+v", diagnostics)
	}
}
