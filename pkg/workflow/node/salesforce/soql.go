package salesforce

import (
	"errors"
	"fmt"
	"strings"
)

type sobjectQuery struct {
	Fields  []string
	SObject string
	Limit   int
	Offset  int
	Like    *likeSql
}

type queryOpt struct {
	Fields  []string
	SObject string
	Limit   int
	Offset  int
	Like    *likeSql
}

func newQuery(opt queryOpt) (*sobjectQuery, error) {
	if opt.SObject == "" {
		return nil, fmt.Errorf("sobject can not be empty")
	}
	if len(opt.Fields) == 0 {
		return nil, fmt.Errorf("field list can not be empty")
	}
	return &sobjectQuery{
		Fields:  opt.Fields,
		SObject: opt.SObject,
		Limit:   opt.Limit,
		Offset:  opt.Offset,
		Like:    opt.Like,
	}, nil
}

func (q *sobjectQuery) Format() (string, error) {
	if q.SObject == "" {
		return "", fmt.Errorf("sobject can not be empty")
	}
	if len(q.Fields) == 0 {
		return "", fmt.Errorf("field list can not be empty")
	}

	soql := "SELECT " + strings.Join(q.Fields, ",")
	soql += " FROM " + q.SObject

	if q.Like != nil {
		soql += " WHERE " + q.Like.expression
	}

	if q.Limit > 0 {
		soql += fmt.Sprintf(" Limit %d", q.Limit)
	}
	if q.Offset > 0 {
		soql += fmt.Sprintf(" OFFSET %d", q.Offset)
	}
	return soql, nil
}

type likeSql struct {
	expression string
}

// WhereLike will form the LIKE expression.
func WhereLike(field string, value string) (*likeSql, error) {
	if field == "" {
		return nil, errors.New("soql where: field can not be empty")
	}
	if value == "" {
		return nil, errors.New("soql where: value can not be empty")
	}
	value = "%" + value + "%"
	return &likeSql{
		expression: fmt.Sprintf("%s LIKE '%s'", field, value),
	}, nil
}
