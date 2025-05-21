package sanitizer

import (
	"slices"
	"time"

	"github.com/iypetrov/lambdas/filter-sns-topic-from-amazonq-chat/message"
)

var now = time.Now

type OffHoursCriteria struct {
	AlarmName string
	OffHours  map[time.Weekday][]int
}

func OffHoursCriterias() []OffHoursCriteria {
	return []OffHoursCriteria{
		{
			AlarmName: "C17-test-gasx-https-int-healthy-instaces-alarm",
			OffHours: map[time.Weekday][]int{
				time.Monday:    {0, 1, 2, 3, 4, 5, 6, 18, 19, 20, 21, 22, 23},
				time.Tuesday:   {0, 1, 2, 3, 4, 5, 6, 18, 19, 20, 21, 22, 23},
				time.Wednesday: {0, 1, 2, 3, 4, 5, 6, 18, 19, 20, 21, 22, 23},
				time.Thursday:  {0, 1, 2, 3, 4, 5, 6, 18, 19, 20, 21, 22, 23},
				time.Friday:    {0, 1, 2, 3, 4, 5, 6, 18, 19, 20, 21, 22, 23},
				time.Saturday:  {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23},
				time.Sunday:    {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23},
			},
		},
		{
			AlarmName: "C41-test-gasx-longRunning-healthy-instaces-alarm",
			OffHours: map[time.Weekday][]int{
				time.Monday:    {22, 23},
				time.Tuesday:   {22, 23},
				time.Wednesday: {22, 23},
				time.Thursday:  {22, 23},
				time.Friday:    {22, 23},
				time.Saturday:  {22, 23},
				time.Sunday:    {22, 23},
			},
		},
		{
			AlarmName: "C41-prod-gasx-longRunning-healthy-instaces-alarm",
			OffHours: map[time.Weekday][]int{
				time.Monday:    {22, 23},
				time.Tuesday:   {22, 23},
				time.Wednesday: {22, 23},
				time.Thursday:  {22, 23},
				time.Friday:    {22, 23},
				time.Saturday:  {22, 23},
				time.Sunday:    {22, 23},
			},
		},
	}
}

func CurrentDayAndHour() (time.Weekday, int) {
	return now().UTC().Weekday(), now().UTC().Hour()
}

func NewOffHoursSanitizer() Func {
	return sanitizeOffHoursSanitizer
}

func sanitizeOffHoursSanitizer(msg message.AlarmMessage) *message.AlarmMessage {
	day, hour := CurrentDayAndHour()
	for _, criteria := range OffHoursCriterias() {
		if msg.AlarmName != criteria.AlarmName {
			continue
		}

		if slices.Contains(criteria.OffHours[day], hour) {
			return nil
		}
	}

	return &msg
}
