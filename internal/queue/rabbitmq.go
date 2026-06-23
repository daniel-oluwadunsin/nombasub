package queue

import (
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Connection struct {
	url  string
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewConnection(url string) (*Connection, error) {
	c := &Connection{url: url}
	if err := c.connect(); err != nil {
		return nil, err
	}
	go c.reconnectLoop()
	return c, nil
}

func (c *Connection) connect() error {
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}
	c.conn = conn
	c.ch = ch
	return nil
}

func (c *Connection) reconnectLoop() {
	for {
		reason, ok := <-c.conn.NotifyClose(make(chan *amqp.Error))
		if !ok {
			return
		}
		log.Printf("rabbitmq connection closed: %v — reconnecting...", reason)
		for {
			time.Sleep(5 * time.Second)
			if err := c.connect(); err == nil {
				log.Println("rabbitmq reconnected")
				break
			}
		}
	}
}

func (c *Connection) Channel() *amqp.Channel {
	return c.ch
}

func (c *Connection) Close() {
	if c.ch != nil {
		c.ch.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}
