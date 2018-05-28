package auth

import (
	"log"
	"os"

	"sync"

	"path/filepath"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/joho/godotenv"
)

var (
	authSetupOnce sync.Once
	twitterClient *twitter.Client
)

func NewTwitterClient() *twitter.Client {
	if twitterClient != nil {
		return twitterClient
	}

	authSetupOnce.Do(func() {
		envPath, err := filepath.Abs("../" + ".env")
		if err != nil {
			log.Fatal("Error finding .env file")
		}

		if err := godotenv.Load(envPath); err != nil {
			log.Fatal("Error loading .env file")
		}

		config := oauth1.NewConfig(os.Getenv("SP_TWITTER_KEY"), os.Getenv("SP_TWITTER_SECRET"))
		token := oauth1.NewToken(os.Getenv("SP_TWITTER_ACCESS_TOKEN"), os.Getenv("SP_TWITTER_ACCESS_SECRET"))

		httpClient := config.Client(oauth1.NoContext, token)

		twitterClient = twitter.NewClient(httpClient)
	})

	return twitterClient
}
