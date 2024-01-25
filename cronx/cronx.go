package cronx

import (
	"time"

	"github.com/robfig/cron/v3"
)

// ImmediateSchedule 加入cron任务后立即运行
// https://github.com/robfig/cron
type ImmediateSchedule struct {
	first    int32
	schedule cron.Schedule
}

func NewImmediateSchedule(spec string) (*ImmediateSchedule, error) {
	schedule, err := cron.ParseStandard(spec)
	if err != nil {
		return nil, err
	}

	return &ImmediateSchedule{schedule: schedule}, nil
}

func (schedule *ImmediateSchedule) Next(t time.Time) time.Time {
	if schedule.first == 0 {
		schedule.first = 1
		return t
	}

	return schedule.schedule.Next(t)
}
