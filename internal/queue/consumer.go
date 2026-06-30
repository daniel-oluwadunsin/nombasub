package queue

import "log"

type HandlerFunc func(body []byte) error

type Consumer struct {
	conn     *Connection
	handlers map[string]HandlerFunc
}

func NewConsumer(conn *Connection) *Consumer {
	return &Consumer{
		conn:     conn,
		handlers: make(map[string]HandlerFunc),
	}
}

func (c *Consumer) Register(queue string, handler HandlerFunc) {
	c.handlers[queue] = handler
}

func (c *Consumer) Start() {
	for queue, handler := range c.handlers {
		go c.consume(queue, handler)
	}
}

func (c *Consumer) consume(queue string, handler HandlerFunc) {
	msgs, err := c.conn.Channel().Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		log.Printf("failed to consume queue %s: %v", queue, err)
		return
	}
	for msg := range msgs {
		if err := handler(msg.Body); err != nil {
			log.Printf("handler error for queue %s: %v", queue, err)
			msg.Nack(false, true)
			continue
		}
		msg.Ack(false)
	}
}
