package metadata

import (
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	type args struct {
		mds []map[string]string
	}
	tests := []struct {
		name string
		args args
		want Metadata
	}{
		{
			name: "hello",
			args: args{[]map[string]string{{"hello": "kratos"}, {"hello2": "go-kratos"}}},
			want: Metadata{"hello": "kratos", "hello2": "go-kratos"},
		},
		{
			name: "hi",
			args: args{[]map[string]string{{"hi": "kratos"}, {"hi2": "go-kratos"}}},
			want: Metadata{"hi": "kratos", "hi2": "go-kratos"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.mds...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetadata_Get(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name string
		m    Metadata
		args args
		want string
	}{
		{
			name: "kratos",
			m:    Metadata{"kratos": "value", "env": "dev"},
			args: args{key: "kratos"},
			want: "value",
		},
		{
			name: "env",
			m:    Metadata{"kratos": "value", "env": "dev"},
			args: args{key: "env"},
			want: "dev",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.Get(tt.args.key); got != tt.want {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetadata_Set(t *testing.T) {
	type args struct {
		key   string
		value string
	}
	tests := []struct {
		name string
		m    Metadata
		args args
		want Metadata
	}{
		{
			name: "kratos",
			m:    Metadata{},
			args: args{key: "hello", value: "kratos"},
			want: Metadata{"hello": "kratos"},
		},
		{
			name: "env",
			m:    Metadata{"hello": "kratos"},
			args: args{key: "env", value: "pro"},
			want: Metadata{"hello": "kratos", "env": "pro"},
		},
		{
			name: "empty",
			m:    Metadata{},
			args: args{key: "", value: ""},
			want: Metadata{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.m.Set(tt.args.key, tt.args.value)
			if !reflect.DeepEqual(tt.m, tt.want) {
				t.Errorf("Set() = %v, want %v", tt.m, tt.want)
			}
		})
	}
}

func TestMetadata_Range(t *testing.T) {
	md := Metadata{"kratos": "kratos", "https://go-kratos.dev/": "https://go-kratos.dev/", "go-kratos": "go-kratos"}
	tmp := Metadata{}
	md.Range(func(k, v string) bool {
		if k == "https://go-kratos.dev/" || k == "kratos" {
			tmp[k] = v
		}
		return true
	})
	if !reflect.DeepEqual(tmp, Metadata{"https://go-kratos.dev/": "https://go-kratos.dev/", "kratos": "kratos"}) {
		t.Errorf("metadata = %v, want %v", tmp, Metadata{"https://go-kratos.dev/": "https://go-kratos.dev/", "kratos": "kratos"})
	}
	tmp = Metadata{}
	md.Range(func(k, v string) bool {
		return false
	})
	if !reflect.DeepEqual(tmp, Metadata{}) {
		t.Errorf("metadata = %v, want %v", tmp, Metadata{})
	}
}

func TestMetadata_Clone(t *testing.T) {
	tests := []struct {
		name string
		m    Metadata
		want Metadata
	}{
		{
			name: "kratos",
			m:    Metadata{"kratos": "kratos", "https://go-kratos.dev/": "https://go-kratos.dev/", "go-kratos": "go-kratos"},
			want: Metadata{"kratos": "kratos", "https://go-kratos.dev/": "https://go-kratos.dev/", "go-kratos": "go-kratos"},
		},
		{
			name: "go",
			m:    Metadata{"language": "golang"},
			want: Metadata{"language": "golang"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.m.Clone()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Clone() = %v, want %v", got, tt.want)
			}
			got["kratos"] = "go"
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("want got != want got %v want %v", got, tt.want)
			}
		})
	}
}
