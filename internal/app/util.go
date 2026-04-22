package app

func mergeAgents(old, new []string) []string {
	m := make(map[string]bool)
	for _, a := range old {
		m[a] = true
	}
	for _, a := range new {
		m[a] = true
	}
	res := []string{}
	for a := range m {
		res = append(res, a)
	}
	return res
}
