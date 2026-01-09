package core

import (
	"anmit007/go-redis/config"
	"log"
	"os"
	"sync/atomic"
)

var rewriteInProgress atomic.Bool

func snapshotStore() map[string]*Obj {
	storeMu.RLock()
	defer storeMu.RUnlock()

	snapshot := make(map[string]*Obj, len(store))
	for k, v := range store {
		snapshot[k] = v
	}
	return snapshot
}

func BGRewriteAOF() {

	if !rewriteInProgress.CompareAndSwap(false, true) {
		return
	}
	log.Println("Rewriting AOF file at", config.AOFFILEPATH)

	snapShot := snapshotStore()
	go func() {
		defer rewriteInProgress.Store(false)
		tempFilePath := config.AOFFILEPATH + ".tmp"
		fp, err := os.OpenFile(tempFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Println("error in AOF", err)
			return
		}
		for k, v := range snapShot {
			dumpKey(fp, k, v)
		}
		fp.Close()
		os.Rename(tempFilePath, config.AOFFILEPATH)
	}()

}
