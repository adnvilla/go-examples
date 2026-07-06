//nolint:staticcheck // aws-sdk-go v1 is deprecated in favor of v2; this is a legacy example
package main

import (
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

func main() {

}

func FilterGreaterThan(name string, value any) expression.ConditionBuilder {
	return expression.Name(name).GreaterThan(expression.Value(value))
}

func ProjectionNames(names ...string) expression.ProjectionBuilder {

	var nameBuilder []expression.NameBuilder
	for _, name := range names {
		nameBuilder = append(nameBuilder, expression.Name(name))
	}

	return expression.NamesList(nameBuilder[0], nameBuilder[1:]...)
}
