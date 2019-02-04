package graphql

import "github.com/graphql-go/graphql"

//ResolverFactory denotes a functin that generates a graphql resolver function
//this is used to generate listing functions given a dynamic filter list: on a map[field name] = filter type
type ResolverFactory func(entity *Entity) graphql.FieldResolveFn

//singleEntityResolver creates a graphql resolver for a Single *Entity based on id and/or slug
func singleEntityResolver(entity *Entity) graphql.FieldResolveFn {
	panic("not implemented")
	return nil
}

//listingEntityResolver creates a graphql resolver for multiple entities given a map of possible filters
func listingEntityResolver(entity *Entity) graphql.FieldResolveFn {
	panic("not implemented")
	return nil
}
