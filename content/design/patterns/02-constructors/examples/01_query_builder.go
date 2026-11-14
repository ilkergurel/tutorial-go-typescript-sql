// LEARN(design.patterns.constructors): a builder earns its place when construction accumulates; its errors stick.
package main

import (
	"fmt"
	"strings"
)

// Query accumulates conditions and their arguments over several calls. Each
// method returns the receiver so calls chain, and the first error sticks.
type Query struct {
	where []string
	args  []any
	order string
	err   error
}

// Where takes a condition with one ? and its value. The builder numbers the
// placeholders, so no caller can get $3 wrong or splice a value into the SQL.
func (q *Query) Where(cond string, value any) *Query {
	q.args = append(q.args, value)
	q.where = append(q.where, strings.Replace(cond, "?", fmt.Sprintf("$%d", len(q.args)), 1))
	return q
}

// Column names can't be parameters: map public sort keys to fixed SQL.
var sortKeys = map[string]string{"title": "title", "newest": "created_at DESC"}

func (q *Query) OrderBy(key string) *Query {
	col, ok := sortKeys[key]
	if !ok && q.err == nil {
		q.err = fmt.Errorf("unknown sort key %q", key)
	}
	q.order = col
	return q
}

// Build is the one place the error surfaces, like bufio.Writer's Flush.
func (q *Query) Build() (string, []any, error) {
	if q.err != nil {
		return "", nil, q.err
	}
	sql := "SELECT id, title FROM lessons"
	if len(q.where) > 0 {
		sql += " WHERE " + strings.Join(q.where, " AND ")
	}
	if q.order != "" {
		sql += " ORDER BY " + q.order
	}
	return sql, q.args, nil
}

func main() {
	// A search form: any subset of the filters may be filled in.
	forms := []struct{ track, level, sort string }{
		{track: "go", sort: "newest"},
		{track: "sql", level: "advanced", sort: "title"},
		{level: "basic", sort: "title; DROP TABLE lessons"},
	}
	for _, f := range forms {
		q := &Query{}
		if f.track != "" {
			q.Where("track = ?", f.track)
		}
		if f.level != "" {
			q.Where("level = ?", f.level)
		}
		sql, args, err := q.OrderBy(f.sort).Build()
		if err != nil {
			fmt.Println("error:", err)
			continue
		}
		fmt.Printf("%s\n  args %q\n", sql, args)
	}
}
