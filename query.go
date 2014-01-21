package gorp

import (
	"fmt"
)

type join struct {
	joinType string
	tables   string
}

type Query struct {
	selectStr  string
	fromStr    string
	joins      []join
	wheres     []string
	orderByStr string
	groupByStr string
	havingStr  string
	limitInt   int
	offsetInt  int
}

func Select(columns string) *Query {
	return &Query{selectStr: columns}
}

func (q *Query) From(tables string) *Query {
	q.fromStr = tables
	return q
}

func (q *Query) Join(tables string) *Query {
	q.joins = append(q.joins, join{"INNER", tables})
	return q
}

func (q *Query) LeftJoin(tables string) *Query {
	q.joins = append(q.joins, join{"LEFT", tables})
	return q
}

func (q *Query) Where(query string) *Query {
	q.wheres = append(q.wheres, query)
	return q
}

func (q *Query) OrderBy(orderBy string) *Query {
	q.orderByStr = orderBy
	return q
}

func (q *Query) GroupBy(keys string) *Query {
	q.groupByStr = keys
	return q
}

func (q *Query) Having(conditions string) *Query {
	q.havingStr = conditions
	return q
}

func (q *Query) Limit(limit int) *Query {
	q.limitInt = limit
	return q
}

func (q *Query) Offset(offset int) *Query {
	q.offsetInt = offset
	return q
}

func (q *Query) Sql() string {
	// Select
	sql := fmt.Sprintf("SELECT %v", q.selectStr)

	// From
	sql = fmt.Sprintf("%v\nFROM %v", sql, q.fromStr)

	// Join
	for _, join := range q.joins {
		sql = fmt.Sprintf("%v\n%v JOIN %v", sql, join.joinType, join.tables)
	}

	// Where
	if len(q.wheres) == 1 {
		sql = fmt.Sprintf("%v\nWHERE %v", sql, q.wheres[0])
	}
	if len(q.wheres) > 1 {
		for i, where := range q.wheres {
			if i == 0 {
				sql = fmt.Sprintf("%v\nWHERE (%v)", sql, where)
			} else {
				sql = fmt.Sprintf("%v\nAND (%v)", sql, where)
			}
		}
	}

	// OrderBy
	if q.orderByStr != "" {
		sql = fmt.Sprintf("%v\nORDER BY %v", sql, q.orderByStr)
	}

	// GroupBy
	if q.groupByStr != "" {
		sql = fmt.Sprintf("%v\nGROUP BY %v", sql, q.groupByStr)
	}

	// Having
	if q.havingStr != "" {
		sql = fmt.Sprintf("%v\nHAVING %v", sql, q.havingStr)
	}

	// Limit
	if q.limitInt != 0 {
		sql = fmt.Sprintf("%v\nLIMIT %v", sql, q.limitInt)
	}

	// Offset
	if q.offsetInt != 0 {
		sql = fmt.Sprintf("%v\nOFFSET %v", sql, q.offsetInt)
	}

	return sql
}
