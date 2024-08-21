package utils

import (
	"fmt"
	"path"
	"testing"
	"time"
)

func TestEqualFloat64(t *testing.T) {
	type args struct {
		f1 interface{}
		f2 interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{"TestEqual", args{52, "51"}, 1, false},
		{"one", args{52, "51.5"}, 1, false},
		{"one", args{52.01, "51"}, 1, false},
		{"one", args{52.01, "51.5"}, 1, false},
		{"one", args{"52", "51"}, 1, false},
		{"one", args{"52", "51.5"}, 1, false},
		{"one", args{"52.5", "51"}, 1, false},
		{"one", args{"52.5", "51.5"}, 1, false},

		{"one", args{52, "52"}, 0, false},
		{"one", args{52, "52.00"}, 0, false},
		{"one", args{52.00, "52"}, 0, false},
		{"one", args{52.00, "52.00"}, 0, false},

		{"one", args{51, "52"}, -1, false},
		{"one", args{51, "52.5"}, -1, false},
		{"one", args{51.5, "52"}, -1, false},
		{"one", args{51.5, "52.5"}, -1, false},
		{"one", args{"51.5", "52"}, -1, false},
		{"one", args{"51.5", "52.5"}, -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EqualFloat64(tt.args.f1, tt.args.f2)
			if (err != nil) != tt.wantErr {
				t.Errorf("EqualFloat64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EqualFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareHideString(t *testing.T) {
	fmt.Println(CompareHideString("350204199806138023", "3502************23", "*"))
	fmt.Println(CompareHideString("3502************23", "350204199806138023", "*"))
	fmt.Println(CompareHideString("*****1", "11111*", "*"))
	fmt.Println(CompareHideString("你说什么", "你说**", "*"))
}

func TestPathJoin(t *testing.T) {
	tt := []struct {
		paths []string
		want  string
	}{
		{[]string{"a", "b", "c"}, "a/b/c"},
		{[]string{"a", "b", "c/"}, "a/b/c/"},
		{[]string{"http://www.example.com/", "/sub", "/item/"}, "http:/www.example.com/sub/item"},
	}

	for _, tc := range tt {
		if got := path.Join(tc.paths...); got != tc.want {
			t.Errorf("PathJoin(%v) = %v, want %v", tc.paths, got, tc.want)
		}
	}
}

func TestElapsed(t *testing.T) {
	defer Elapsed(func(funcName string, elapsed time.Duration) {
		fmt.Println(funcName, elapsed)
	})()

	time.Sleep(time.Second * 5)
}
