package graph

import (
	"fmt"
	"net/url"
	"strings"
)

type ProfileField int

const (
	ID ProfileField = iota
	FirstName
	LastName
	ProfilePic
	Locale
	Timezone
)

func (f ProfileField) String() string {
	return []string{"id", "first_name", "last_name", "profile_pic", "locale", "timezone"}[f]
}

type QueryBuilder struct {
	base   string
	node   string
	fields []string
	query  url.Values
}

func NewQuery(apiURL, node string) *QueryBuilder {
	return &QueryBuilder{
		base:  strings.TrimSuffix(apiURL, "/"),
		node:  node,
		query: url.Values{},
	}
}

func (q *QueryBuilder) WithFields(fields ...ProfileField) *QueryBuilder {
	for _, f := range fields {
		q.fields = append(q.fields, f.String())
	}
	return q
}

func (q *QueryBuilder) WithToken(token string) *QueryBuilder {
	q.query.Set("access_token", token)
	return q
}

func (q *QueryBuilder) Build() (string, error) {
	u, err := url.Parse(fmt.Sprintf("%s/%s", q.base, q.node))
	if err != nil {
		return "", err
	}
	if len(q.fields) > 0 {
		q.query.Set("fields", strings.Join(q.fields, ","))
	}
	u.RawQuery = q.query.Encode()
	return u.String(), nil
}
