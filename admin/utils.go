package admin

import "golang.org/x/xerrors"

// Find the index of an element in the slice. Return -1 and false if the value is not in the slice
func Find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

func Add(slice *[]string, val string) error {
	idx, _ := Find(*slice, val)
	if idx != -1 {
		return xerrors.New("The id is already registered")
	}
	// Add the new querier ID and access rights
	*slice = append(*slice, val)
	return nil
}

func Remove(slice *[]string, val string) error {
	idx, _ := Find(*slice, val)
	if idx == -1 {
		return xerrors.New("There is no such value")
	}
	// Add the new querier ID and access rights
	*slice = append((*slice)[:idx], (*slice)[idx+1:]...)
	return nil
}

func Update(slice *[]string, oldVal, newVal string) error {
	idx, _ := Find(*slice, oldVal)
	if idx == -1 {
		return xerrors.New("There is no such value")
	}
	(*slice)[idx] = newVal
	return nil
}
