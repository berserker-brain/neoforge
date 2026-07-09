package schema

import (
	"log/slog"
	"strings"

	"github.com/berserker-brain/neoforge"
)

// Contribution is one package's schema input. A package supplies tag-carrying
// model structs to walk (Models), pre-built objects for the composite/exotic
// cases (Objects), and the names of any legacy constraints/indexes it is
// retiring in favor of derived-name equivalents (LegacyDrops). Each bounded
// context owns its own Contribution, keeping schema declarations local while
// the engine stays shared.
type Contribution struct {
	Models      []any
	Objects     []Object
}

// runner is the slice of *neoforge.CypherRepository the applier needs. It keeps
// the pipeline unit-testable without a live database.
type runner interface {
	RunQuery(*neoforge.CypherQuery)
}

// Failure records a schema object that could not be generated or applied.
type Failure struct {
	Name  string
	Query string
	Err   error
}

// Report is the outcome of an Apply run. It is returned (not just logged) so
// tests and callers can assert on it. Apply never returns an error and never
// panics: schema problems are logged, not fatal to boot.
type Report struct {
	Edition    string
	Applied    []string // created, or already present (IF NOT EXISTS no-op)
	Skipped    []string // Enterprise-only objects skipped on a non-Enterprise edition
	Dropped    []string // legacy names we issued drops for
	Failed     []Failure
	Drift      []string // present in the DB but declared by no model
	WalkErrors []error  // model declaration problems (missing GetLabel, bad tag, ...)
}

// Apply reconciles the declared schema against the live database. It is safe to
// call on every boot: it is additive (only CREATE ... IF NOT EXISTS), it skips
// Enterprise-only objects on Community, it reports drift without dropping it,
// and no failure aborts the process. The one destructive action is the
// LegacyDrops list, which is an explicit, bounded set of known names being
// migrated to the derived-name convention.
func Apply(repo runner, contributions ...Contribution) Report {
	var report Report

	// 1. Gather declared objects and legacy drops from every contribution.
	var objects []Object
	for _, c := range contributions {
		if len(c.Models) > 0 {
			objs, errs := Walk(c.Models)
			objects = append(objects, objs...)
			report.WalkErrors = append(report.WalkErrors, errs...)
		}
		objects = append(objects, c.Objects...)
	}
	for _, err := range report.WalkErrors {
		slog.Error("schema: model declaration error", "err", err)
	}

	// 2. Detect edition once so we can skip Enterprise-only objects cleanly.
	report.Edition = detectEdition(repo)
	enterprise := strings.EqualFold(report.Edition, "enterprise")

	// 4. Apply every declared object additively.
	for _, o := range objects {
		name := o.DerivedName()
		if o.Enterprise() && !enterprise {
			report.Skipped = append(report.Skipped, name)
			continue
		}
		stmt, err := o.Cypher()
		if err != nil {
			report.Failed = append(report.Failed, Failure{Name: name, Err: err})
			slog.Error("schema: cypher generation failed", "name", name, "err", err)
			continue
		}
		q := &neoforge.CypherQuery{Query: stmt}
		repo.RunQuery(q)
		if q.Error != nil {
			report.Failed = append(report.Failed, Failure{Name: name, Query: stmt, Err: q.Error})
			slog.Warn("schema: apply failed (likely violating data or unsupported feature)", "name", name, "err", q.Error)
			continue
		}
		report.Applied = append(report.Applied, name)
	}

	// 5. Report drift: schema in the DB that no model declares. Logged, never dropped.
	declared := make(map[string]bool, len(objects))
	for _, o := range objects {
		declared[o.DerivedName()] = true
	}
	report.Drift = detectDrift(repo, declared)
	for _, name := range report.Drift {
		slog.Info("schema: undeclared object present in DB (drift; not dropped)", "name", name)
	}

	// 6. One summary line.
	if len(report.Skipped) > 0 {
		slog.Info("schema: skipped Enterprise-only objects on non-Enterprise edition",
			"count", len(report.Skipped), "edition", report.Edition)
	}
	slog.Info("schema: apply complete",
		"edition", report.Edition,
		"applied", len(report.Applied),
		"skipped", len(report.Skipped),
		"dropped", len(report.Dropped),
		"failed", len(report.Failed),
		"drift", len(report.Drift),
		"walk_errors", len(report.WalkErrors),
	)
	return report
}

// detectEdition returns "enterprise" or "community". On any probe failure it
// assumes community so Enterprise-only objects are skipped rather than spammed
// as errors.
func detectEdition(repo runner) string {
	type editionRow struct {
		Edition string `key:"edition"`
	}
	var rows []editionRow
	q := &neoforge.CypherQuery{
		Query:   "CALL dbms.components() YIELD edition RETURN edition",
		Result:  &rows,
		EmptyOk: true,
	}
	repo.RunQuery(q)
	if q.Error != nil {
		slog.Warn("schema: edition probe failed; assuming community", "err", q.Error)
		return "community"
	}
	if len(rows) == 0 {
		slog.Warn("schema: edition probe returned no rows; assuming community")
		return "community"
	}
	return rows[0].Edition
}

// detectDrift returns the names of constraints and indexes that exist in the DB
// but are declared by no model. LOOKUP token indexes and constraint-backing
// indexes (which share their constraint's name) are excluded.
func detectDrift(repo runner, declared map[string]bool) []string {
	type schemaRow struct {
		Name string `key:"name"`
		Type string `key:"type,omitempty"`
	}

	var drift []string

	var constraints []schemaRow
	qc := &neoforge.CypherQuery{Query: "SHOW CONSTRAINTS YIELD name RETURN name", Result: &constraints, EmptyOk: true}
	repo.RunQuery(qc)
	if qc.Error != nil {
		slog.Warn("schema: SHOW CONSTRAINTS failed; skipping drift report", "err", qc.Error)
		return nil
	}
	constraintNames := make(map[string]bool, len(constraints))
	for _, r := range constraints {
		constraintNames[r.Name] = true
		if !declared[r.Name] {
			drift = append(drift, r.Name)
		}
	}

	var indexes []schemaRow
	qi := &neoforge.CypherQuery{Query: "SHOW INDEXES YIELD name, type RETURN name, type", Result: &indexes, EmptyOk: true}
	repo.RunQuery(qi)
	if qi.Error != nil {
		slog.Warn("schema: SHOW INDEXES failed; drift report is constraints-only", "err", qi.Error)
		return drift
	}
	for _, r := range indexes {
		if r.Type == "LOOKUP" || constraintNames[r.Name] {
			continue
		}
		if !declared[r.Name] {
			drift = append(drift, r.Name)
		}
	}
	return drift
}
