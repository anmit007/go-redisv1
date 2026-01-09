package core

import (
	"bytes"
	"errors"
	"fmt"
)

func readSimpleString(data []byte) (interface{}, int, error) {
	pos := 1

	for ; data[pos] != '\r'; pos++ {
	}
	return string(data[1:pos]), pos + 2, nil
}

func readError(data []byte) (string, int, error) {
	val, n, err := readSimpleString(data)
	return val.(string), n, err
}

func readInt64(data []byte) (int64, int, error) {
	pos := 1
	var value int64 = 0
	for ; data[pos] != '\r'; pos++ {
		value = value*10 + int64(data[pos]-'0')
	}
	return value, pos + 2, nil
}

func readLength(data []byte) (int, int) {
	pos, length := 0, 0
	for pos = range data {
		b := data[pos]
		if !(b >= '0' && b <= '9') {
			return length, pos + 2
		}
		length = length*10 + int(b-'0')
	}
	return 0, 0
}
func readBulkString(data []byte) (string, int, error) {
	pos := 1

	length, delta := readLength(data[pos:])
	pos += delta
	if pos+length > len(data) {
		return "", 0, errors.New("incomplete bulk string")
	}
	return string(data[pos:(pos + length)]), pos + length + 2, nil
}

func readArray(data []byte) (interface{}, int, error) {

	pos := 1

	count, delta := readLength(data[pos:])
	pos += delta
	var elems []interface{} = make([]interface{}, count)
	for i := range elems {
		elem, delta, err := decodeOne(data[pos:])
		if err != nil {
			return nil, 0, err
		}
		elems[i] = elem
		pos += delta
	}
	return elems, pos, nil

}

func decodeOne(data []byte) (interface{}, int, error) {
	if len(data) == 0 {
		return nil, 0, errors.New("no data")
	}
	switch data[0] {
	case '+':
		return readSimpleString(data)
	case '-':
		return readError(data)
	case ':':
		return readInt64(data)
	case '$':
		return readBulkString(data)
	case '*':
		return readArray(data)
	default:
		return readInlineCommand(data)
	}
}

func readInlineCommand(data []byte) (interface{}, int, error) {
	pos := 0
	for pos < len(data) && data[pos] != '\r' && data[pos] != '\n' {
		pos++
	}
	if pos == 0 {
		return nil, 0, errors.New("empty inline command")
	}
	line := string(data[:pos])
	tokens := splitInlineCommand(line)
	result := make([]interface{}, len(tokens))
	for i, t := range tokens {
		result[i] = t
	}
	endPos := pos
	if endPos < len(data) && data[endPos] == '\r' {
		endPos++
	}
	if endPos < len(data) && data[endPos] == '\n' {
		endPos++
	}

	return result, endPos, nil
}

func splitInlineCommand(line string) []string {
	var tokens []string
	var current string
	inQuote := false

	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == '"' {
			inQuote = !inQuote
		} else if c == ' ' && !inQuote {
			if current != "" {
				tokens = append(tokens, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		tokens = append(tokens, current)
	}
	return tokens
}

func Decode(data []byte) ([]interface{}, error) {
	if len(data) == 0 {
		return nil, errors.New("no data")
	}
	var values []interface{} = make([]interface{}, 0)
	var index int = 0
	for index < len(data) {
		value, delta, err := decodeOne(data[index:])
		if err != nil {
			return values, err
		}
		index = index + delta
		values = append(values, value)
	}
	return values, nil
}

func Encode(value interface{}, isSimple bool) []byte {
	switch v := value.(type) {
	case string:
		if isSimple {
			return []byte(fmt.Sprintf("+%s\r\n", v))
		}
		return encodeString(v)
	case int, int8, int16, int32, int64:
		return []byte(fmt.Sprintf(":%d\r\n", v))
	case error:
		return []byte(fmt.Sprintf("-%s\r\n", v))
	case []string:
		var b []byte
		buf := bytes.NewBuffer(b)
		for _, b := range value.([]string) {
			buf.Write(encodeString(b))
		}
		return []byte(fmt.Sprintf("*%d\r\n%s", len(v), buf.Bytes()))
	default:
		return RESP_NIL
	}
}

func encodeString(v string) []byte {
	return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(v), v))
}
