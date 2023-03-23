package handler

import (
	"testing"
)

func Test_checksum(t *testing.T) {
	tests := []struct {
		name string
		args int
		want int
	}{
		{
			name: "test 1 Luhn",
			args: 1234567890,
			want: 7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checksum(tt.args); got != tt.want {
				t.Errorf("checksum() = %v, want %v", got, tt.want)
			}
		})
	}
}
