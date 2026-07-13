package schema

import (
	"fmt"
	"reflect"
	"strings"
)

// Labeler is implemented by every node model that contributes schema. The
// returned label is the single node label constraints/indexes are declared
// against. Walk refuses (with an error) to process a model that implements
// neither Labeler nor RelLabeler.
type Labeler interface {
	GetLabel() string
}

// RelLabeler is implemented by relationship models that contribute schema. The
// returned type is the single relationship type constraints/indexes are declared
// against. A model implements Labeler or RelLabeler, not both; if it implements
// both, Labeler (node scope) wins.
type RelLabeler interface {
	GetRelType() string
}

// Provider is the optional escape hatch for schema that a single-property tag
// cannot express: composite constraints, full-text and vector indexes. Objects
// with an empty Label inherit the model's GetLabel().
type Provider interface {
	SchemaObjects() []Object
}

// Walk reflects over the given model instances and returns the schema objects
// declared by their `neo` struct tags plus any returned from SchemaObjects().
// It returns a partial result alongside a slice of errors: a model that fails
// (missing GetLabel, a neo tag on a field with no json tag, an undecodable
// type) is reported but does not abort the others, so Apply can log loudly and
// still apply everything valid.
func Walk(models []any) ([]Object, []error) {
	var objects []Object
	var errs []error

	for _, m := range models {
		var label string
		var scope Scope
		switch labeler := m.(type) {
		case Labeler:
			label, scope = labeler.GetLabel(), NodeScope
			if label == "" {
				errs = append(errs, fmt.Errorf("schema: %T.GetLabel() returned an empty label", m))
				continue
			}
		case RelLabeler:
			label, scope = labeler.GetRelType(), RelScope
			if label == "" {
				errs = append(errs, fmt.Errorf("schema: %T.GetRelType() returned an empty relationship type", m))
				continue
			}
		default:
			errs = append(errs, fmt.Errorf("schema: %T implements neither Labeler (GetLabel) nor RelLabeler (GetRelType); cannot contribute schema", m))
			continue
		}

		objs, tagErrs := walkTags(m, label, scope)
		objects = append(objects, objs...)
		errs = append(errs, tagErrs...)

		if provider, ok := m.(Provider); ok {
			for _, o := range provider.SchemaObjects() {
				if o.Label == "" {
					o.Label = label
				}
				objects = append(objects, o)
			}
		}
	}

	return objects, errs
}

// walkTags scans a single struct's fields for `neo` tags.
func walkTags(m any, label string, scope Scope) ([]Object, []error) {
	v := reflect.ValueOf(m)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, []error{fmt.Errorf("schema: %s: expected a struct, got %s", label, v.Kind())}
	}

	var objs []Object
	var errs []error
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		neoTag := field.Tag.Get("neo")
		if neoTag == "" || neoTag == "-" {
			continue
		}

		prop, ok := jsonProp(field)
		if !ok {
			errs = append(errs, fmt.Errorf("schema: %s.%s has a neo tag but no usable json tag to source the property name from", label, field.Name))
			continue
		}

		var fieldObjs []Object
		for _, directive := range strings.Split(neoTag, ",") {
			directive = strings.TrimSpace(directive)
			if directive == "" {
				continue
			}
			obj, err := directiveToObject(directive, label, prop, field, scope)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			fieldObjs = append(fieldObjs, obj)
		}
		objs = append(objs, subsumeRedundant(fieldObjs)...)
	}
	return objs, errs
}

// subsumeRedundant drops objects on a single field that Neo4j already provides
// implicitly, so we don't create duplicate schema:
//
//   - A uniqueness or node-key constraint auto-creates a backing RANGE index,
//     so an explicit range index on the same field is redundant. TEXT and POINT
//     indexes are NOT provided by the backing index (they serve different query
//     shapes), so they are kept.
//   - A node-key constraint is uniqueness + existence combined, so unique and
//     exists on the same field are redundant alongside it.
//
// Property-type constraints are independent and never subsumed.
func subsumeRedundant(objs []Object) []Object {
	var hasKey, hasUnique bool
	for _, o := range objs {
		if o.Kind == KindConstraint {
			switch o.Constraint {
			case NodeKey:
				hasKey = true
			case Unique:
				hasUnique = true
			}
		}
	}
	if !hasKey && !hasUnique {
		return objs
	}

	out := make([]Object, 0, len(objs))
	for _, o := range objs {
		if hasKey && o.Kind == KindConstraint && (o.Constraint == Unique || o.Constraint == Exists) {
			continue // NODE KEY already enforces uniqueness and existence
		}
		if (hasKey || hasUnique) && o.Kind == KindIndex && o.Index == RangeIndex {
			continue // the uniqueness/node-key constraint provides the backing range index
		}
		out = append(out, o)
	}
	return out
}

// directiveToObject maps one comma-separated tag token to an Object. scope is
// stamped onto every returned object so the same tag vocabulary produces node or
// relationship schema depending on the model's interface.
func directiveToObject(directive, label, prop string, field reflect.StructField, scope Scope) (Object, error) {
	name, arg, _ := strings.Cut(directive, ":")

	switch name {
	case "unique":
		return Object{Kind: KindConstraint, Constraint: Unique, Scope: scope, Label: label, Properties: []string{prop}}, nil
	case "key":
		return Object{Kind: KindConstraint, Constraint: NodeKey, Scope: scope, Label: label, Properties: []string{prop}}, nil
	case "exists":
		return Object{Kind: KindConstraint, Constraint: Exists, Scope: scope, Label: label, Properties: []string{prop}}, nil
	case "type":
		pt, err := neoType(field.Type)
		if err != nil {
			return Object{}, fmt.Errorf("schema: %s.%s: %w", label, prop, err)
		}
		return Object{Kind: KindConstraint, Constraint: PropType, Scope: scope, Label: label, Properties: []string{prop}, PropType: pt}, nil
	case "index":
		ik := RangeIndex
		switch arg {
		case "", "range":
			ik = RangeIndex
		case "text":
			ik = TextIndex
		case "point":
			ik = PointIndex
		default:
			return Object{}, fmt.Errorf("schema: %s.%s: unknown index type %q (want range|text|point; fulltext/vector go through SchemaObjects)", label, prop, arg)
		}
		return Object{Kind: KindIndex, Index: ik, Scope: scope, Label: label, Properties: []string{prop}}, nil
	default:
		return Object{}, fmt.Errorf("schema: %s.%s: unknown neo directive %q", label, prop, directive)
	}
}

// jsonProp extracts the Neo4j property name from a field's json tag (the first
// comma-segment, so `json:"id,omitempty"` yields "id"). A missing or "-" tag
// means the field is not persisted, so it cannot carry a constraint.
func jsonProp(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return "", false
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		return "", false
	}
	return name, true
}

// neoType derives the Neo4j property type from a Go field type for the `type`
// (property-type) constraint. Pointers are followed; slices become LIST<...>.
func neoType(t reflect.Type) (string, error) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return "STRING", nil
	case reflect.Bool:
		return "BOOLEAN", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "INTEGER", nil
	case reflect.Float32, reflect.Float64:
		return "FLOAT", nil
	case reflect.Slice:
		elem, err := neoType(t.Elem())
		if err != nil {
			return "", err
		}
		return "LIST<" + elem + " NOT NULL>", nil
	default:
		return "", fmt.Errorf("cannot derive a Neo4j type for the `type` constraint from Go kind %s", t.Kind())
	}
}
