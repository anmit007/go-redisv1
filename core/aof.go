package core

import (
	"anmit007/go-redis/config"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func dumpKey(fp *os.File, k string, v *Obj) {
	cmd := fmt.Sprintf("SET %s %s", k, v.Value)
	tokens := strings.Split(cmd, " ")
	fp.Write(Encode(tokens, false))
}

func LoadAOF() error {
	log.Println("Loading AOF file at", config.AOFFILEPATH)
	fp, err := os.Open(config.AOFFILEPATH)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer fp.Close()
	data, err := io.ReadAll(fp)
	if err != nil {
		return err
	}
	values, err := Decode(data)
	if err != nil {
		return err
	}
	for _, val := range values {
		tokens, ok := val.([]interface{})
		if !ok {
			continue
		}
		args := make([]string, len(tokens))
		for i, t := range tokens {
			args[i] = t.(string)
		}

		cmd := strings.ToUpper(args[0])
		switch cmd {
		case "SET":
			evalSet(args[1:])
		case "DEL":
			evalDEL(args[1:])
		}

	}
	return nil
}
