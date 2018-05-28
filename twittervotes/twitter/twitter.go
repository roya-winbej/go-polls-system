package twitter

import (
	"log"

	"books/go-blueprints/distributed-system/twittervotes/auth"
	"books/go-blueprints/distributed-system/twittervotes/db"
	"strings"

	"time"

	"net/http"

	"github.com/dghubble/go-twitter/twitter"
)

type poll struct {
	Options []string
}

func loadOptions() ([]string, error) {
	var options []string
	iter := db.DB.DB("twittervotes").C("polls").Find(nil).Iter()
	var p poll
	for iter.Next(&p) {
		options = append(options, p.Options...)
	}
	iter.Close()
	return options, iter.Err()
}

func StartTwitterStream(stopchan <-chan struct{}, votes chan<- string) <-chan struct{} {
	stoppedChan := make(chan struct{}, 1)

	defer func() {
		stoppedChan <- struct{}{}
	}()

	options, err := loadOptions()

	client := auth.NewTwitterClient()

	demux := twitter.NewSwitchDemux()

	demux.Tweet = func(tweet *twitter.Tweet) {
		log.Println("Recieved tweet: \n", tweet.Text)

		for _, option := range options {
			if strings.Contains(
				strings.ToLower(tweet.Text),
				strings.ToLower(option),
			) {
				log.Println("vote: ", option)
				votes <- option
			}
		}

		log.Println(" (polling timeout) ")
		time.Sleep(10 * time.Second)
	}

	log.Println("Starting Stream...")

	params := &twitter.StreamFilterParams{
		Track:         options,
		StallWarnings: twitter.Bool(true),
	}
	stream, err := client.Streams.Filter(params)
	_, res, _ := client.Trends.Available()

	if res.StatusCode == http.StatusUnauthorized {
		log.Fatalln("Oauth error: ", res.StatusCode)
	}

	if err != nil {
		log.Fatalln("Error when reading from twitter stream api: ", err)
	}

	go demux.HandleChan(stream.Messages)

	for {
		select {
		case <-stopchan:
			log.Println("Stopping Stream...")
			stream.Stop()
			return stoppedChan
		}
	}

	return stoppedChan
}
