package graphql

import (
	"fmt"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/jinzhu/inflection"
)

//Resolvers provides a means to group all resolvers
type Resolvers struct {
	Single  ResolverFactory
	Listing ResolverFactory
}

//SchemaOpts provides a means to define options for graphql schema definition
type SchemaOpts struct {
	DefaultResolvers Resolvers
}

//Schema creates a graphql schema definition including Query & Mutation with types and resolvers for a set of entities
func Schema(opts SchemaOpts, entities ...Entity) (*graphql.Schema, error) {
	var (
		query           = make(graphql.Fields)
		entitiesObjects = make(map[string]*graphql.Object)
		entitiesMap     = make(map[string]*Entity)
	)

	//first pass is to define all entity types without relationships
	for i := range entities {
		entity := &entities[i]
		name := strings.ToLower(entity.Name)
		typeInfo, err := FieldType(entity)

		if err != nil {
			return nil, err
		}
		entitiesObjects[name] = typeInfo
		entitiesMap[name] = entity
	}

	//define relationship types
	for i := range entities {
		entity := &entities[i]
		name := strings.ToLower(entity.Name)

		fields, err := RelationshipType(entitiesMap, entitiesObjects, entity, getResolvers(&opts, entity))

		if err != nil {
			return nil, err
		} else if fields == nil {
			continue
		}

		for f, field := range fields {
			entitiesObjects[name].AddFieldConfig(f, field)
		}
	}

	//finally we define the query itself
	for i := range entities {
		entity := &entities[i]
		name := strings.ToLower(entity.Name)
		plural := inflection.Plural(name)
		typeInfo := entitiesObjects[name]
		resolvers := getResolvers(&opts, entity)

		query[name] = &graphql.Field{
			Name:        name,
			Type:        typeInfo,
			Description: fmt.Sprintf("Get a single %s (%s) by id or slug", entity.Name, entity.Description),
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{
					Type: graphql.String,
				},
			},
			Resolve: resolvers.Single(entity),
		}

		query[plural] = &graphql.Field{
			Name:        plural,
			Type:        graphql.NewList(typeInfo),
			Description: fmt.Sprintf("Get a list of %s (%s) according to filters", entity.Name, entity.Description),
			Resolve:     resolvers.Listing(entity),
		}
	}

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name:   "Query",
			Fields: query,
		}),
	})

	return &schema, err
}

func getResolvers(opts *SchemaOpts, entity *Entity) Resolvers {
	var singleResolver, listingResolver ResolverFactory

	if entity.Resolvers.Single == nil {
		singleResolver = opts.DefaultResolvers.Single
	} else {
		singleResolver = entity.Resolvers.Single
	}

	if entity.Resolvers.Listing == nil {
		listingResolver = opts.DefaultResolvers.Listing
	} else {
		listingResolver = entity.Resolvers.Listing
	}

	return Resolvers{
		Single:  singleResolver,
		Listing: listingResolver,
	}
}
