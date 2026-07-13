package schema

import (
	"strings"
	"testing"

	"github.com/berserker-brain/neoforge"
)

// recordingRunner records every statement it is handed. It leaves read results
// empty, so detectEdition falls back to "community" and detectDrift finds
// nothing -- exactly the conditions to exercise Enterprise-skip and drop
// ordering without a live database.
type recordingRunner struct{ queries []string }

func (r *recordingRunner) RunQuery(q *neoforge.CypherQuery) {
	r.queries = append(r.queries, q.Query)
}

func TestApplyCommunitySkipsEnterprise(t *testing.T) {
	rr := &recordingRunner{}
	report := Apply(rr, Contribution{
		Objects: []Object{
			UniqueConstraint("Widget", "id"),   // Community-safe -> applied
			ExistsConstraint("Widget", "name"), // Enterprise-only -> skipped on Community
		},
	})

	if report.Edition != "community" {
		t.Fatalf("edition: want community, got %q", report.Edition)
	}
	if got := strings.Join(report.Applied, ","); got != "Widget_id_unique" {
		t.Errorf("applied: want Widget_id_unique, got %q", got)
	}
	if got := strings.Join(report.Skipped, ","); got != "Widget_name_exists" {
		t.Errorf("skipped: want Widget_name_exists, got %q", got)
	}

	// The Enterprise-only exists constraint must never have been sent.
	for _, q := range rr.queries {
		if strings.Contains(q, "IS NOT NULL") {
			t.Errorf("enterprise exists constraint should have been skipped, got %q", q)
		}
	}
}

// tagModel exercises every tag directive plus the escape hatch.
type tagModel struct {
	Labels    []string `db:"labels"`
	ID        string   `json:"id,omitempty" neo:"unique"`
	Handle    string   `json:"handle" neo:"index"`
	Bio       string   `json:"bio" neo:"index:text"`
	Priority  int64    `json:"priority" neo:"type"`
	Roles     []string `json:"roles" neo:"exists,type"`
	Ignored   string   `json:"ignored"`
	NoPersist string   `json:"-" neo:"unique"`
}

func (tagModel) GetLabel() string { return "Widget" }

func (tagModel) SchemaObjects() []Object {
	return []Object{
		UniqueConstraint("", "tenant_id", "slug"), // composite, label filled by Walk
		Vector("Widget", "embedding", 1536, "cosine"),
	}
}

func TestWalkTagsAndEscapeHatch(t *testing.T) {
	objs, errs := Walk([]any{tagModel{}})

	// NoPersist has json:"-" so it must produce a walk error, not an object.
	if len(errs) != 1 {
		t.Fatalf("want 1 walk error (NoPersist), got %d: %v", len(errs), errs)
	}

	got := map[string]string{}
	for _, o := range objs {
		stmt, err := o.Cypher()
		if err != nil {
			t.Fatalf("Cypher() for %s: %v", o.DerivedName(), err)
		}
		got[o.DerivedName()] = stmt
	}

	want := map[string]string{
		"Widget_id_unique":             "CREATE CONSTRAINT Widget_id_unique IF NOT EXISTS FOR (n:Widget) REQUIRE n.id IS UNIQUE",
		"Widget_handle_range":          "CREATE RANGE INDEX Widget_handle_range IF NOT EXISTS FOR (n:Widget) ON (n.handle)",
		"Widget_bio_text":              "CREATE TEXT INDEX Widget_bio_text IF NOT EXISTS FOR (n:Widget) ON (n.bio)",
		"Widget_priority_type":         "CREATE CONSTRAINT Widget_priority_type IF NOT EXISTS FOR (n:Widget) REQUIRE n.priority IS :: INTEGER",
		"Widget_roles_exists":          "CREATE CONSTRAINT Widget_roles_exists IF NOT EXISTS FOR (n:Widget) REQUIRE n.roles IS NOT NULL",
		"Widget_roles_type":            "CREATE CONSTRAINT Widget_roles_type IF NOT EXISTS FOR (n:Widget) REQUIRE n.roles IS :: LIST<STRING NOT NULL>",
		"Widget_tenant_id_slug_unique": "CREATE CONSTRAINT Widget_tenant_id_slug_unique IF NOT EXISTS FOR (n:Widget) REQUIRE (n.tenant_id, n.slug) IS UNIQUE",
		"Widget_embedding_vector":      "CREATE VECTOR INDEX Widget_embedding_vector IF NOT EXISTS FOR (n:Widget) ON (n.embedding) OPTIONS { indexConfig: { `vector.dimensions`: 1536, `vector.similarity_function`: 'cosine' } }",
	}

	if len(got) != len(want) {
		t.Fatalf("object count: want %d, got %d (%v)", len(want), len(got), keys(got))
	}
	for name, wantStmt := range want {
		if got[name] != wantStmt {
			t.Errorf("%s:\n  want %q\n  got  %q", name, wantStmt, got[name])
		}
	}
}

type subsumeModel struct {
	// unique + range index -> range index dropped (unique provides the backing index)
	A string `json:"a" neo:"unique,index"`
	// unique + text index -> both kept (backing index does not serve text queries)
	B string `json:"b" neo:"unique,index:text"`
	// key subsumes unique, exists, and the range index; type is independent
	C string `json:"c" neo:"key,unique,exists,index,type"`
}

func (subsumeModel) GetLabel() string { return "Sub" }

func TestSubsumeRedundant(t *testing.T) {
	objs, errs := Walk([]any{subsumeModel{}})
	if len(errs) != 0 {
		t.Fatalf("unexpected walk errors: %v", errs)
	}
	got := map[string]bool{}
	for _, o := range objs {
		got[o.DerivedName()] = true
	}
	want := []string{
		"Sub_a_unique", // A: index dropped
		"Sub_b_unique", // B: unique kept
		"Sub_b_text",   // B: text index kept
		"Sub_c_key",    // C: key kept
		"Sub_c_type",   // C: type kept
	}
	dropped := []string{
		"Sub_a_range",  // subsumed by unique
		"Sub_c_unique", // subsumed by key
		"Sub_c_exists", // subsumed by key
		"Sub_c_range",  // subsumed by key
	}
	for _, name := range want {
		if !got[name] {
			t.Errorf("want %s present, missing", name)
		}
	}
	for _, name := range dropped {
		if got[name] {
			t.Errorf("want %s subsumed, but present", name)
		}
	}
	if len(objs) != len(want) {
		t.Errorf("object count: want %d, got %d (%v)", len(want), len(objs), got)
	}
}

// relModel exercises the relationship path: tag directives plus the escape hatch,
// all scoped to a relationship type via GetRelType.
type relModel struct {
	Since  string `json:"since" neo:"unique"`
	Weight int64  `json:"weight" neo:"index"`
	Note   string `json:"note" neo:"index:text"`
	Role   string `json:"role" neo:"exists,type"`
}

func (relModel) GetRelType() string { return "WORKS_AT" }

func (relModel) SchemaObjects() []Object {
	return []Object{
		RelUniqueConstraint("", "tenant_id", "seq"), // composite, type filled by Walk
		RelVector("WORKS_AT", "embedding", 8, "cosine"),
	}
}

func TestWalkRelationshipTagsAndEscapeHatch(t *testing.T) {
	objs, errs := Walk([]any{relModel{}})
	if len(errs) != 0 {
		t.Fatalf("unexpected walk errors: %v", errs)
	}

	got := map[string]string{}
	for _, o := range objs {
		if o.Scope != RelScope {
			t.Errorf("%s: want RelScope, got node scope", o.DerivedName())
		}
		stmt, err := o.Cypher()
		if err != nil {
			t.Fatalf("Cypher() for %s: %v", o.DerivedName(), err)
		}
		got[o.DerivedName()] = stmt
	}

	want := map[string]string{
		"WORKS_AT_since_unique":         "CREATE CONSTRAINT WORKS_AT_since_unique IF NOT EXISTS FOR ()-[r:WORKS_AT]-() REQUIRE r.since IS UNIQUE",
		"WORKS_AT_weight_range":         "CREATE RANGE INDEX WORKS_AT_weight_range IF NOT EXISTS FOR ()-[r:WORKS_AT]-() ON (r.weight)",
		"WORKS_AT_note_text":            "CREATE TEXT INDEX WORKS_AT_note_text IF NOT EXISTS FOR ()-[r:WORKS_AT]-() ON (r.note)",
		"WORKS_AT_role_exists":          "CREATE CONSTRAINT WORKS_AT_role_exists IF NOT EXISTS FOR ()-[r:WORKS_AT]-() REQUIRE r.role IS NOT NULL",
		"WORKS_AT_role_type":            "CREATE CONSTRAINT WORKS_AT_role_type IF NOT EXISTS FOR ()-[r:WORKS_AT]-() REQUIRE r.role IS :: STRING",
		"WORKS_AT_tenant_id_seq_unique": "CREATE CONSTRAINT WORKS_AT_tenant_id_seq_unique IF NOT EXISTS FOR ()-[r:WORKS_AT]-() REQUIRE (r.tenant_id, r.seq) IS UNIQUE",
		"WORKS_AT_embedding_vector":     "CREATE VECTOR INDEX WORKS_AT_embedding_vector IF NOT EXISTS FOR ()-[r:WORKS_AT]-() ON (r.embedding) OPTIONS { indexConfig: { `vector.dimensions`: 8, `vector.similarity_function`: 'cosine' } }",
	}

	if len(got) != len(want) {
		t.Fatalf("object count: want %d, got %d (%v)", len(want), len(got), keys(got))
	}
	for name, wantStmt := range want {
		if got[name] != wantStmt {
			t.Errorf("%s:\n  want %q\n  got  %q", name, wantStmt, got[name])
		}
	}
}

func TestRelKeyConstraintCypher(t *testing.T) {
	o := RelKeyConstraint("WORKS_AT", "id")
	stmt, err := o.Cypher()
	if err != nil {
		t.Fatalf("Cypher(): %v", err)
	}
	want := "CREATE CONSTRAINT WORKS_AT_id_key IF NOT EXISTS FOR ()-[r:WORKS_AT]-() REQUIRE r.id IS RELATIONSHIP KEY"
	if stmt != want {
		t.Errorf("want %q, got %q", want, stmt)
	}
}

func TestRelationshipConstraintsAreEnterprise(t *testing.T) {
	// Every relationship constraint is Enterprise-only, uniqueness included.
	cases := []Object{
		RelUniqueConstraint("R", "id"),
		RelKeyConstraint("R", "id"),
		RelExistsConstraint("R", "id"),
	}
	for _, o := range cases {
		if !o.Enterprise() {
			t.Errorf("%s: want Enterprise, got community", o.DerivedName())
		}
	}
	// Relationship indexes remain available on Community.
	if RelVector("R", "v", 4, "cosine").Enterprise() {
		t.Errorf("relationship vector index should not be Enterprise")
	}
}

func TestApplyCommunitySkipsRelationshipConstraints(t *testing.T) {
	rr := &recordingRunner{}
	report := Apply(rr, Contribution{
		Objects: []Object{
			RelUniqueConstraint("WORKS_AT", "id"), // Enterprise (rel) -> skipped on Community
			{Kind: KindIndex, Index: RangeIndex, Scope: RelScope, Label: "WORKS_AT", Properties: []string{"since"}}, // Community-safe -> applied
		},
	})
	if got := strings.Join(report.Skipped, ","); got != "WORKS_AT_id_unique" {
		t.Errorf("skipped: want WORKS_AT_id_unique, got %q", got)
	}
	if got := strings.Join(report.Applied, ","); got != "WORKS_AT_since_range" {
		t.Errorf("applied: want WORKS_AT_since_range, got %q", got)
	}
}

type unlabeled struct {
	ID string `json:"id" neo:"unique"`
}

func TestWalkRequiresLabeler(t *testing.T) {
	_, errs := Walk([]any{unlabeled{}})
	if len(errs) != 1 {
		t.Fatalf("want 1 error for missing GetLabel, got %d: %v", len(errs), errs)
	}
}

func TestEnterpriseDetection(t *testing.T) {
	cases := []struct {
		obj  Object
		want bool
	}{
		{UniqueConstraint("X", "id"), false},
		{ExistsConstraint("X", "id"), true},
		{NodeKeyConstraint("X", "id"), true},
		{Object{Kind: KindConstraint, Constraint: PropType, Label: "X", Properties: []string{"id"}, PropType: "STRING"}, true},
		{Object{Kind: KindIndex, Index: RangeIndex, Label: "X", Properties: []string{"id"}}, false},
	}
	for _, c := range cases {
		if got := c.obj.Enterprise(); got != c.want {
			t.Errorf("%s Enterprise(): want %v, got %v", c.obj.DerivedName(), c.want, got)
		}
	}
}

func TestPinnedNameOverridesDerived(t *testing.T) {
	o := Object{Kind: KindConstraint, Constraint: Unique, Label: "X", Properties: []string{"id"}, Name: "legacy_pinned"}
	if o.DerivedName() != "legacy_pinned" {
		t.Errorf("pinned name: want legacy_pinned, got %s", o.DerivedName())
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
