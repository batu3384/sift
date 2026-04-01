package parity

import (
	"strings"
	"testing"
)

func TestLoadMatrix(t *testing.T) {
	t.Parallel()
	matrix, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if matrix.MoleCommit == "" {
		t.Fatal("expected mole commit to be populated")
	}
	if matrix.Upstream.BaselineCommit == "" || matrix.Upstream.CompareRange == "" {
		t.Fatalf("expected upstream baseline metadata to be populated, got %+v", matrix.Upstream)
	}
	if len(matrix.Features) < 20 {
		t.Fatalf("expected a non-trivial parity matrix, got %d features", len(matrix.Features))
	}
}

func TestSummarizeMatrix(t *testing.T) {
	t.Parallel()
	summary := Summarize(MustLoad())
	if summary.Covered == 0 {
		t.Fatalf("expected covered features in parity matrix, got %+v", summary)
	}
	if summary.Missing != 0 {
		t.Fatalf("expected zero missing features after parity closure, got %+v", summary)
	}
	if summary.Partial != 0 {
		t.Fatalf("expected no partial features after premium parity closure, got %+v", summary)
	}
	if summary.RegressionRisk != 0 {
		t.Fatalf("expected no regression-risk features after baseline refresh, got %+v", summary)
	}
}

func TestMatrixPlannedWavesCarryEvidence(t *testing.T) {
	t.Parallel()
	matrix := MustLoad()
	if matrix.Upstream.CompareFromCommit == "" || matrix.Upstream.CompareRange == "" {
		t.Fatalf("expected compare metadata to be populated, got %+v", matrix.Upstream)
	}
	if len(matrix.Upstream.ChangedFiles) == 0 {
		t.Fatalf("expected changed files to be populated, got %+v", matrix.Upstream)
	}
	for _, feature := range matrix.Features {
		if strings.TrimSpace(feature.PlannedWave) == "" {
			continue
		}
		if strings.TrimSpace(feature.MoleSurface) == "" {
			t.Fatalf("expected mole surface for planned wave feature %s", feature.ID)
		}
		if strings.TrimSpace(feature.Evidence) == "" {
			t.Fatalf("expected evidence for planned wave feature %s", feature.ID)
		}
		if strings.TrimSpace(feature.Impact) == "" {
			t.Fatalf("expected impact for planned wave feature %s", feature.ID)
		}
	}
}
