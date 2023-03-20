package utils

import (
	"reflect"
	"testing"
	"time"
)

func TestGetLocation(t *testing.T) {
	type args struct {
		inOffset string
	}
	tests := []struct {
		name string
		args args
		want *time.Location
	}{
		{
			name: "Test Location",
			args: args{
				"03:22:10",
			},
			want: time.FixedZone("UTC3", 3*60*60),
		},
		{
			name: "Test short offset ",
			args: args{
				"03",
			},
			want: time.FixedZone("UTC3", 3*60*60),
		},
		{
			name: "Short String, return local timezone",
			args: args{
				"03:00",
			},
			want: time.Local,
		},
		{
			name: "Empty string, return local timezone",
			args: args{
				"",
			},
			want: time.Local,
		},
		{
			name: "offset > 12, return local timezone",
			args: args{
				"24:00:00",
			},
			want: time.Local,
		},
		{
			name: "minus offset, return normal zone",
			args: args{
				"-08:00:00",
			},
			want: time.FixedZone("UTC-8", -8*60*60),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetLocation(tt.args.inOffset); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CalcLocation() = %v, want %v", got, tt.want)
			}
		})
	}
}
