package core_test

import (
	"anmit007/go-redis/core"
	"fmt"
	"testing"
)

func TestSimpleStringDecode(t *testing.T) {
	cases := map[string]string{
		"+OK\r\n": "OK",
	}
	for k, v := range cases {
		values, _ := core.Decode([]byte(k))
		if len(values) == 0 {
			t.Fatalf("Expected at least one value, got none for input: %s", k)
		}
		value := values[0].(string)
		if v != value {
			t.Fatalf("Expected %s, got %s for input: %s", v, value, k)
		}
	}
}

func TestError(t *testing.T) {
	cases := map[string]string{
		"-Error message\r\n": "Error message",
	}
	for k, v := range cases {
		values, _ := core.Decode([]byte(k))
		if len(values) == 0 {
			t.Fatalf("Expected at least one value, got none for input: %s", k)
		}
		value := values[0].(string)
		if v != value {
			t.Fatalf("Expected %s, got %s for input: %s", v, value, k)
		}
	}
}

func TestInt64(t *testing.T) {
	cases := map[string]int64{
		":0\r\n":    0,
		":1000\r\n": 1000,
	}
	for k, v := range cases {
		values, _ := core.Decode([]byte(k))
		if len(values) == 0 {
			t.Fatalf("Expected at least one value, got none for input: %s", k)
		}
		value := values[0].(int64)
		if v != value {
			t.Fatalf("Expected %d, got %d for input: %s", v, value, k)
		}
	}
}

func TestBulkStringDecode(t *testing.T) {
	cases := map[string]string{
		"$5\r\nhello\r\n": "hello",
		"$0\r\n\r\n":      "",
	}
	for k, v := range cases {
		values, _ := core.Decode([]byte(k))
		if len(values) == 0 {
			t.Fatalf("Expected at least one value, got none for input: %s", k)
		}
		value := values[0].(string)
		if v != value {
			t.Fatalf("Expected %s, got %s for input: %s", v, value, k)
		}
	}
}

func TestArrayDecode(t *testing.T) {
	cases := map[string][]interface{}{
		"*0\r\n":                                                   {},
		"*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n":                     {"hello", "world"},
		"*3\r\n:1\r\n:2\r\n:3\r\n":                                 {int64(1), int64(2), int64(3)},
		"*5\r\n:1\r\n:2\r\n:3\r\n:4\r\n$5\r\nhello\r\n":            {int64(1), int64(2), int64(3), int64(4), "hello"},
		"*2\r\n*3\r\n:1\r\n:2\r\n:3\r\n*2\r\n+Hello\r\n-World\r\n": {[]interface{}{int64(1), int64(2), int64(3)}, []interface{}{"Hello", "World"}},
	}
	for k, v := range cases {
		values, _ := core.Decode([]byte(k))
		if len(values) == 0 {
			t.Fatalf("Expected at least one value, got none for input: %s", k)
		}
		array := values[0].([]interface{})
		if len(array) != len(v) {
			t.Fatalf("Expected array length %d, got %d for input: %s", len(v), len(array), k)
		}
		for i := range array {
			if fmt.Sprintf("%v", v[i]) != fmt.Sprintf("%v", array[i]) {
				t.Fatalf("Expected %v, got %v at index %d for input: %s", v[i], array[i], i, k)
			}
		}
	}
}
