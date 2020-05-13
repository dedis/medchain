package admin

// Find the index of an element in the slice. Return -1 and false if the value is not in the slice
func Find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}
