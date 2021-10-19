package structs

import (
	"database/sql"
	"fmt"
	"github.com/knadh/listmonk/utl/secure"
	"github.com/mitchellh/mapstructure"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func MergeSqlInsert(req interface{}, col *strings.Builder, bindVal *strings.Builder, binds *[]interface{}) {
	bind := []interface{}{}
	s := reflect.ValueOf(req)
	if s.Kind() != reflect.Ptr {
		return
	}
	count := 1
	for i := 0; i < s.Elem().NumField(); i++ {
		v := s.Elem().Field(i)
		skip := s.Elem().Type().Field(i).Tag.Get("json")
		if skip == "-" {
			continue
		}
		if v.Kind() > reflect.Float64 &&
			v.Kind() != reflect.String &&
			v.Kind() != reflect.Ptr {
			continue
		}
		if CheckNil(v) {
			continue
		}

		if len(col.String()) != 0 {
			col.WriteString(", ")
			bindVal.WriteString(", ")
		}
		col.WriteString(skip)
		bindVal.WriteString("$")
		bindVal.WriteString(strconv.Itoa(count))
		bind = append(bind, ConvertValue(v))
		count++
	}
	*binds = append(*binds, bind...)
}

func DifSqlSet(src, req interface{}, setUpdate *strings.Builder, binds *[]interface{}) {
	bind := []interface{}{}
	s := reflect.ValueOf(src)
	r := reflect.ValueOf(req)
	if s.Kind() != reflect.Ptr || r.Kind() != reflect.Ptr {
		return
	}
	for i := 0; i < s.Elem().NumField(); i++ {
		v := s.Elem().Field(i)
		fieldName := s.Elem().Type().Field(i).Name
		vr := r.Elem().FieldByName(fieldName)
		skip := s.Elem().Type().Field(i).Tag.Get("json")
		if skip == "-" {
			continue
		}
		if !vr.IsValid() {
			continue
		}
		if CheckNil(vr) {
			continue
		}
		if v.Kind() > reflect.Float64 &&
			v.Kind() != reflect.String &&
			v.Kind() != reflect.Struct &&
			v.Kind() != reflect.Ptr &&
			v.Kind() != reflect.Slice {
			continue
		}
		var flagDif bool
		if vr.Kind() == reflect.Ptr {
			if v.Interface() != ConvertValue(vr) {
				flagDif = true
			}
		} else {
			if v.Interface() != vr.Interface() {
				flagDif = true
			}
		}
		if flagDif {
			if skip == "password" {
				raws, _ := secure.Decode([]byte(v.Interface().(string)))
				ok, _ := raws.Verify([]byte(vr.Interface().(string)))
				if ok {
					continue
				}
			}
			if len(setUpdate.String()) == 0 {
				setUpdate.WriteString(" SET ")
			} else {
				setUpdate.WriteString(", ")
			}
			setUpdate.WriteString(skip)
			setUpdate.WriteString(" = ? ")
			if skip == "password" {
				secArgon2 := secure.DefaultConfig()
				raw, _ := secArgon2.Hash([]byte(vr.Interface().(string)), nil)
				bind = append(bind, string(raw.Encode()))
			} else {
				bind = append(bind, ConvertValue(vr))
			}
		}
	}
	*binds = append(*binds, bind...)
}

func ConvertValue(val reflect.Value) interface{} {
	v := val.Interface()
	switch v.(type) {
	case *int:
		return *v.(*int)
	case *int32:
		return *v.(*int32)
	case *int64:
		return *v.(*int64)
	case *float32:
		return *v.(*float32)
	case *float64:
		return *v.(*float64)
	case *string:
		return *v.(*string)
	case int:
		return v.(int)
	case int32:
		return v.(int32)
	case int64:
		return v.(int64)
	case float32:
		return v.(float32)
	case float64:
		return v.(float64)
	default:
		return v.(string)
	}

}

func CheckNil(val reflect.Value) bool {
	v := val.Interface()
	switch v.(type) {
	case *int:
		if v.(*int) != nil {
			return false
		} else {
			return true
		}
	case *int32:
		if v.(*int32) != nil {
			return false
		} else {
			return true
		}
	case *int64:
		if v.(*int64) != nil {
			return false
		} else {
			return true
		}
	case *float32:
		if v.(*float32) != nil {
			return false
		} else {
			return true
		}
	case *float64:
		if v.(*float64) != nil {
			return false
		} else {
			return true
		}
	case *string:
		if v.(*string) != nil {
			return false
		} else {
			return true
		}
	case int:
		if v.(int) > 0 {
			return false
		} else {
			return true
		}
	case int32:
		if v.(int32) > 0 {
			return false
		} else {
			return true
		}
	case int64:
		if v.(int64) > 0 {
			return false
		} else {
			return true
		}
	case float32:
		if v.(float32) > 0 {
			return false
		} else {
			return true
		}
	case float64:
		if v.(float64) != 0 {
			return false
		} else {
			return true
		}
	default:
		if len(v.(string)) > 0 {
			return false
		} else {
			return true
		}
	}
}

// Merge receives two structs, and merges them excluding fields with tag name: `structs`, value "-"
func Merge(dst, src interface{}) {
	s := reflect.ValueOf(src)
	d := reflect.ValueOf(dst)
	if s.Kind() != reflect.Ptr || d.Kind() != reflect.Ptr {
		return
	}
	for i := 0; i < s.Elem().NumField(); i++ {
		v := s.Elem().Field(i)
		fieldName := s.Elem().Type().Field(i).Name
		skip := s.Elem().Type().Field(i).Tag.Get("json")
		if skip == "-" {
			continue
		}
		if v.Kind() > reflect.Float64 &&
			v.Kind() != reflect.String &&
			v.Kind() != reflect.Struct &&
			v.Kind() != reflect.Ptr &&
			v.Kind() != reflect.Slice {
			continue
		}
		vr := d.Elem().FieldByName(fieldName)
		if !vr.IsValid() {
			continue
		}
		if v.Kind() == reflect.Ptr {
			// Field is pointer check if it's nil or set
			if !v.IsNil() {
				// Field is set assign it to dest

				if d.Elem().FieldByName(fieldName).Kind() == reflect.Ptr {
					d.Elem().FieldByName(fieldName).Set(v)
					continue
				}
				f := d.Elem().FieldByName(fieldName)
				if f.IsValid() {
					f.Set(v.Elem())
				}
			}
			continue
		}
		d.Elem().FieldByName(fieldName).Set(v)
	}
}

func MergeRow(rows *sql.Rows, dst interface{}) {
	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		fmt.Println(err.Error())
	}
	// Get column type
	columns_type, err := rows.ColumnTypes()
	if err != nil {
		fmt.Println(err.Error())
	}

	// Make a slice for the values
	values := make([]sql.RawBytes, len(columns))

	// rows.Scan wants '[]interface{}' as an argument, so we must copy the
	// references into such a slice
	// See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// get RawBytes from data
	err = rows.Scan(scanArgs...)
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	// Now do something with the data.
	// Here we just print each column as a string.
	m := make(map[string]interface{})
	//var value string
	for i, col := range values {
		// Here we can check if the value is nil (NULL value)
		if col != nil {
			if strings.ToLower(columns_type[i].DatabaseTypeName()) == "varchar" {
				m[columns[i]] = string(col)
			} else if strings.ToLower(columns_type[i].DatabaseTypeName()) == "decimal" {
				s, _ := strconv.ParseFloat(string(col), 64)
				m[columns[i]] = s
			} else if strings.ToLower(columns_type[i].DatabaseTypeName()) == "int" {
				m[columns[i]], err = strconv.Atoi(string(col))
			} else if strings.ToLower(columns_type[i].DatabaseTypeName()) == "timestamp" ||
				strings.ToLower(columns_type[i].DatabaseTypeName()) == "datetime" {
				if len(string(col)) > 0 {
					temp, _ := time.Parse("2006-01-02T15:04:05Z", string(col))
					m[columns[i]] = temp.Format("2006-01-02 15:04:05")
				}
			} else if strings.ToLower(columns_type[i].DatabaseTypeName()) == "text" {
				m[columns[i]] = string(col)
			} else {
				m[columns[i]], err = strconv.Atoi(string(col))
				if err != nil {
					m[columns[i]] = string(col)
				}
			}
		}
	}

	config := &mapstructure.DecoderConfig{
		TagName: "json",
		Result:  &dst,
	}
	decoder, _ := mapstructure.NewDecoder(config)
	err = decoder.Decode(m)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func MergeStructToMap(i interface{}) map[string]interface{} {
	values := map[string]interface{}{}
	iVal := reflect.ValueOf(i).Elem()
	typ := iVal.Type()
	for i := 0; i < iVal.NumField(); i++ {
		f := iVal.Field(i)
		// You ca use tags here...
		// tag := typ.Field(i).Tag.Get("tagname")
		// Convert each type into a string for the url.Values string map
		tv := typ.Field(i).Tag.Get("json")
		tagParts := strings.Split(tv, ",")
		keyName := typ.Field(i).Name
		if tagParts[0] != "" {
			if tagParts[0] == "-" {
				continue
			}
			keyName = tagParts[0]
		}
		switch f.Interface().(type) {
		case int, int8, int16, int32, int64:
			values[keyName] = strconv.FormatInt(f.Int(), 10)
		case uint, uint8, uint16, uint32, uint64:
			values[keyName] = strconv.FormatUint(f.Uint(), 10)
		case float32:
			values[keyName] = strconv.FormatFloat(f.Float(), 'f', 4, 32)
		case float64:
			values[keyName] = strconv.FormatFloat(f.Float(), 'f', 4, 64)
		case []byte:
			values[keyName] = string(f.Bytes())
		case string:
			values[keyName] = f.String()
		case []string:
			values[keyName] = strings.Join(f.Interface().([]string), ",")
		}
	}
	return values
}

func MergeRedis(src, dst interface{}) {
	s := reflect.ValueOf(src)
	d := reflect.ValueOf(dst)
	if d.Kind() != reflect.Ptr && s.Kind() != reflect.Map {
		return
	}
	myMap := src.(map[string]string)
	for i := 0; i < d.Elem().NumField(); i++ {
		v := d.Elem().Field(i)
		fieldName := d.Elem().Type().Field(i).Name
		tv := d.Elem().Type().Field(i).Tag.Get("json")
		tagParts := strings.Split(tv, ",")
		keyName := fieldName
		if tagParts[0] != "" {
			if tagParts[0] == "-" {
				continue
			}
			keyName = tagParts[0]
		}
		if fieldName == "-" {
			continue
		}
		if v.Kind() > reflect.Float64 &&
			v.Kind() != reflect.String &&
			v.Kind() != reflect.Struct &&
			v.Kind() != reflect.Ptr &&
			v.Kind() != reflect.Slice {
			continue
		}

		vType := v.Interface()
		switch vType.(type) {
		case int, int8, int16, int32, int64:
			val, _ := strconv.Atoi(myMap[keyName])
			d.Elem().FieldByName(fieldName).SetInt(int64(val))
		case float32, float64:
			val, _ := strconv.ParseFloat(myMap[keyName], 64)
			d.Elem().FieldByName(fieldName).SetFloat(val)
		case []string:
			slc := strings.Split(myMap[keyName], ",")
			rS := reflect.ValueOf(&slc)
			d.Elem().FieldByName(fieldName).Set(rS.Elem())
		default:
			d.Elem().FieldByName(fieldName).SetString(myMap[keyName])
		}
	}
}

func SqlINIntSeq(ns []int) string {
	if len(ns) == 0 {
		return ""
	}

	estimate := len(ns) * 4
	b := make([]byte, 0, estimate)
	for _, n := range ns {
		b = strconv.AppendInt(b, int64(n), 10)
		b = append(b, ',')
	}
	b = b[:len(b)-1]
	return string(b)
}
