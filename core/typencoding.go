package core

import "errors"

func getEncoding(te uint8) uint8 {
	return te & 0b00001111
}

func getType(te uint8) uint8 {
	return (te >> 4) << 4
}

func assertType(te uint8, t uint8) error {
	if getType(te) != t {
		return errors.New("invalid type encoding")
	}
	return nil
}

func assertEncoding(te uint8, e uint8) error {
	if getEncoding(te) != e {
		return errors.New("this op is not permitted for this encoding")
	}
	return nil
}
