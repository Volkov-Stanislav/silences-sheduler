package utils

import "sort"

// SorterSplitter sort string array.
type SorterSplitter [][]string

func (o SorterSplitter) Len() int {
	return len(o)
}

// Less sorter interface.
func (o SorterSplitter) Less(i, j int) bool {
	if len(o[i]) > 2 && len(o[j]) > 2 {
		return o[i][2] < o[j][2]
	}

	return true
}

// Swap sorter interface.
func (o SorterSplitter) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

// Split splin array on sections with identical 2-nd field in string.
func (o SorterSplitter) Split() [][][]string {
	var (
		result [][][]string
		inx    string
	)

	sort.Sort(o)

	for _, line := range o {
		if len(line) > 2 {
			if len(result) == 0 {
				result = append(result, [][]string{})
			}

			if line[2] == inx {
				result[len(result)-1] = append(result[len(result)-1], line)
			} else {
				result = append(result, [][]string{})
				inx = line[2]
				result[len(result)-1] = append(result[len(result)-1], line)
			}
		} else if len(result) == 0 {
			result = append(result, [][]string{})
		}
	}

	return result
}
