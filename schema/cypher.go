package schema

import (
	"fmt"
	"strings"
)

// nodeVar is the pattern variable used in generated DDL.
const nodeVar = "n"

// Cypher renders the CREATE statement for this object. All statements use
// IF NOT EXISTS so Apply is idempotent and additive.
func (o Object) Cypher() (string, error) {
	if o.Label == "" {
		return "", fmt.Errorf("schema: object %q has no label", o.DerivedName())
	}
	if len(o.Properties) == 0 {
		return "", fmt.Errorf("schema: object %q has no properties", o.DerivedName())
	}
	switch o.Kind {
	case KindConstraint:
		return o.constraintCypher()
	case KindIndex:
		return o.indexCypher()
	default:
		return "", fmt.Errorf("schema: object %q has unknown kind %d", o.DerivedName(), o.Kind)
	}
}

func (o Object) constraintCypher() (string, error) {
	name := o.DerivedName()
	head := fmt.Sprintf("CREATE CONSTRAINT %s IF NOT EXISTS FOR (%s:%s) REQUIRE ", name, nodeVar, o.Label)

	switch o.Constraint {
	case Unique:
		return head + o.propExpr() + " IS UNIQUE", nil
	case NodeKey:
		return head + o.propExpr() + " IS NODE KEY", nil
	case Exists:
		if len(o.Properties) != 1 {
			return "", fmt.Errorf("schema: %s: existence constraints are single-property only", name)
		}
		return head + o.propRef(o.Properties[0]) + " IS NOT NULL", nil
	case PropType:
		if len(o.Properties) != 1 {
			return "", fmt.Errorf("schema: %s: property-type constraints are single-property only", name)
		}
		if o.PropType == "" {
			return "", fmt.Errorf("schema: %s: property-type constraint has no type", name)
		}
		return head + o.propRef(o.Properties[0]) + " IS :: " + o.PropType, nil
	default:
		return "", fmt.Errorf("schema: %s: unknown constraint kind %q", name, o.Constraint)
	}
}

func (o Object) indexCypher() (string, error) {
	name := o.DerivedName()

	switch o.Index {
	case RangeIndex, TextIndex, PointIndex:
		if o.Index != RangeIndex && len(o.Properties) != 1 {
			return "", fmt.Errorf("schema: %s: %s indexes are single-property only", name, o.Index)
		}
		refs := o.propRefs()
		return fmt.Sprintf("CREATE %s INDEX %s IF NOT EXISTS FOR (%s:%s) ON (%s)",
			strings.ToUpper(string(o.Index)), name, nodeVar, o.Label, strings.Join(refs, ", ")), nil
	case FullTextIndex:
		stmt := fmt.Sprintf("CREATE FULLTEXT INDEX %s IF NOT EXISTS FOR (%s:%s) ON EACH [%s]",
			name, nodeVar, o.Label, strings.Join(o.propRefs(), ", "))
		return o.withOptions(stmt), nil
	case VectorIndex:
		if len(o.Properties) != 1 {
			return "", fmt.Errorf("schema: %s: vector indexes are single-property only", name)
		}
		stmt := fmt.Sprintf("CREATE VECTOR INDEX %s IF NOT EXISTS FOR (%s:%s) ON (%s)",
			name, nodeVar, o.Label, o.propRef(o.Properties[0]))
		return o.withOptions(stmt), nil
	default:
		return "", fmt.Errorf("schema: %s: unknown index kind %q", name, o.Index)
	}
}

func (o Object) withOptions(stmt string) string {
	if o.Options == "" {
		return stmt
	}
	return stmt + " " + o.Options
}

// propRef renders a single "n.prop" reference.
func (o Object) propRef(prop string) string {
	return nodeVar + "." + prop
}

// propRefs renders every property as "n.prop".
func (o Object) propRefs() []string {
	refs := make([]string, len(o.Properties))
	for i, p := range o.Properties {
		refs[i] = o.propRef(p)
	}
	return refs
}

// propExpr renders the REQUIRE target: "n.p" for one property, "(n.p1, n.p2)"
// for a composite.
func (o Object) propExpr() string {
	if len(o.Properties) == 1 {
		return o.propRef(o.Properties[0])
	}
	return "(" + strings.Join(o.propRefs(), ", ") + ")"
}
