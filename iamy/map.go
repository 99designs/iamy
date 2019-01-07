package iamy

func mapStringSetDifference(aa, bb map[string]string) map[string]string {
	rr := make(map[string]string)
	for k, v := range aa {
		if bb[k] != v {
			rr[k] = v
		}
	}
	return rr
}
