package sanitizer

import (
	"github.com/iypetrov/lambdas/filter-sns-topic-from-amazonq-chat/message"
)

type Func func(msg message.AlarmMessage) *message.AlarmMessage

var Sanitize = NewChainedSanitizer(NewStandardSanitizers()...)

func NewStandardSanitizers() []Func {
	return []Func{
		NewOffHoursSanitizer(),
	}
}

func NewChainedSanitizer(sanitizers ...Func) Func {
	if len(sanitizers) == 1 {
		return sanitizers[0]
	}
	return func(msg message.AlarmMessage) *message.AlarmMessage {
		for _, s := range sanitizers {
			res := s(msg)
			if res == nil {
				return nil
			}
			msg = *res
		}
		return &msg
	}
}
