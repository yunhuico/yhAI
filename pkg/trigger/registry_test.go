package trigger

import "testing"

func Test_parseTimeInDay(t *testing.T) {
	tests := []struct {
		timeInDay  string
		wantHour   int
		wantMin    int
		wantSecond int
		wantErr    bool
	}{
		{
			timeInDay:  "",
			wantHour:   0,
			wantMin:    0,
			wantSecond: 0,
			wantErr:    true,
		},
		{
			timeInDay:  "00:00:00",
			wantHour:   0,
			wantMin:    0,
			wantSecond: 0,
			wantErr:    false,
		},
		{
			timeInDay:  "13:50:46",
			wantHour:   13,
			wantMin:    50,
			wantSecond: 46,
			wantErr:    false,
		},
		{
			timeInDay:  "05:02:07",
			wantHour:   5,
			wantMin:    2,
			wantSecond: 7,
			wantErr:    false,
		},
		{
			timeInDay:  "13",
			wantHour:   13,
			wantMin:    50,
			wantSecond: 46,
			wantErr:    true,
		},
		{
			timeInDay:  "13:",
			wantHour:   13,
			wantMin:    50,
			wantSecond: 46,
			wantErr:    true,
		},
		{
			timeInDay:  "13:50",
			wantHour:   13,
			wantMin:    50,
			wantSecond: 46,
			wantErr:    true,
		},
		{
			timeInDay:  "13:50:",
			wantHour:   13,
			wantMin:    50,
			wantSecond: 46,
			wantErr:    true,
		},
		{
			timeInDay:  "13:50:2",
			wantHour:   13,
			wantMin:    50,
			wantSecond: 46,
			wantErr:    true,
		},
		{
			timeInDay:  "13:50:2a",
			wantHour:   13,
			wantMin:    50,
			wantSecond: 46,
			wantErr:    true,
		},
		{
			timeInDay:  "1:2:3",
			wantHour:   13,
			wantMin:    50,
			wantSecond: 46,
			wantErr:    true,
		},
		{
			timeInDay:  "-05:02:07",
			wantHour:   5,
			wantMin:    2,
			wantSecond: 7,
			wantErr:    true,
		},
		{
			timeInDay:  "24:02:07",
			wantHour:   5,
			wantMin:    2,
			wantSecond: 7,
			wantErr:    true,
		},
		{
			timeInDay:  "05:60:07",
			wantHour:   5,
			wantMin:    2,
			wantSecond: 7,
			wantErr:    true,
		},
		{
			timeInDay:  "05:02:60",
			wantHour:   5,
			wantMin:    2,
			wantSecond: 60,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.timeInDay, func(t *testing.T) {
			gotHour, gotMin, gotSecond, err := ParseTimeInDay(tt.timeInDay)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTimeInDay() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if gotHour != tt.wantHour {
				t.Errorf("parseTimeInDay() gotHour = %v, want %v", gotHour, tt.wantHour)
			}
			if gotMin != tt.wantMin {
				t.Errorf("parseTimeInDay() gotMin = %v, want %v", gotMin, tt.wantMin)
			}
			if gotSecond != tt.wantSecond {
				t.Errorf("parseTimeInDay() gotSecond = %v, want %v", gotSecond, tt.wantSecond)
			}
		})
	}
}
