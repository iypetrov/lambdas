package sanitizer

import (
	"fmt"
	"testing"
	"time"

	"github.com/iypetrov/lambdas/filter-sns-topic-from-amazonq-chat/message"
)

type TestScenario struct {
	name       string
	alarmName  string
	mockTime   time.Time
	shouldDrop bool
}

func shouldDrop(
	criteria OffHoursCriteria,
	alarmName string,
	day time.Weekday,
	hour int,
) bool {
	switch alarmName {
	case "C17-test-gasx-https-int-healthy-instaces-alarm":
		if day == time.Saturday || day == time.Sunday {
			return true
		}

		if hour <= 6 || hour >= 18 {
			return true
		}

		return false
	case "C41-test-gasx-longRunning-healthy-instaces-alarm":
		if hour >= 22 && hour <= 23 {
			return true
		}

		return false
	case "C41-prod-gasx-longRunning-healthy-instaces-alarm":
		if hour >= 22 && hour <= 23 {
			return true
		}

		return false
	default:
		return false
	}
}

func TestOffHoursSanitizer(t *testing.T) {
	var tests []TestScenario

	criterias := OffHoursCriterias()

	for _, criteria := range criterias {
		if criteria.AlarmName != "C17-test-gasx-https-int-healthy-instaces-alarm" {
			continue
		}
		for day := time.Sunday; day <= time.Saturday; day++ {
			hours := make([]int, 24)
			for i := range hours {
				hours[i] = i
			}

			for _, hour := range hours {
				tests = append(tests, TestScenario{
					name:      fmt.Sprintf("%s-%s-%02d:00", criteria.AlarmName, day.String(), hour),
					alarmName: criteria.AlarmName,
					// Don't change the date, the 1st of the month should be Sunday
					mockTime:   time.Date(2025, 6, int(day)+1, hour, 0, 0, 0, time.UTC),
					shouldDrop: shouldDrop(criteria, criteria.AlarmName, day, hour),
				})
			}
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now = func() time.Time { return tt.mockTime }
			defer func() { now = time.Now }()

			msg := message.AlarmMessage{
				AlarmName: tt.alarmName,
			}
			result := sanitizeOffHoursSanitizer(msg)

			if tt.shouldDrop && result != nil {
				t.Errorf("Expected message to be dropped, got not nil")
			}
			if !tt.shouldDrop && result == nil {
				t.Errorf("Expected message to pass, got nil")
			}
		})
	}
}
