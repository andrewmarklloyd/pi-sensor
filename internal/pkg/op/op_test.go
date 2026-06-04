package op

import (
	"testing"
)

func TestParseRateLimitOutput(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantLimited bool
		wantDur     string
		wantErr     bool
	}{
		{
			name: "no limits hit",
			input: `TYPE       ACTION        LIMIT    USED    REMAINING    RESET
token      write         100      0       100          N/A
token      read          1000     0       1000         N/A
account    read_write    1000     35      965          22 hours from now
`,
			wantLimited: false,
			wantDur:     "1h0m0s",
		},
		{
			name: "rate limited",
			input: `TYPE       ACTION        LIMIT    USED    REMAINING    RESET
token      write         100      0       0            57 minutes from now
`,
			wantLimited: true,
			wantDur:     "57m0s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limited, dur, err := parseRateLimitOutput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRateLimitOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if limited != tt.wantLimited {
				t.Errorf("parseRateLimitOutput() limited = %v, want %v", limited, tt.wantLimited)
			}
			if dur.String() != tt.wantDur {
				t.Errorf("parseRateLimitOutput() dur = %v, want %v", dur, tt.wantDur)
			}
		})
	}
}

func Test_parseOPResetDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "N/A",
			input:   "N/A",
			want:    "0s",
			wantErr: false,
		},
		{
			name:    "57 minutes from now",
			input:   "57 minutes from now",
			want:    "57m0s",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOPResetDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOPResetDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.String() != tt.want {
				t.Errorf("parseOPResetDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
