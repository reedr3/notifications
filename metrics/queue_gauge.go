package metrics

import (
	"fmt"
	"log"
	"time"
)

type QueueGauge struct {
	logger *log.Logger
	queue  queue
	timer  <-chan time.Time
}

type queue interface {
	Len() (int, error)
	RetryQueueLengths() (map[int]int, error)
}

func NewQueueGauge(queue queue, logger *log.Logger, timer <-chan time.Time) QueueGauge {
	return QueueGauge{
		logger: logger,
		queue:  queue,
		timer:  timer,
	}
}

func (g QueueGauge) Run() {
	for _ = range g.timer {
		length, _ := g.queue.Len()
		retryCounts, _ := g.queue.RetryQueueLengths()

		for number, value := range retryCounts {
			NewMetric("gauge", map[string]interface{}{
				"name":  fmt.Sprintf("queue-retry-counts.%d", number),
				"value": value,
			}).LogWith(g.logger)
		}

		NewMetric("gauge", map[string]interface{}{
			"name":  "queue-length",
			"value": length,
		}).LogWith(g.logger)
	}
}
