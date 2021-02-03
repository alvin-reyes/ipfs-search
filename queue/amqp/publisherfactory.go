package amqp

import (
	"context"
	"github.com/ipfs-search/ipfs-search/instr"
	"github.com/ipfs-search/ipfs-search/queue"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/label"
	"log"
)

// PublisherFactory automates creation of AMQP Publishers.
type PublisherFactory struct {
	AMQPURL string
	Queue   string
	*instr.Instrumentation
}

func (f PublisherFactory) NewPublisher(ctx context.Context) (queue.Publisher, error) {
	ctx, span := f.Tracer.Start(ctx, "queue.amqp.NewPublisher",
		trace.WithAttributes(label.String("amqp_url", f.AMQPURL)),
		trace.WithAttributes(label.String("queue", f.Queue)),
	)
	defer span.End()

	// Create and configure add queue
	conn, err := NewConnection(ctx, f.AMQPURL, f.Instrumentation)
	if err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return nil, err
	}

	// Close connection when context closes
	go func() {
		<-ctx.Done()
		span.AddEvent(ctx, "closing-amqp-context-closed")
		log.Printf("Closing AMQP connection; context closed")
		conn.Close()
	}()

	return conn.NewChannelQueue(ctx, f.Queue)
}
