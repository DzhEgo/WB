package service

import (
	"github.com/nats-io/stan.go"
	"log"
)

func ConnectNATS(clusterID, clientID, natsURL string) (stan.Conn, error) {
	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL(natsURL))
	if err != nil {
		return nil, err
	}
	return sc, nil
}

func SubscribeToNATS(sc stan.Conn, subject string, cb stan.MsgHandler) (stan.Subscription, error) {
	sub, err := sc.Subscribe(subject, cb, stan.DurableName("God"))
	if err != nil {
		return nil, err
	}
	return sub, nil
}

func Publish(sc stan.Conn, subject, message string) {
	if sc == nil {
		log.Println("NATS Streaming соединение не инициализировано")
		return
	}
	if err := sc.Publish(subject, []byte(message)); err != nil {
		log.Printf("Ошибка публикации сообщения в NATS Streaming: %v\n", err)
	}
}
