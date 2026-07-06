package main

import (
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
)

func main() {

}

// FilterGreaterThan builds a filter condition for "name > value".
func FilterGreaterThan(name string, value any) expression.ConditionBuilder {
	return expression.Name(name).GreaterThan(expression.Value(value))
}

// ProjectionNames builds a projection listing the attributes a Scan should return.
func ProjectionNames(names ...string) expression.ProjectionBuilder {
	var nameBuilder []expression.NameBuilder
	for _, name := range names {
		nameBuilder = append(nameBuilder, expression.Name(name))
	}

	return expression.NamesList(nameBuilder[0], nameBuilder[1:]...)
}
