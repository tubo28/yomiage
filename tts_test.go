package main

import (
	"testing"
)

func TestSanitize(t *testing.T) {
	type args struct {
		s    string
		lang string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal text should not be modified",
			args: args{
				s:    "hello hello",
				lang: "ja-JP",
			},
			want: "hello hello",
		},
		{
			name: "continuous whitespace chars should be single space",
			args: args{
				s:    "　\nhello   hello  ",
				lang: "ja-JP",
			},
			want: "hello hello",
		},
		{
			name: "URL should be replaced with 'URL'",
			args: args{
				s:    "http://example.com example.com 8.8.8.8",
				lang: "ja-JP",
			},
			want: "URL URL URL",
		},
		{
			name: "text in parenthesis should become empty",
			args: args{
				s:    "（aaa）  ",
				lang: "ja-JP",
			},
			want: "",
		},
		{
			name: "prefix 'w's should be replaced with kusa",
			args: args{
				s:    "あwww",
				lang: "ja-JP",
			},
			want: "あ くさ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sanitize(tt.args.s, tt.args.lang); got != tt.want {
				t.Errorf("Sanitize() = %v, want %v", got, tt.want)
			}
		})
	}
}
