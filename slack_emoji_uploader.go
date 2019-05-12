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
	var slackToken string

	emojiList := os.Args[1:]

	f, err := os.OpenFile("emoji.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	logger := log.New(f, "", log.LstdFlags)

	logger.Println("Reading emoji_conf.txt...")
	// Check ini file existence
	file, err := os.Open("./emoji_conf.txt")
	if err != nil {
		logger.Fatal("Could not load emoji_conf.txt file.")
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

	if confList.Len() < 2 {
		logger.Println("please setup emoji_conf.txt file")
		return
	}

	slackApiURL := confList.Remove(confList.Front()).(string)

	defaultEmojiList, err := slackemojiupload.GetDefaultEmojiList()
	if err != nil {
		logger.Fatal(err)
	}
	var wg sync.WaitGroup
	// slack token base loop
	for confList.Len() > 0 {

		// Set conf data
		slackToken = confList.Remove(confList.Front()).(string)
		s, err := slackemojiupload.New(slackApiURL, slackToken, logger, slackemojiupload.LogLevelInfo)
		if err != nil {
			logger.Println(err)
			continue
		}

		existEmojiList, err := s.GetExistEmojiList()
		if err != nil {
			logger.Println(err)
			continue
		}
		existEmojiList.PushBackList(defaultEmojiList)

		wg.Add(1)
		s.SetEmojiList(existEmojiList)
		logger.Println(slackToken[0:10] + "... start")
		go s.SlackEmojiUpload(&wg, emojiList)
	}
	wg.Wait()
}
