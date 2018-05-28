package main

import (
	"flag"
	"log"
	"os"

	"sync"

	"os/signal"
	"syscall"
	"time"

	"github.com/nsqio/go-nsq"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var fatalErr error

func fatal(e error) {
	log.Println(e)
	flag.PrintDefaults()
	fatalErr = e
}

func main() {

	var counts map[string]int
	var countsLock sync.Mutex
	var updater *time.Timer

	const updateDuration = 5 * time.Second

	defer func() {
		if fatalErr != nil {
			os.Exit(1)
		}
	}()

	log.Println("Connection to database...")
	db, err := mgo.Dial("localhost")
	if err != nil {
		fatal(err)
		return
	}

	defer func() {
		log.Println("Closing database connection...")
		db.Close()
	}()

	pollData := db.DB("twittervotes").C("polls")

	log.Println("Connecting to nsq...")
	q, err := nsq.NewConsumer("votes", "counter", nsq.NewConfig())
	if err != nil {
		fatal(err)
		return
	}

	q.AddHandler(nsq.HandlerFunc(func(m *nsq.Message) error {
		countsLock.Lock()
		defer countsLock.Unlock()
		if counts == nil {
			counts = make(map[string]int)
		}
		vote := string(m.Body)
		counts[vote]++
		return nil
	}))

	if err := q.ConnectToNSQLookupd("localhost:4161"); err != nil {
		fatal(err)
		return
	}

	log.Println("Waiting for votes on nsq...")

	updater = time.AfterFunc(updateDuration, func() {
		countsLock.Lock()
		defer countsLock.Unlock()
		if len(counts) == 0 {
			log.Println("No new votes, skipping database update")
		} else {
			log.Println("Updating database...")
			log.Println(counts)
			ok := true
			for options, count := range counts {
				sel := bson.M{"options": bson.M{"$in": []string{options}}}
				up := bson.M{"$inc": bson.M{"results." + options: count}}
				if _, err := pollData.UpdateAll(sel, up); err != nil {
					log.Println("Failed to update: ", err)
					ok = false
				}
			}
			if ok {
				log.Println("Finished updating database...")
				counts = nil
			}
		}
		updater.Reset(updateDuration)
	})

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		select {
		case <-termChan:
			updater.Stop()
			q.Stop()
		case <-q.StopChan:
			return
		}
	}
}
