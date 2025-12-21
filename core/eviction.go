package core

func evict() {
	evictFirst()
}

func evictFirst() {
	for k := range store {
		delete(store, k)
		return
	}
}
