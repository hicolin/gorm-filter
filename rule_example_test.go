package filter_test

import (
	"gorm.io/gorm"
)

type MockUser struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type MockUserFilter struct {
	Name string `json:"name" filter:"opt:rlike"` // 没有 filter Tag 或 filter:"-", 则不会添加到过滤条件中
	Age  int    `json:"age" filter:"opt:="`
}

// Mock database
var db *gorm.DB

func ExampleWithFilter() {
	var users []MockUser
	user := MockUserFilter{
		Name: "John",
		Age:  20,
	}
	db.Scopes(filter.WithFilter(user)).Find(&users)
}

func ExampleWithSearch() {
	var users []MockUser
	user := MockUserFilter{
		Name: "John",
		Age:  20,
	}
	rule := []filter.Rule{{Name: "name", Opt: "rlike"}, {Name: "age", Opt: "="}}
	db.Scopes(filter.WithSearch(rule, user)).Find(&users)
}

func ExampleWithMultiSearch() {
	var users []MockUser
	keyword := "keyword"
	rule := []filter.Rule{{Name: "name", Opt: "rlike"}, {Name: "age", Opt: "="}}
	db.Scopes(filter.WithMultiSearch(rule, keyword)).Find(&users)
}
