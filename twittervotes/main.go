package main

import (
	"books/go-blueprints/distributed-system/twittervotes/db"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"books/go-blueprints/distributed-system/twittervotes/twitter"
	"time"

	"github.com/nsqio/go-nsq"
)

func main() {
	if err := db.Dialdb(); err != nil {
		log.Fatalln("Failed to dial MongoDB: ", err)
	}
	defer db.Closedb()

	var stoplock sync.Mutex
	stop := false
	stopChan := make(chan struct{}, 1)
	signalChan := make(chan os.Signal, 1)
	go func() {
		<-signalChan
		stoplock.Lock()
		stop = true
		stoplock.Unlock()
		log.Println("Stopping...")
		stopChan <- struct{}{}
	}()
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	votes := make(chan string)
	publisherStoppedChan := publishVotes(votes)
	twitterStoppedChan := twitter.StartTwitterStream(stopChan, votes)

	go func() {
		for {
			time.Sleep(1 * time.Minute)
			stoplock.Lock()
			if stop {
				stoplock.Unlock()
				break
			}
			stoplock.Unlock()
		}
	}()

	<-twitterStoppedChan
	close(votes)
	<-publisherStoppedChan
}

func publishVotes(votes <-chan string) <-chan struct{} {
	stopchan := make(chan struct{}, 1)

	pub, _ := nsq.NewProducer("localhost:4150", nsq.NewConfig())
	go func() {
		for vote := range votes {
			log.Println("PUBLISHED vote: ", vote)
			pub.Publish("votes", []byte(vote))
		}
		log.Println("Publisher: Stopping")
		pub.Stop()
		log.Println("Publisher: Stopped")
		stopchan <- struct{}{}
	}()
	return stopchan
}
