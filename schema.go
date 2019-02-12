package graphql

import (
	"errors"
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/jinzhu/inflection"
)

//ResolverFactory denotes a functin that generates a graphql resolver function
//this is used to generate listing functions given a dynamic filter list: on a map[field name] = filter type
type ResolverFactory func(entity Entity) graphql.FieldResolveFn

//Resolvers provides a means to group all resolvers
type Resolvers struct {
	Single  ResolverFactory
	Listing ResolverFactory
}

//SchemaOpts provides a means to define options for graphql schema definition
type SchemaOpts struct {
	DefaultResolvers Resolvers
}

//emptyEntityResolver is a placeholder resolver in case passed resolver is nil
func emptyEntityResolver(entity Entity) graphql.FieldResolveFn {
	var err = errors.New(`unresolvable: ` + entity.Description())
	return func(p graphql.ResolveParams) (interface{}, error) {
		return nil, err
	}
}

//Schema creates a graphql schema definition including Query & Mutation with types and resolvers for a set of entities
func Schema(opts SchemaOpts, entities ...Entity) (*graphql.Schema, error) {
	var (
		query           = make(graphql.Fields)
		entitiesObjects = make(map[string]*graphql.Object)
		entitiesMap     = make(map[string]Entity)
		entitiesNames   = make([]string, len(entities))
		// filters         map[string]int8 //todo use filters in listing
	)

	//first pass is to define all entity types without relationships
	for i := range entities {
		var (
			typeInfo *graphql.Object
			err      error
		)
		entity := entities[i]
		typeInfo, _, err = FieldType(entity) //todo typeInfo, filters, err : use filters in listing
		name := typeInfo.Name()

		if err != nil {
			return nil, err
		}
		entitiesObjects[name] = typeInfo
		entitiesMap[name] = entity
		entitiesNames[i] = name
	}

	//define relationship types
	for i := range entities {
		entity := entities[i]
		name := entitiesNames[i]

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
		entity := entities[i]
		name := entitiesNames[i]
		description := entity.Description()
		plural := inflection.Plural(name)
		typeInfo := entitiesObjects[name]
		resolvers := getResolvers(&opts, entity)

		query[name] = &graphql.Field{
			Name:        name,
			Type:        typeInfo,
			Description: fmt.Sprintf("Get a single %s (%s) by id or slug", name, description),
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
			Description: fmt.Sprintf("Get a list of %s (%s) according to filters", name, description),
			Resolve:     resolvers.Listing(entity),
			// todo Args: use filters in listing
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

func getResolvers(opts *SchemaOpts, entity Entity) Resolvers {
	var singleResolver, listingResolver ResolverFactory
	resolvers := entity.Resolvers()

	if resolvers.Single != nil {
		singleResolver = resolvers.Single
	} else if opts.DefaultResolvers.Single != nil {
		singleResolver = opts.DefaultResolvers.Single
	} else {
		singleResolver = emptyEntityResolver
	}

	if resolvers.Listing != nil {
		listingResolver = resolvers.Listing
	} else if opts.DefaultResolvers.Listing != nil {
		listingResolver = opts.DefaultResolvers.Listing
	} else {
		listingResolver = emptyEntityResolver
	}

	return Resolvers{
		Single:  singleResolver,
		Listing: listingResolver,
	}
}
