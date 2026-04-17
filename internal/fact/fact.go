package fact

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

var (
	ErrFactorizationCancelled = errors.New("cancelled")
	ErrWriterInteraction      = errors.New("writer interaction")
)

type Factorizer interface {
	Factorize(ctx context.Context, numbers []int, writer io.Writer) error
}

type factorizerImpl struct {
	factorizerWorkers int
	writerWorkers     int
}

func New(opts ...FactorizeOption) (*factorizerImpl, error) {
	const (
		defaultFactorizerWorkers = 1
		defaultWriterWorkers     = 1
	)

	f := &factorizerImpl{
		factorizerWorkers: defaultFactorizerWorkers,
		writerWorkers:     defaultWriterWorkers,
	}
	for _, opt := range opts {
		opt(f)
	}
	if err := f.isValid(); err != nil {
		return nil, err
	}

	return f, nil
}

func (f *factorizerImpl) isValid() error {
	if f.writerWorkers < 1 {
		return fmt.Errorf("invalid write workers value: %d", f.writerWorkers)
	}

	if f.factorizerWorkers < 1 {
		return fmt.Errorf("invalid factorization workers value: %d", f.factorizerWorkers)
	}

	return nil
}

type FactorizeOption func(*factorizerImpl)

func WithFactorizationWorkers(workers int) FactorizeOption {
	return func(f *factorizerImpl) {
		f.factorizerWorkers = workers
	}
}

func WithWriteWorkers(workers int) FactorizeOption {
	return func(f *factorizerImpl) {
		f.writerWorkers = workers
	}
}

func factorizeNumber(n int) string {
	if n == 0 {
		return "0 = 0"
	}
	if n == 1 {
		return "1 = 1"
	}
	if n == -1 {
		return "-1 = -1"
	}

	original := n
	factors := make([]string, 0)

	var x uint64
	if n < 0 {
		factors = append(factors, "-1")
		x = uint64(-(n + 1)) + 1
	} else {
		x = uint64(n)
	}

	for k := uint64(2); k*k <= x; k++ {
		for x%k == 0 {
			factors = append(factors, strconv.FormatUint(k, 10))
			x /= k
		}
	}

	if x > 1 {
		factors = append(factors, strconv.FormatUint(x, 10))
	}

	var b strings.Builder
	fmt.Fprint(&b, original, " = ", strings.Join(factors, " * "))
	return b.String()
}

func factorizationWorker(ctx context.Context, inputCh <-chan int, resultCh chan<- string) {
	for {
		select {
		case <-ctx.Done():
			return
		case n, ok := <-inputCh:
			if !ok {
				return
			}
			result := factorizeNumber(n)
			select {
			case <-ctx.Done():
				return
			case resultCh <- result:
			}
		}
	}
}

func writerWorker(ctx context.Context, resultCh <-chan string, writer io.Writer, setErr func(err error)) {
	for {
		select {
		case <-ctx.Done():
			return
		case result, ok := <-resultCh:
			if !ok {
				return
			}
			_, err := writer.Write([]byte(result + "\n"))
			if err != nil {
				setErr(fmt.Errorf("%w: %w", ErrWriterInteraction, err))
				return
			}
		}
	}
}

func (f *factorizerImpl) Factorize(parentCtx context.Context, numbers []int, writer io.Writer) error {
	resultCh := make(chan string)
	inputCh := make(chan int)

	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	var (
		factorWG     sync.WaitGroup
		writerWG     sync.WaitGroup
		producerWG   sync.WaitGroup
		once         sync.Once
		firstError   error
		resultClosed = make(chan struct{})
	)

	setErr := func(err error) {
		if err == nil {
			return
		}
		once.Do(func() {
			firstError = err
			cancel()
		})
	}

	for range f.factorizerWorkers {
		factorWG.Go(func() {
			factorizationWorker(ctx, inputCh, resultCh)
		})
	}

	for range f.writerWorkers {
		writerWG.Go(func() {
			writerWorker(ctx, resultCh, writer, setErr)
		})
	}

	producerWG.Go(func() {
		defer close(inputCh)
		for _, v := range numbers {
			select {
			case <-ctx.Done():
				return
			case inputCh <- v:
			}
		}
	})

	go func() {
		factorWG.Wait()
		close(resultCh)
		writerWG.Wait()
		producerWG.Wait()
		close(resultClosed)
	}()

	<-resultClosed

	if firstError != nil {
		return firstError
	}
	if cause := context.Cause(parentCtx); cause != nil {
		if errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded) {
			return ErrFactorizationCancelled
		}
		return fmt.Errorf("%w: %w", ErrFactorizationCancelled, cause)
	}

	return nil
}
