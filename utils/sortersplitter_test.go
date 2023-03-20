package utils

import (
	"reflect"
	"testing"
)

func TestSorterSplitter_Split(t *testing.T) {
	tests := []struct {
		name string
		o    SorterSplitter
		want [][][]string
	}{
		{
			name: "Normal string set",
			o: [][]string{
				{"badhost8", "sss_ddd_www"},
				{"host7", "sss_ddd_www", ""},
				{"host6", "sss_ddd_www", "24:00:00"},
				{"badhost32"},
				{"host1", "sss_ddd_www", "01:00:00"},
				{"host3", "sss_ddd_www", "12:00:00"},
				{"host4", "sss_ddd_www", ""},
				{"host2", "sss_ddd_www", "01:00:00"},
				{"badhost30", "sss_ddd_www"},
				{"host5", "sss_ddd_www", "24:00:00"},
				{"host4", "sss_ddd_www", "21:00:00"},
			},
			want: [][][]string{
				{
					{"host7", "sss_ddd_www", ""},
					{"host4", "sss_ddd_www", ""},
				},
				{
					{"host1", "sss_ddd_www", "01:00:00"},
					{"host2", "sss_ddd_www", "01:00:00"},
				},
				{
					{"host3", "sss_ddd_www", "12:00:00"},
				},
				{
					{"host4", "sss_ddd_www", "21:00:00"},
				},
				{
					{"host6", "sss_ddd_www", "24:00:00"},
					{"host5", "sss_ddd_www", "24:00:00"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.Split(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SorterSplitter.Split() = %v, want %v", got, tt.want)
			}
		})
	}
}
