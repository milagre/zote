package deleter

import "github.com/milagre/zote/go/zamqp"

const QueueName = "item-deletions"

var Declarations = zamqp.Declarations{
	Queues: []zamqp.Queue{{Name: QueueName}},
}
