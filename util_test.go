package main

import "testing"

func Test_pad(t *testing.T) {
	type args struct {
		first string
		last  string
	}
	tests := []struct {
		args args
		want string
	}{
		{args{"Бaтя", "-1h46m31s"}, "Бaтя..........-1h46m31s"},
		{args{"Добрый Бобр", "11m28s"}, "Добрый Бобр......11m28s"},
		{args{"PaulBenzeneSuperStar", "-1h30m31s"}, "PaulBenzeneSu.-1h30m31s"},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := pad(tt.args.first, tt.args.last); got != tt.want {
				t.Errorf("pad() = %v, want %v", got, tt.want)
			}
		})
	}
}
