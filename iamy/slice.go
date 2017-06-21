package iamy

import "reflect"

// inlinePolicySetDifference is the set of elements in aa but not in bb
func inlinePolicySetDifference(aa, bb []InlinePolicy) []InlinePolicy {
	rr := []InlinePolicy{}

LoopInlinePolicies:
	for _, a := range aa {
		for _, b := range bb {
			if reflect.DeepEqual(a, b) {
				continue LoopInlinePolicies
			}
		}

		rr = append(rr, a)
	}

	return rr
}

// stringSetDifference is the set of elements in aa but not in bb
func stringSetDifference(aa, bb []string) []string {
	rr := []string{}

LoopStrings:
	for _, a := range aa {
		for _, b := range bb {
			if reflect.DeepEqual(a, b) {
				continue LoopStrings
			}
		}

		rr = append(rr, a)
	}

	return rr
}
