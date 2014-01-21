package gorp

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestSimpleQuery(t *testing.T) {
	actual := Select("*").
		From("TableX").
		Where("ColumnA IS NULL").
		Sql()

	expected := `
SELECT *
FROM TableX
WHERE ColumnA IS NULL`
	expected = strings.TrimSpace(expected)
	assert.Equal(t, expected, actual)
}

func TestComplicatedQuery(t *testing.T) {
	actual := Select("DISTINCT ColumnA").
		From("TableA A").
		Join("TableB B ON B.Id = A.Id").
		LeftJoin("TableC C ON C.Id = A.Id").
		Where("ColumnA = 1 OR (ColumnB IS NULL AND ColumnC > 5)").
		Where("ColumnD = 'Xyz'").
		OrderBy("ColumnA DESC").
		GroupBy("ColumnF").
		Having("COUNT(ColumnF) > 10").
		Limit(4).
		Offset(4).
		Sql()

	expected := `
SELECT DISTINCT ColumnA
FROM TableA A
INNER JOIN TableB B ON B.Id = A.Id
LEFT JOIN TableC C ON C.Id = A.Id
WHERE (ColumnA = 1 OR (ColumnB IS NULL AND ColumnC > 5))
AND (ColumnD = 'Xyz')
ORDER BY ColumnA DESC
GROUP BY ColumnF
HAVING COUNT(ColumnF) > 10
LIMIT 4
OFFSET 4`
	expected = strings.TrimSpace(expected)
	assert.Equal(t, expected, actual)
}
