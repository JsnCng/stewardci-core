package test

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/SAP/stewardci-core/pkg/k8s"
	"go.opencensus.io/trace"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	interval = 1 * time.Second
	timeout  = 2 * time.Minute
)

// Waiter is a waiter waiting for a condition to be fullfilled
type Waiter interface {
	WaitFor(t *testing.T, condition WaitCondition) error
}

type waiter struct {
	clientFactory k8s.ClientFactory
}

// NewWaiter returns a new Waiter
func NewWaiter(clientFactory k8s.ClientFactory) Waiter {
	return &waiter{clientFactory: clientFactory}
}

// WaitFor waits for a condition
// it returns an error if condition is not fullfilled
func (w *waiter) WaitFor(t *testing.T, condition WaitCondition) error {
	t.Helper()
	startTime := time.Now()
	log.Printf("wait for %s", condition.Name())
	_, span := trace.StartSpan(context.Background(), condition.Name())
	defer span.End()
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		return condition.Check(w.clientFactory)
	})
	log.Printf("waiting completed for %s after %s", condition.Name(), time.Now().Sub(startTime))
	return err
}

// WaitFor waits for a condition
func (w *waiter) WaitForX(t *testing.T, condition WaitCondition) error {
	t.Helper()
	log.Printf("wait for %s", condition.Name())
	startTime := time.Now()
	for {
		result, err := condition.Check(w.clientFactory)
		if err != nil {
			log.Printf("waiting completed for %s after %s", condition.Name(), time.Now().Sub(startTime))
			return err
		}
		if result {
			break
		}
		time.Sleep(interval)
	}
	log.Printf("waiting completed for %s after %s", condition.Name(), time.Now().Sub(startTime))
	return nil
}
