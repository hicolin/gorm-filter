package filter

import (
	"reflect"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

const (
	Eq        = "="
	Like      = "like"
	Rlike     = "rlike"
	GT        = ">"
	LT        = "<"
	GTE       = ">="
	LTE       = "<="
	In        = "in"
	DateRange = "date_range"
)

// Rule represents a search rule for a field in a struct
type Rule struct {
	Name    string // 字段名
	Opt     string // 操作
	Table   string // 表名
	UseZero bool   // 是否使用零值
}

// Filter applies filter rules to the given dest struct
func Filter(dest any) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		rv := reflect.ValueOf(dest)

		if rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		if rv.Kind() != reflect.Struct {
			return db
		}

		destMap := make(map[string]reflect.Value)
		var rules []Rule
		for i := 0; i < rv.NumField(); i++ {
			filterTagStr := rv.Type().Field(i).Tag.Get("filter")
			filterTagStr = strings.Trim(filterTagStr, " ;,") // 去除首尾多余的逗号和分号
			if filterTagStr == "" || filterTagStr == "-" {   // 忽略没有filter标签的字段或filter:"-"的字段
				continue
			}

			var rule Rule
			rule.Name = strings.TrimSpace(removeOmitempty(rv.Type().Field(i).Tag.Get("json")))
			destMap[rule.Name] = rv.Field(i)

			filterTags := strings.Split(filterTagStr, ";")
			for _, filterTag := range filterTags {
				kv := strings.Split(filterTag, ":")
				k := strings.TrimSpace(kv[0])
				v := strings.TrimSpace(kv[1])

				switch k {
				case "opt":
					rule.Opt = v
				case "table":
					rule.Table = v
				case "use_zero", "useZero": // 兼容小驼峰和蛇形名称
					b, err := strconv.ParseBool(v)
					if err != nil {
						panic(err)
					}
					rule.UseZero = b
				}
			}
			rules = append(rules, rule)
		}

		if len(rules) == 0 {
			return db
		}

		var conditions []string
		var params []interface{}

		for _, rule := range rules {
			rfVal := destMap[rule.Name] // ensure the field exists

			// Skip zero values and empty slices if UseZero is false
			emptySlice := rfVal.Kind() == reflect.Slice && rfVal.Len() == 0 // 兼容空切片
			if (rfVal.IsZero() || emptySlice) && !rule.UseZero {
				continue
			}

			conditions, params = parseRule(rule, rfVal, conditions, params)
		}

		queryStr := strings.Join(conditions, " AND ")
		db.Where(queryStr, params...)

		return db
	}
}

// Search applies search rules to the given dest struct
func Search(rules []Rule, dest any) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		rv := reflect.ValueOf(dest)

		if rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		if rv.Kind() != reflect.Struct {
			return db
		}

		if len(rules) == 0 {
			return db
		}

		// create a map of dest struct fields to their values
		destMap := make(map[string]reflect.Value)
		for i := 0; i < rv.NumField(); i++ {
			jsonField := strings.TrimSpace(removeOmitempty(rv.Type().Field(i).Tag.Get("json")))
			if jsonField != "" {
				destMap[jsonField] = rv.Field(i)
			}
		}

		var conditions []string
		var params []interface{}

		for _, rule := range rules {
			rfVal, ok := destMap[rule.Name]
			if !ok {
				continue
			}

			// Skip zero values if UseZero is false
			if rfVal.IsZero() && !rule.UseZero {
				continue
			}

			conditions, params = parseRule(rule, rfVal, conditions, params)
		}

		queryStr := strings.Join(conditions, " AND ")
		db.Where(queryStr, params...)

		return db
	}
}

// MultiSearch applies search rules to the given dest string
func MultiSearch(rules []Rule, dest string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		dest = strings.TrimSpace(dest)
		if dest == "" {
			return db
		}
		if len(rules) == 0 {
			return db
		}

		rfVal := reflect.ValueOf(dest)
		var conditions []string
		var params []interface{}

		for _, rule := range rules {
			conditions, params = parseRule(rule, rfVal, conditions, params)
		}

		queryStr := strings.Join(conditions, " OR ")
		db.Where(queryStr, params...)

		return db
	}
}

// parseRule parses a search rule and returns a condition string and a slice of parameters
func parseRule(rule Rule, rfVal reflect.Value, conditions []string, params []interface{}) ([]string, []interface{}) {
	if rule.Table != "" {
		rule.Name = rule.Table + "." + rule.Name
	}
	if rule.Opt == "" {
		rule.Opt = Eq
	}

	value := rfVal.Interface()
	switch rule.Opt {
	case Eq:
		conditions = append(conditions, rule.Name+" = ?")
		params = append(params, value)
	case Like:
		conditions = append(conditions, rule.Name+" like ?")
		params = append(params, "%"+value.(string)+"%")
	case Rlike:
		conditions = append(conditions, rule.Name+" rlike ?")
		params = append(params, value)
	case GT:
		conditions = append(conditions, rule.Name+" > ?")
		params = append(params, value)
	case LT:
		conditions = append(conditions, rule.Name+" < ?")
		params = append(params, value)
	case GTE:
		conditions = append(conditions, rule.Name+" >= ?")
		params = append(params, value)
	case LTE:
		conditions = append(conditions, rule.Name+" <= ?")
		params = append(params, value)
	case In:
		conditions = append(conditions, rule.Name+" in (?)")
		params = append(params, value)
	case DateRange:
		dates := value.([]string)
		if len(dates) != 2 {
			panic("date_range rule requires two values")
		}
		sTime := dates[0] + " 00:00:00"
		eTime := dates[1] + " 23:59:59"
		conditions = append(conditions, rule.Name+" between ? and ?")
		params = append(params, sTime, eTime)
	}

	return conditions, params
}

func removeOmitempty(tag string) string {
	if idx := strings.Index(tag, ",omitempty"); idx != -1 {
		return tag[:idx]
	}
	// 兼容 go-zero
	if idx := strings.Index(tag, ",optional"); idx != -1 {
		return tag[:idx]
	}
	return tag
}
