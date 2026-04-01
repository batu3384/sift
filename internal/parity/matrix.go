package parity

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed matrix.json
var embeddedFiles embed.FS

type Status string

const (
	StatusCovered        Status = "covered"
	StatusPartial        Status = "partial"
	StatusMissing        Status = "missing"
	StatusRegressionRisk Status = "regression-risk"
	StatusBetterThanMole Status = "better-than-mole"
)

type Upstream struct {
	Repo              string   `json:"repo"`
	CanonicalClone    string   `json:"canonical_clone,omitempty"`
	BaselineCommit    string   `json:"baseline_commit"`
	BaselineDate      string   `json:"baseline_date,omitempty"`
	CompareFromCommit string   `json:"compare_from_commit,omitempty"`
	CompareRange      string   `json:"compare_range,omitempty"`
	ChangedFiles      []string `json:"changed_files,omitempty"`
}

type Feature struct {
	Command     string `json:"command"`
	Area        string `json:"area,omitempty"`
	Group       string `json:"group,omitempty"`
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      Status `json:"status"`
	MoleSurface string `json:"mole_surface,omitempty"`
	SiftSurface string `json:"sift_surface,omitempty"`
	Notes       string `json:"notes,omitempty"`
	Evidence    string `json:"evidence,omitempty"`
	Impact      string `json:"impact,omitempty"`
	PlannedWave string `json:"planned_wave,omitempty"`
}

type Matrix struct {
	Version    int       `json:"version"`
	UpdatedAt  string    `json:"updated_at"`
	MoleCommit string    `json:"mole_commit"`
	Upstream   Upstream  `json:"upstream"`
	Features   []Feature `json:"features"`
}

type Summary struct {
	Covered        int
	Partial        int
	Missing        int
	RegressionRisk int
	BetterThanMole int
}

func Load() (Matrix, error) {
	raw, err := embeddedFiles.ReadFile("matrix.json")
	if err != nil {
		return Matrix{}, err
	}
	var matrix Matrix
	if err := json.Unmarshal(raw, &matrix); err != nil {
		return Matrix{}, err
	}
	if matrix.Version == 0 {
		return Matrix{}, fmt.Errorf("parity matrix version is required")
	}
	if strings.TrimSpace(matrix.Upstream.BaselineCommit) == "" {
		matrix.Upstream.BaselineCommit = matrix.MoleCommit
	}
	if strings.TrimSpace(matrix.MoleCommit) == "" {
		matrix.MoleCommit = matrix.Upstream.BaselineCommit
	}
	if strings.TrimSpace(matrix.MoleCommit) == "" {
		return Matrix{}, fmt.Errorf("parity matrix mole commit is required")
	}
	for idx := range matrix.Features {
		feature := &matrix.Features[idx]
		if strings.TrimSpace(feature.Area) == "" {
			feature.Area = feature.Group
		}
		if strings.TrimSpace(feature.Command) == "" || strings.TrimSpace(feature.ID) == "" {
			return Matrix{}, fmt.Errorf("parity matrix contains a feature without command or id")
		}
		if strings.TrimSpace(feature.Area) == "" {
			return Matrix{}, fmt.Errorf("parity matrix contains a feature without area or group: %s", feature.ID)
		}
		switch feature.Status {
		case StatusCovered, StatusPartial, StatusMissing, StatusRegressionRisk, StatusBetterThanMole:
		default:
			return Matrix{}, fmt.Errorf("invalid parity status %q for %s", feature.Status, feature.ID)
		}
	}
	return matrix, nil
}

func MustLoad() Matrix {
	matrix, err := Load()
	if err != nil {
		panic(err)
	}
	return matrix
}

func Summarize(matrix Matrix) Summary {
	var summary Summary
	for _, feature := range matrix.Features {
		switch feature.Status {
		case StatusCovered:
			summary.Covered++
		case StatusPartial:
			summary.Partial++
		case StatusMissing:
			summary.Missing++
		case StatusRegressionRisk:
			summary.RegressionRisk++
		case StatusBetterThanMole:
			summary.BetterThanMole++
		}
	}
	return summary
}
