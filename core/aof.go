package core

import (
	"anmit007/go-redis/config"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

var aofFile *os.File
var aofBuffer bytes.Buffer

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
		// Use internal functions that don't call AppendAOF to avoid duplicating data
		switch cmd {
		case "SET":
			internalSet(args[1:])
		case "DEL":
			internalDEL(args[1:])
		case "EXPIRE":
			internalExpire(args[1:])
		case "INCR":
			internalIncr(args[1:])
		}

	}
	return nil
}

func InitAOF() error {
	var err error
	aofFile, err = os.OpenFile(config.AOFFILEPATH, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	return err
}

func CloseAOF() {
	FlushAOF()
	if aofFile != nil {

		aofFile.Close()
	}
}

func AppendAOF(cmd string, args []string) {

	tokens := append([]string{cmd}, args...)
	aofBuffer.Write(Encode(tokens, false))

}

func FlushAOF() {
	if aofBuffer.Len() == 0 {
		return
	}
	aofFile.Write(aofBuffer.Bytes())
	aofFile.Sync()
	aofBuffer.Reset()

}
