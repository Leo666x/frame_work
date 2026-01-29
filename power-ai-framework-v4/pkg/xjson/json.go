package xjson

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Escape returns an escaped path component.
//
//	json := `{
//	  "user":{
//	     "first.name": "Janet",
//	     "last.name": "Prichard"
//	   }
//	}`
//	user := gjson.Get(json, "user")
//	println(user.Get(gjson.Escape("first.name"))
//	println(user.Get(gjson.Escape("last.name"))
//	// Output:
//	// Janet
//	// Prichard
func Escape(comp string) string {
	return gjson.Escape(comp)
}

// Get searches json for the specified path.
// A path is in dot syntax, such as "name.last" or "age".
// When the value is found it's returned immediately.
//
// A path is a series of keys separated by a dot.
// A key may contain special wildcard characters '*' and '?'.
// To access an array value use the index as the key.
// To get the number of elements in an array or to access a child path, use
// the '#' character.
// The dot and wildcard character can be escaped with '\'.
//
//	{
//	  "name": {"first": "Tom", "last": "Anderson"},
//	  "age":37,
//	  "children": ["Sara","Alex","Jack"],
//	  "friends": [
//	    {"first": "James", "last": "Murphy"},
//	    {"first": "Roger", "last": "Craig"}
//	  ]
//	}
//	"name.last"          >> "Anderson"
//	"age"                >> 37
//	"children"           >> ["Sara","Alex","Jack"]
//	"children.#"         >> 3
//	"children.1"         >> "Alex"
//	"child*.2"           >> "Jack"
//	"c?ildren.0"         >> "Sara"
//	"friends.#.first"    >> ["James","Roger"]
//
// This function expects that the json is well-formed, and does not validate.
// Invalid json will not panic, but it may return back unexpected results.
// If you are consuming JSON from an unpredictable source then you may want to
// use the Valid function first.
func Get(json, path string) gjson.Result {
	return gjson.Get(json, path)
}

func GetBytes(json []byte, path string) gjson.Result {
	return gjson.GetBytes(json, path)
}

// GetMany searches json for the multiple paths.
// The return value is a Result array where the number of items
// will be equal to the number of input paths.
func GetMany(json string, path ...string) []gjson.Result {
	return gjson.GetMany(json, path...)
}

// GetManyBytes searches json for the multiple paths.
// The return value is a Result array where the number of items
// will be equal to the number of input paths.
func GetManyBytes(json []byte, path ...string) []gjson.Result {
	return gjson.GetManyBytes(json, path...)
}

// Set sets a json value for the specified path.
// A path is in dot syntax, such as "name.last" or "age".
// This function expects that the json is well-formed, and does not validate.
// Invalid json will not panic, but it may return back unexpected results.
// An error is returned if the path is not valid.
//
// A path is a series of keys separated by a dot.
//
//	{
//	  "name": {"first": "Tom", "last": "Anderson"},
//	  "age":37,
//	  "children": ["Sara","Alex","Jack"],
//	  "friends": [
//	    {"first": "James", "last": "Murphy"},
//	    {"first": "Roger", "last": "Craig"}
//	  ]
//	}
//	"name.last"          >> "Anderson"
//	"age"                >> 37
//	"children.1"         >> "Alex"
func Set(json, path string, value interface{}) (string, error) {
	return sjson.Set(json, path, value)
}

// SetBytes sets a json value for the specified path.
// If working with bytes, this method preferred over
// Set(string(data), path, value)
func SetBytes(json []byte, path string, value interface{}) ([]byte, error) {
	return sjson.SetBytes(json, path, value)
}

// SetRaw sets a raw json value for the specified path.
// This function works the same as Set except that the value is set as a
// raw block of json. This allows for setting premarshalled json objects.
func SetRaw(json, path, value string) (string, error) {
	return sjson.SetRaw(json, path, value)
}

// Delete deletes a value from json for the specified path.
func Delete(json, path string) (string, error) {
	return sjson.Delete(json, path)
}

// DeleteBytes deletes a value from json for the specified path.
func DeleteBytes(json []byte, path string) ([]byte, error) {
	return sjson.DeleteBytes(json, path)
}

// Valid Valid json
func Valid(json string) bool {
	return gjson.Valid(json)
}
