package ztime

import (
	"time"
)

type Resolution time.Duration

const (
	ResolutionSecond = Resolution(time.Second)
	ResolutionMilli  = Resolution(time.Millisecond)
	ResolutionMicro  = Resolution(time.Microsecond)
	ResolutionNano   = Resolution(time.Nanosecond)
)
