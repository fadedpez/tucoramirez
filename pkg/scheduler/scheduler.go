package scheduler

import (
	"context"
	"log"
	"sync"
	"time"
)

// Task represents a scheduled task
type Task struct {
	Name     string
	Interval time.Duration
	Fn       func(context.Context) error
}

// Scheduler manages scheduled tasks
type Scheduler struct {
	tasks   []*Task
	running bool
	mutex   sync.Mutex
	cancel  context.CancelFunc
}

// NewScheduler creates a new scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks:   make([]*Task, 0),
		running: false,
	}
}

// AddTask adds a task to the scheduler
func (s *Scheduler) AddTask(name string, interval time.Duration, fn func(context.Context) error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.tasks = append(s.tasks, &Task{
		Name:     name,
		Interval: interval,
		Fn:       fn,
	})
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.running {
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.running = true

	for _, task := range s.tasks {
		go s.runTask(ctx, task)
	}

	log.Println("Scheduler started with", len(s.tasks), "tasks")
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return
	}

	s.cancel()
	s.running = false
	log.Println("Scheduler stopped")
}

// runTask runs a task at the specified interval
func (s *Scheduler) runTask(ctx context.Context, task *Task) {
	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop()

	// Run the task immediately on startup
	log.Printf("Running task %s immediately on startup", task.Name)
	err := task.Fn(ctx)
	if err != nil {
		log.Printf("Error running task %s: %v", task.Name, err)
	}

	for {
		select {
		case <-ticker.C:
			log.Printf("Running scheduled task: %s", task.Name)
			err := task.Fn(ctx)
			if err != nil {
				log.Printf("Error running task %s: %v", task.Name, err)
			}
		case <-ctx.Done():
			log.Printf("Task %s stopped", task.Name)
			return
		}
	}
}
