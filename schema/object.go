// Package schema lets storage models declare their Neo4j constraints and
// indexes programmatically -- via `neo:"..."` struct tags for the simple
// single-property cases, or via the SchemaObjects() escape hatch for composite
// and option-bearing objects (FULLTEXT, VECTOR). A reflection walker turns
// both sources into a normalized []Object, and Apply reconciles them against a
// live database additively (never destructive) at startup.
//
// See the design notes in the models manifest (storage/models/manifest.go) for
// the tag vocabulary and naming rules.
package schema

import (
	"fmt"
	"strings"
)

// Kind distinguishes a constraint from an index.
type Kind int

const (
	KindConstraint Kind = iota
	KindIndex
)

// Scope selects whether an object targets a node label or a relationship type.
// NodeScope is the zero value, so objects built before relationship support --
// and every node constructor -- default to node scope with no changes.
type Scope int

const (
	NodeScope Scope = iota
	RelScope
)

// ConstraintKind enumerates the constraint flavors we generate. The string
// values double as the suffix in a derived name (e.g. Business_name_unique).
type ConstraintKind string

const (
	Unique   ConstraintKind = "unique"
	NodeKey  ConstraintKind = "key"
	Exists   ConstraintKind = "exists"
	PropType ConstraintKind = "type"
)

// IndexKind enumerates the index flavors we generate. As with ConstraintKind,
// the string values are the derived-name suffix.
type IndexKind string

const (
	RangeIndex    IndexKind = "range"
	TextIndex     IndexKind = "text"
	PointIndex    IndexKind = "point"
	FullTextIndex IndexKind = "fulltext"
	VectorIndex   IndexKind = "vector"
)

// Object is a single schema element (one constraint or one index) in a
// normalized, source-agnostic form. Both the struct-tag walker and the
// SchemaObjects() escape hatch produce these; Apply consumes them.
type Object struct {
	Kind Kind
	// Scope selects whether Label names a node label (NodeScope, the zero value)
	// or a relationship type (RelScope). It controls the pattern rendered in DDL:
	// (n:Label) for nodes, ()-[r:Label]-() for relationships.
	Scope Scope
	// Label is the single node label or relationship type the element applies to.
	// Required.
	Label string
	// Properties are the Neo4j property names (json-tag values). One entry for
	// simple tag-derived objects; multiple for composite constraints / fulltext.
	Properties []string
	// Name, if set, is used verbatim; otherwise DerivedName() is used. Only the
	// escape hatch sets it (to pin a legacy name); tag-derived objects leave it
	// empty so the naming convention is uniform.
	Name string

	// Constraint fields (Kind == KindConstraint).
	Constraint ConstraintKind
	// PropType is the Neo4j type for a PropType constraint, e.g. "STRING". The
	// walker derives it from the Go field kind.
	PropType string

	// Index fields (Kind == KindIndex).
	Index IndexKind
	// Options is appended verbatim after the index target, e.g.
	// "OPTIONS { indexConfig: { `vector.dimensions`: 1536, ... } }".
	Options string
}

// Enterprise reports whether creating this object requires Neo4j Enterprise.
// For nodes, uniqueness constraints and all plain indexes are on Community while
// node-key, existence, and property-type constraints are Enterprise-only. For
// relationships, every constraint (including uniqueness) is Enterprise-only;
// relationship indexes remain available on Community.
func (o Object) Enterprise() bool {
	if o.Kind != KindConstraint {
		return false
	}
	if o.Scope == RelScope {
		return true // all relationship constraints are Enterprise-only
	}
	switch o.Constraint {
	case NodeKey, Exists, PropType:
		return true
	}
	return false
}

// suffix returns the trailing token used in a derived name.
func (o Object) suffix() string {
	if o.Kind == KindConstraint {
		return string(o.Constraint)
	}
	return string(o.Index)
}

// DerivedName returns o.Name if set, otherwise the convention-derived name:
// <CaseAccurateLabel>_<jsonProp(s) joined by _>_<kind>. The label keeps its
// exact case; properties are the json-tag values verbatim.
func (o Object) DerivedName() string {
	if o.Name != "" {
		return o.Name
	}
	parts := make([]string, 0, len(o.Properties)+2)
	parts = append(parts, o.Label)
	parts = append(parts, o.Properties...)
	parts = append(parts, o.suffix())
	return strings.Join(parts, "_")
}

// -- Escape-hatch constructors -------------------------------------------------
// Models that need composite or option-bearing objects return these from
// SchemaObjects(); the Label may be left empty and the walker fills it from
// GetLabel().

// UniqueConstraint builds a (possibly composite) uniqueness constraint.
func UniqueConstraint(label string, properties ...string) Object {
	return Object{Kind: KindConstraint, Constraint: Unique, Label: label, Properties: properties}
}

// NodeKeyConstraint builds a (possibly composite) node-key constraint. Enterprise.
func NodeKeyConstraint(label string, properties ...string) Object {
	return Object{Kind: KindConstraint, Constraint: NodeKey, Label: label, Properties: properties}
}

// ExistsConstraint builds a single-property existence constraint. Enterprise.
func ExistsConstraint(label, property string) Object {
	return Object{Kind: KindConstraint, Constraint: Exists, Label: label, Properties: []string{property}}
}

// FullText builds a full-text index over one or more properties. options, when
// non-empty, must be a complete "OPTIONS { ... }" clause.
func FullText(label string, properties []string, options string) Object {
	return Object{Kind: KindIndex, Index: FullTextIndex, Label: label, Properties: properties, Options: options}
}

// Vector builds a vector index on a single property with the given dimensions
// and similarity function (e.g. "cosine").
func Vector(label, property string, dimensions int, similarity string) Object {
	opts := fmt.Sprintf("OPTIONS { indexConfig: { `vector.dimensions`: %d, `vector.similarity_function`: '%s' } }", dimensions, similarity)
	return Object{Kind: KindIndex, Index: VectorIndex, Label: label, Properties: []string{property}, Options: opts}
}

// -- Relationship escape-hatch constructors -----------------------------------
// These mirror the node constructors but target a relationship type. relType may
// be left empty when returned from a RelLabeler model's SchemaObjects(); Walk
// fills it from GetRelType(). Every relationship constraint is Enterprise-only.

// RelUniqueConstraint builds a (possibly composite) relationship uniqueness constraint. Enterprise.
func RelUniqueConstraint(relType string, properties ...string) Object {
	return Object{Kind: KindConstraint, Constraint: Unique, Scope: RelScope, Label: relType, Properties: properties}
}

// RelKeyConstraint builds a (possibly composite) relationship key constraint. Enterprise.
func RelKeyConstraint(relType string, properties ...string) Object {
	return Object{Kind: KindConstraint, Constraint: NodeKey, Scope: RelScope, Label: relType, Properties: properties}
}

// RelExistsConstraint builds a single-property relationship existence constraint. Enterprise.
func RelExistsConstraint(relType, property string) Object {
	return Object{Kind: KindConstraint, Constraint: Exists, Scope: RelScope, Label: relType, Properties: []string{property}}
}

// RelFullText builds a full-text index over one or more relationship properties.
// options, when non-empty, must be a complete "OPTIONS { ... }" clause.
func RelFullText(relType string, properties []string, options string) Object {
	return Object{Kind: KindIndex, Index: FullTextIndex, Scope: RelScope, Label: relType, Properties: properties, Options: options}
}

// RelVector builds a vector index on a single relationship property with the
// given dimensions and similarity function (e.g. "cosine").
func RelVector(relType, property string, dimensions int, similarity string) Object {
	opts := fmt.Sprintf("OPTIONS { indexConfig: { `vector.dimensions`: %d, `vector.similarity_function`: '%s' } }", dimensions, similarity)
	return Object{Kind: KindIndex, Index: VectorIndex, Scope: RelScope, Label: relType, Properties: []string{property}, Options: opts}
}
