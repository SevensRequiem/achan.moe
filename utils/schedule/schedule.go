package schedule

import (
	"time"

	"achan.moe/logs"
)

type Task struct {
	Action   func()
	Duration time.Duration
}

type Scheduler struct {
	StartTime          time.Time
	LastUpdate         time.Time
	LastUpdateDuration time.Duration
	Tasks              []Task
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		StartTime: time.Now(),
		Tasks:     []Task{},
	}
}

func (s *Scheduler) ScheduleTask(task Task) {
	s.Tasks = append(s.Tasks, task)
}

func (s *Scheduler) Run() {
	s.StartTime = time.Now()
	logs.Info("Scheduler started at: %v", s.StartTime)

	for _, task := range s.Tasks {
		go func(t Task) {
			ticker := time.NewTicker(t.Duration)
			defer ticker.Stop()

			for range ticker.C {
				t.Action()
				s.LastUpdate = time.Now()
				s.LastUpdateDuration = s.LastUpdate.Sub(s.StartTime)
				logs.Info("Task executed: %v", t.Duration)
			}
		}(task)
	}
}
