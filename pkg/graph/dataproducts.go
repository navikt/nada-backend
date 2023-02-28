package graph

func contains(keywords []string, keyword string) bool {
	for _, k := range keywords {
		if k == keyword {
			return true
		}
	}
	return false
}
