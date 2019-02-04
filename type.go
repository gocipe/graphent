package graphql

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/jinzhu/inflection"
)

var (
	errorNotSimpleFieldType  = errors.New("not a simple field type")
	errorNotRelationshipType = errors.New("not a relationship field type")
	errorUnknownFieldType    = errors.New("unknown type")
)

type fieldDefinition struct {
	filter int8
	field  *graphql.Field
}

//FieldType creates a graphql type given an entity definition
func FieldType(e *Entity) (*graphql.Object, error) {
	var fields graphql.Fields
	t := reflect.TypeOf(e.Instance)
	var numfields = t.NumField()
	fields = make(graphql.Fields, numfields)
	e.filters = make(map[string]int8)

	for i := 0; i < numfields; i++ {
		if def, err := field(t.Field(i)); err == errorNotSimpleFieldType {
			continue //not a simple type
		} else if err != nil {
			return nil, err
		} else if def.field != nil && def.field.Name != "" {
			fields[def.field.Name] = def.field
			if def.filter != filterNone {
				e.filters[def.field.Name] = def.filter
			}
		}
	}

	return graphql.NewObject(
		graphql.ObjectConfig{
			Name:        e.Label,
			Description: e.Description,
			Fields:      fields,
		},
	), nil
}

//field creates a field definition (used by type) object given a struct field
func field(f reflect.StructField) (fieldDefinition, error) {
	var (
		t          graphql.Output
		name       string
		filterable bool
		ftype      fieldDefinition
		kind       = f.Type.Kind()
		fulltype   = f.Type.PkgPath() + "." + f.Type.Name()
	)

	if v, ok := f.Tag.Lookup("filterable"); ok {
		filterable, _ = strconv.ParseBool(v)
	}

	if kind == reflect.String {
		t = graphql.String
		if filterable {
			ftype.filter = filterString
		}
	} else if kind == reflect.Bool {
		t = graphql.Boolean
		if filterable {
			ftype.filter = filterBool
		}
	} else if kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64 || kind == reflect.Uint || kind == reflect.Uint8 || kind == reflect.Uint16 || kind == reflect.Uint32 || kind == reflect.Uint64 {
		t = graphql.Int
		if filterable {
			ftype.filter = filterInt
		}
	} else if kind == reflect.Float32 || kind == reflect.Float64 {
		t = graphql.Float
		if filterable {
			ftype.filter = filterFloat
		}
	} else if fulltype == "time.Time" {
		t = graphql.DateTime
		if filterable {
			ftype.filter = filterDate
		}
	} else if kind == reflect.Struct || kind == reflect.Slice {
		return ftype, errorNotSimpleFieldType
	} else {
		return ftype, errorNotRelationshipType
	}

	if v, ok := f.Tag.Lookup("json"); ok {
		name = v
	} else {
		name = strings.ToLower(f.Name)
	}

	ftype.field = &graphql.Field{
		Type: t,
		Name: name,
	}

	return ftype, nil
}

//RelationshipType creates a graphql type given an entity definition
func RelationshipType(entitiesMap map[string]*Entity, entitiesObjects map[string]*graphql.Object, e *Entity, resolvers Resolvers) (graphql.Fields, error) {
	t := reflect.TypeOf(e.Instance)
	var numfields = t.NumField()
	var fields = make(graphql.Fields, numfields)
	e.filters = make(map[string]int8)

	for i := 0; i < numfields; i++ {
		if f, err := relationship(entitiesMap, entitiesObjects, t.Field(i), resolvers); err == errorNotRelationshipType {
			continue //not relationship material
		} else if err != nil {
			return fields, err
		} else {
			fields[f.Name] = f
		}
	}

	return fields, nil
}

func relationship(entitiesMap map[string]*Entity, entitiesObjects map[string]*graphql.Object, f reflect.StructField, resolvers Resolvers) (*graphql.Field, error) {
	var (
		typeInfo    graphql.Output
		resolver    graphql.FieldResolveFn
		description string
	)

	kind := f.Type.Kind()
	fulltype := f.Type.PkgPath() + "." + f.Type.Name()
	entityName := strings.ToLower(f.Name)
	name := entityName

	if kind == reflect.Slice {
		entityName = inflection.Singular(entityName)
	} else if fulltype == "time.Time" || (kind != reflect.Struct && kind != reflect.Slice) {
		return nil, errorNotRelationshipType
	} else if _, ok := entitiesMap[entityName]; !ok {
		return nil, errorUnknownFieldType
	} else if _, ok := entitiesObjects[entityName]; !ok {
		return nil, errorUnknownFieldType
	}

	entity := entitiesMap[entityName]

	if kind == reflect.Struct {
		typeInfo = entitiesObjects[entityName]
		description = fmt.Sprintf("Get a single %s (%s) by id or slug", entity.Name, entity.Description)
		resolver = resolvers.Single(entity)
	} else if kind == reflect.Slice {
		typeInfo = graphql.NewList(entitiesObjects[entityName])
		description = fmt.Sprintf("Get a list of %s (%s) according to filters", entity.Name, entity.Description)
		resolver = resolvers.Listing(entity)
	} else {
		return nil, errorNotRelationshipType
	}

	return &graphql.Field{
		Name:        name,
		Type:        typeInfo,
		Description: description,
		Resolve:     resolver,
	}, nil
}
