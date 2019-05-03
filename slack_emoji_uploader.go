package main

import (
	"bufio"
	"container/list"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/from-unknown/slackemojiupload"
)

func main() {
	// Variables
	var slackURL string
	var email string
	var password string

	emojiList := os.Args[1:]

	log.Println("Reading emoji_conf.txt...")
	// Check ini file existance
	file, err := os.Open("./emoji_conf.txt")
	if err != nil {
		log.Fatal("Could not load emoji_conf.txt file.")
	}
	defer file.Close()

	// Read conf file
	confList := list.New()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		tmp := scanner.Text()
		tmp = strings.TrimSpace(tmp)
		if tmp != "" && string(tmp[0:1]) != "#" {
			confList.PushBack(tmp)
		}
	}

	// confList size must be multiple of 3
	if confList.Len()%3 != 0 {
		log.Fatal("Config file doesn't have enough settings.")
	}

	confLen := confList.Len() / 3

	var wg sync.WaitGroup
	// slack team base loop
	for confCounter := 0; confCounter < confLen; confCounter++ {
		// Set conf data
		slackURL = confList.Remove(confList.Front()).(string)
		email = confList.Remove(confList.Front()).(string)
		password = confList.Remove(confList.Front()).(string)

		wg.Add(1)
		go slackemojiupload.SlackEmojiUpload(&wg, slackURL, email, password, emojiList...)
	}
	wg.Wait()
}
