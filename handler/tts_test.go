package handler_test

import (
	"testing"

	"github.com/tubo28/yomiage/handler"
)

func TestSanitize(t *testing.T) {
	type args struct {
		content string
		lang    string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal text should not be modified",
			args: args{
				content: "hello hello",
				lang:    "ja-JP",
			},
			want: "hello hello",
		},
		{
			name: "continuous whitespace chars should be single space",
			args: args{
				content: "　\nhello   hello  ",
				lang:    "ja-JP",
			},
			want: "hello hello",
		},
		{
			name: "URL should be replaced with 'URL'",
			args: args{
				content: "http://example.com example.com 8.8.8.8",
				lang:    "ja-JP",
			},
			want: "URL URL URL",
		},
		{
			name: "text in parenthesis should become empty",
			args: args{
				content: "（aaa）  ",
				lang:    "ja-JP",
			},
			want: "",
		},
		{
			name: "prefix 'w's should be replaced with kusa",
			args: args{
				content: "あwww",
				lang:    "ja-JP",
			},
			want: "あ くさ",
		},
		{
			name: "user mention string should be replaced",
			args: args{
				content: "a <@!0123> <@4567> b",
				lang:    "ja-JP",
			},
			want: "a abc def b",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.Sanitize(tt.args.content, tt.args.lang); got != tt.want {
				t.Errorf("Sanitize() = %v, want %v", got, tt.want)
			}
		})
	}
}
