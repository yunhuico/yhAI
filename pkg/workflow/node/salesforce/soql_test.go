package salesforce

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewQuery(t *testing.T) {
	like, err := WhereLike("Name", "Test-01")
	assert.NoError(t, err)
	assert.NotNil(t, like)

	q, err := newQuery(queryOpt{
		Fields: []string{
			"Id",
			"Name",
		},
		SObject: "account",
		Limit:   10,

		Like: like,
	})
	assert.NoError(t, err)
	assert.NotNil(t, q)
	sql, err := q.Format()
	assert.NoError(t, err)
	assert.NotEmpty(t, sql)
	assert.Equal(t, sql, "SELECT Id,Name FROM account WHERE Name LIKE '%Test-01%' Limit 10")

	qNoLike, err := newQuery(queryOpt{
		Fields: []string{
			"Id",
			"Name",
		},
		SObject: "account",
		Limit:   10,
	})

	assert.NoError(t, err)
	assert.NotNil(t, qNoLike)
	sql, err = qNoLike.Format()
	assert.NoError(t, err)
	assert.NotEmpty(t, sql)
	assert.Equal(t, sql, "SELECT Id,Name FROM account Limit 10")

	q1, err := newQuery(queryOpt{})
	assert.Error(t, err)
	assert.Nil(t, q1)
}
