package deleter

import "github.com/milagre/zote/go/zamqp"

var _ zamqp.Message = DeleteItemMessage{}

// DeleteItemMessage is published after an item is soft-deleted; the deleter hard-deletes by id.
type DeleteItemMessage struct {
	ItemID string `json:"item_id"`
}

func (m DeleteItemMessage) Content() ([]byte, string, error) {
	return zamqp.MarshalJsonContent(m)
}

func (m DeleteItemMessage) Options() zamqp.MessageOptions {
	return zamqp.MessageOptions{RoutingKey: QueueName}
}

func (m DeleteItemMessage) Exchange() zamqp.Exchange {
	return zamqp.AnonymousExchange
}
