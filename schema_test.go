package graphql_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	gql "github.com/gocipe/graphql"
	"github.com/graphql-go/graphql"
	"github.com/jinzhu/inflection"
	"github.com/stretchr/testify/assert"
)

const introspectionQuery = `
query {
    __schema {
        types {
            name
            description
            fields {
              name
			  description
			  type {
				kind
				name
				ofType {
				  name
				  kind
				}
			  }
            } 
        } 
    }
}
`

type entity struct {
	description string
	instance    interface{}
	resolvers   gql.Resolvers
}

func (e entity) Description() string {
	return e.description
}
func (e entity) Instance() interface{} {
	return e.instance
}

func (e entity) Resolvers() gql.Resolvers {
	return e.resolvers
}

//article represents an article on the website
type article struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Author    author    `json:"author"`
	Tags      []tag     `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

//author represents an Author on the website
type author struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

//tag represents a means of categorization of articles
type tag struct {
	ID       string
	Name     string
	Articles []article
}

var (
	entityArticle = entity{description: "An article on the website", instance: article{}}
	entityAuthor  = entity{description: "A human person who writes things", instance: author{}}
	entityTag     = entity{description: "Tags are used to categorize articles", instance: tag{}}
)

type schemaFieldDef struct {
	Name string `json:"name"`
	Type struct {
		Kind   string `json:"kind"`
		Name   string `json:"name"`
		OfType struct {
			Kind string `json:"kind"`
			Name string `json:"name"`
		} `json:"ofType"`
	} `json:"type"`
}

type schemaTypeDef struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Fields      []schemaFieldDef `json:"fields"`
}

type schemaIntrospection struct {
	Data struct {
		Schema struct {
			Types []schemaTypeDef `json:"types"`
		} `json:"__schema"`
	} `json:"data"`
	Errors []struct {
		Message   string `json:"message"`
		Locations []struct {
			Column int         `json:"column"`
			Line   int         `json:"line"`
			Path   interface{} `json:"path"`
		} `json:"locations"`
	} `json:"errors"`
}

func TestSchema(t *testing.T) {
	type fieldExpected map[string]struct {
		Kind string
		Name string
	}

	entities := map[string]struct {
		Name        string
		Description string
		Fields      fieldExpected
	}{
		"article": {
			Description: entityArticle.description,
			Fields: fieldExpected{
				"id":         {Name: "id", Kind: "String"},
				"title":      {Name: "title", Kind: "String"},
				"body":       {Name: "body", Kind: "String"},
				"author":     {Name: "author", Kind: "OBJECT.author"},
				"tags":       {Name: "tags", Kind: "LIST.OBJECT.tag"},
				"created_at": {Name: "created_at", Kind: "DateTime"},
				"updated_at": {Name: "updated_at", Kind: "DateTime"},
			},
		},
		"author": {
			Description: entityAuthor.description,
			Fields: fieldExpected{
				"id":    {Name: "id", Kind: "String"},
				"name":  {Name: "name", Kind: "String"},
				"email": {Name: "email", Kind: "String"},
			},
		},
		"tag": {
			Description: entityTag.description,
			Fields: fieldExpected{
				"id":       {Name: "id", Kind: "String"},
				"name":     {Name: "name", Kind: "String"},
				"articles": {Name: "articles", Kind: "LIST.OBJECT.article"},
			},
		},
	}

	opts := gql.SchemaOpts{}
	schema, err := gql.Schema(opts, entityArticle, entityAuthor, entityTag)
	assert.NoError(t, err, "Error creating schema definition")

	params := graphql.Params{Schema: *schema, RequestString: introspectionQuery}
	r := graphql.Do(params)
	body, err := json.Marshal(r)
	if !assert.NoError(t, err, "Marshal graphql response") {
		return
	}

	var schemaI schemaIntrospection
	err = json.Unmarshal(body, &schemaI)
	if !assert.NoError(t, err, "Error unmarshalling graphql response") {
		return
	}

	var graphErrors string
	if len(schemaI.Errors) > 0 {
		for i, err := range schemaI.Errors {
			graphErrors += "\t" + strconv.Itoa(i+1) + "." + err.Message
			for _, loc := range err.Locations {
				graphErrors += fmt.Sprintf(" [%d,%d]", loc.Column, loc.Line)
			}

			graphErrors += "\n"
		}
	}

	assert.Empty(t, schemaI.Errors, "Errors received as response to graphql query:\n", graphErrors)

	for _, schemaType := range schemaI.Data.Schema.Types {
		if e, ok := entities[schemaType.Name]; ok {
			assert.Equal(t, e.Description, schemaType.Description, "Invalid Description for %s", schemaType.Name)
			assert.Equal(t, len(e.Fields), len(schemaType.Fields), "Number of defined fields for %s not equal", schemaType.Name)

			fields := make(map[string]struct {
				Kind string
				Name string
			})

			for i, f := range e.Fields {
				fields[f.Name] = e.Fields[i]
			}

			for _, f := range schemaType.Fields {
				var kind, name string
				name = f.Name

				switch f.Type.Kind {
				default:
					kind = f.Type.Name
				case "OBJECT":
					kind = "OBJECT." + f.Type.Name
				case "LIST":
					kind = "LIST." + f.Type.OfType.Kind + "." + f.Type.OfType.Name
					name = inflection.Plural(name)
				}

				if !assert.Contains(t, fields, name, "Unknown field definition for %s (%s)", schemaType.Name, f.Name) {
					return
				}

				if !assert.Equal(t, fields[name].Kind, kind, "Field kind invalid for %s", fields[f.Name].Kind) {
					return
				}

			}

			delete(entities, schemaType.Name)
		}
	}

	assert.Empty(t, entities, "Some entities were improperly defined")

}
