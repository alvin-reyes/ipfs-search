package amqp

import (
	"context"
	"github.com/ipfs-search/ipfs-search/instr"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/label"
	"log"
	"time"
)

const (
	maxReconnect  = 100
	reconnectTime = 2 * time.Second
)

// Connection wraps an AMQP connection
type Connection struct {
	conn *amqp.Connection
	*instr.Instrumentation
}

// NewConnection returns new AMQP connection
func NewConnection(ctx context.Context, url string, i *instr.Instrumentation) (*Connection, error) {
	ctx, span := i.Tracer.Start(ctx, "queue.amqp.NewConnection", trace.WithAttributes(label.String("amqp_url", url)))
	defer span.End()

	amqpConn, err := amqp.Dial(url)

	if err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return nil, err
	}

	c := &Connection{
		conn:            amqpConn,
		Instrumentation: i}

	blockChan := amqpConn.NotifyBlocked(make(chan amqp.Blocking))
	closeChan := amqpConn.NotifyClose(make(chan *amqp.Error))

	monitorConn := func() {
		ctx, span := i.Tracer.Start(ctx, "queue.amqp.monitorConn", trace.WithAttributes(label.Stringer("connection", c)))
		defer span.End()

		errCnt := 0
		for {
			select {
			case <-ctx.Done():
				err := ctx.Err()
				span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
				return
			case b := <-blockChan:
				if b.Active {
					span.AddEvent(ctx, "amqp-connection-blocked",
						label.String("reason", b.Reason),
					)
					log.Printf("AMQP connection blocked")
				} else {
					span.AddEvent(ctx, "amqp-connection-unblocked")
					log.Printf("AMQP connection unblocked")
				}
			case err := <-closeChan:
				span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
				log.Printf("AMQP connection lost, attempting reconnect in %s", reconnectTime)
				time.Sleep(reconnectTime)

				amqpConn, amqpErr := amqp.Dial(url)
				if amqpErr != nil {
					if errCnt > maxReconnect {
						// TODO: Proper error propagation/recovery
						span.RecordError(ctx, amqpErr, trace.WithErrorStatus(codes.Error))
						panic("Repeated AMQP reconnect errors")
					} else {
						errCnt++
						log.Printf("Error connecting to AMQP: %v", amqpErr)
						span.RecordError(ctx, amqpErr)
					}

				}

				// Set new connection
				c.conn = amqpConn
			}
		}
	}
	go monitorConn()

	return c, nil
}

// Channel creates an AMQP channel
func (c *Connection) Channel(ctx context.Context) (*Channel, error) {
	ctx, span := c.Tracer.Start(ctx, "queue.amqp.Channel")
	defer span.End()

	// Create channel
	ch, err := c.conn.Channel()
	if err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return nil, err
	}

	// Set Qos
	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		span.RecordError(ctx, err, trace.WithErrorStatus(codes.Error))
		return nil, err
	}

	return &Channel{
		ch:              ch,
		Instrumentation: c.Instrumentation,
	}, nil
}

// NewChannelQueue returns a new queue on a new channel
func (c *Connection) NewChannelQueue(ctx context.Context, name string) (*Queue, error) {
	ctx, span := c.Tracer.Start(ctx, "queue.amqp.NewChannelQueue", trace.WithAttributes(label.String("queue", name)))
	defer span.End()

	ch, err := c.Channel(ctx)
	if err != nil {
		return nil, err
	}

	return ch.Queue(ctx, name)
}

func (c *Connection) String() string {
	return c.conn.LocalAddr().String()
}

// Close closes the channel
func (c *Connection) Close() error {
	return c.conn.Close()
}
