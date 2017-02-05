package main

import (
	"bufio"
	"bytes"
	"container/list"
	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf"
	"github.com/nfnt/resize"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

func main() {
	// Variables
	var slack_url string
	var email string
	var password string

	const imageMax float64 = 128.0

	const imageSizeMax int = 64

	const loginFail string = "Sorry, you entered an incorrect email address or password."
	const addSuccess string = "Your new emoji has been saved"
	const postFix string = "_resized"

	f, err := os.OpenFile("emoji.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(f)

	log.Println("Reading default_emoji.txt...")
	// Check default emoji file exists
	file, err := os.Open("./default_emoji.txt")
	if err != nil {
		log.Fatal("Could not load default_emoji.txt file.")
	}
	defer file.Close()

	// Read default emoji file
	defaultEmojiList := list.New()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		tmp := scanner.Text()
		tmp = strings.TrimSpace(tmp)
		defaultEmojiList.PushBack(tmp)
	}

	log.Println("Reading emoji_conf.txt...")
	// Check ini file existance
	file, err = os.Open("./emoji_conf.txt")
	if err != nil {
		log.Fatal("Could not load emoji_conf.txt file.")
	}
	defer file.Close()

	// Read conf file
	confList := list.New()
	scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		tmp := scanner.Text()
		tmp = strings.TrimSpace(tmp)
		if tmp != "" && string(tmp[0:1]) != "#" {
			confList.PushBack(tmp)
		}
	}

	// confList size must be greater than 4
	if confList.Len()%3 != 0 {
		log.Fatal("Config file doesn't have enough settings.")
	}

	confLen := confList.Len() / 3

	for confCounter := 0; confCounter < confLen; confCounter++ {
		// Set conf data
		slack_url = confList.Remove(confList.Front()).(string)
		email = confList.Remove(confList.Front()).(string)
		password = confList.Remove(confList.Front()).(string)

		argsLen := len(os.Args)
		for counter := 1; counter < argsLen; counter++ {
			filePath := os.Args[counter]
			tmpExt := strings.ToLower(filepath.Ext(filePath))
			if tmpExt != ".jpg" && tmpExt != ".jpeg" && tmpExt != ".png" && tmpExt != ".gif" {
				log.Fatal("Only jpeg or png or gif file are allowed.")
			}

			emojiBase := filepath.Base(filePath)
			emojiName := emojiBase[0 : len(emojiBase)-len(tmpExt)]
			resizedFileName := emojiBase[0:len(emojiBase)-len(tmpExt)] + postFix + tmpExt

			// Name check - only 0-9, a-z, -_ are allowed
			match, err := regexp.MatchString("^[0-9a-z_-]+$", emojiName)
			if err != nil {
				log.Fatal(err)
			}
			if !match {
				log.Fatal("Custom emoji names can only contain lower case letters, numbers, dashes and underscores.")
			}

			err = resizeImage(filePath, imageMax)
			if err != nil {
				log.Fatal(err)
			}

			// File size check

			log.Println("Opening slack team page...")
			// Create a new browser and open Slack team
			bow := surf.NewBrowser()
			err = bow.Open(slack_url)
			if err != nil {
				log.Fatal("Could not access slack team.")
			}

			log.Println("Trying to sign in...")
			// Log in to the site
			fm, _ := bow.Form("form#signin_form")
			if err != nil {
				log.Fatal("Could not access signin form.")
			}

			fm.Input("email", email)
			fm.Input("password", password)
			if fm.Submit() != nil {
				log.Fatal("Could not sign in to slack team.")
			}

			// Check login success or failed
			bow.Find("p.alert_error").Each(func(_ int, s *goquery.Selection) {
				tmpStr := strings.TrimSpace(s.Text())
				if tmpStr == loginFail {
					log.Fatal("Could not sign in to slack team.\nPlease check email and password.")
				}
			})

			log.Println("Accessing to customize page...")
			// Open customize/emoji page
			err = bow.Open(slack_url + "customize/emoji")
			if err != nil {
				log.Fatal("Could not access slack customize emoji page.")
			}

			// Find registered emoji from webpage
			existList := list.New()
			bow.Find("td.align_middle").Each(func(_ int, s *goquery.Selection) {
				// emoji name are formatted :xxx: form, so use regexp to check
				match, err = regexp.MatchString(":*:", s.Text())
				if err != nil {
					log.Fatal("Error while checking web page.")
				}
				if match {
					tmpStr := strings.TrimSpace(s.Text())
					tmpStr = tmpStr[1 : utf8.RuneCountInString(tmpStr)-1]
					existList.PushBack(tmpStr)
				}
			})

			if includeInList(defaultEmojiList, emojiName) ||
				includeInList(existList, emojiName) {
				log.Fatal("Emoji name " + emojiName + " already Exists.")
			}

			log.Println("Uploading emoji...")
			// Upload emoji
			fm, _ = bow.Form("form#addemoji")
			if err != nil {
				log.Println("Error while finding emoji form.")
			}

			fm.Input("name", emojiName)
			read, err := ioutil.ReadFile(resizedFileName)
			if err != nil {
				log.Println("Error while opening emoji (" + emojiName + ") file.")
			}
			fm.File("img", resizedFileName, bytes.NewBuffer(read))
			fm.Input("mode", "data")
			if fm.Submit() != nil {
				log.Println("Error while submitting emoji.")
			}
			bow.Find("p.alert_success").Each(func(_ int, s *goquery.Selection) {
				tmpStr := strings.TrimSpace(s.Text())
				if err != nil {
					log.Fatal("Error while checking web page.")
				}
				if strings.Index(tmpStr, addSuccess) > -1 {
					log.Println(emojiName + " successfully added.")
				} else {
					log.Println(emojiName + " could not added.")
				}
			})
		}
	}
}

// Check if target is in the list or not.
func includeInList(l *list.List, target string) bool {
	for e := l.Front(); e != nil; e = e.Next() {
		if e.Value == target {
			return true
		}
	}
	return false
}

func resizeImage(filePath string, maxSize float64) error {
	const postFix string = "_resized"
	base := filepath.Base(filePath)
	ext := filepath.Ext(filePath)
	ext = strings.ToLower(ext)

	imageFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer imageFile.Close()

	var decImage image.Image
	var gifImage *gif.GIF
	var imageConfig image.Config
	if ext == ".jpg" || ext == ".jpeg" {
		decImage, err = jpeg.Decode(imageFile)
		if err != nil {
			log.Println(err)
		}
		_, err = imageFile.Seek(io.SeekStart, 0)
		if err != nil {
			return err
		}
		imageConfig, err = jpeg.DecodeConfig(imageFile)
		if err != nil {
			return err
		}
	} else if ext == ".png" {
		decImage, err = png.Decode(imageFile)
		if err != nil {
			return err
		}
		_, err = imageFile.Seek(io.SeekStart, 0)
		if err != nil {
			return err
		}
		imageConfig, err = png.DecodeConfig(imageFile)
		if err != nil {
			return err
		}
	} else if ext == ".gif" {
		gifImage, err = gif.DecodeAll(imageFile)
		if err != nil {
			return err
		}
		imageConfig = gifImage.Config
		if err != nil {
			return err
		}
	} else {
		return nil
	}

	width := float64(imageConfig.Width)
	height := float64(imageConfig.Height)

	var ratio float64
	if width > height && width > maxSize {
		ratio = maxSize / width
	} else if height > maxSize {
		ratio = maxSize / height
	} else {
		ratio = 1
	}

	tmpFileName := base[0:len(base)-len(ext)] + postFix + ext
	tmpFile, err := os.Create(tmpFileName)
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	if ratio == 1 {
		_, err = imageFile.Seek(io.SeekStart, 0)
		if err != nil {
			return err
		}
		_, err := io.Copy(tmpFile, imageFile)
		if err != nil {
			log.Fatal(err)
			return err
		}
	} else {
		if ext == ".jpg" || ext == ".jpeg" {
			resized := resize.Resize(uint(math.Floor(width*ratio)), uint(math.Floor(height*ratio)),
				decImage, resize.Lanczos3)
			jpeg.Encode(tmpFile, resized, nil)
		} else if ext == ".png" {
			resized := resize.Resize(uint(math.Floor(width*ratio)), uint(math.Floor(height*ratio)),
				decImage, resize.Lanczos3)
			png.Encode(tmpFile, resized)
		} else if ext == ".gif" {
			for index, frame := range gifImage.Image {
				rect := frame.Bounds()
				tmpImage := frame.SubImage(rect)
				resizedImage := resize.Resize(uint(math.Floor(float64(rect.Dx())*ratio)),
					uint(math.Floor(float64(rect.Dy())*ratio)),
					tmpImage, resize.Lanczos3)
				// Add colors from original gif image
				var tmpPalette color.Palette
				for x := 1; x <= rect.Dx(); x++ {
					for y := 1; y <= rect.Dy(); y++ {
						if !contains(tmpPalette, gifImage.Image[index].At(x, y)) {
							tmpPalette = append(tmpPalette, gifImage.Image[index].At(x, y))
						}
					}
				}

				// After first image, image may contains only difference
				// bounds may not start from at (0,0)
				resizedBounds := resizedImage.Bounds()
				if index >= 1 {
					marginX := int(math.Floor(float64(rect.Min.X) * ratio))
					marginY := int(math.Floor(float64(rect.Min.Y) * ratio))
					resizedBounds = image.Rect(marginX, marginY, resizedBounds.Dx()+marginX,
						resizedBounds.Dy()+marginY)
				}
				resizedPalette := image.NewPaletted(resizedBounds, tmpPalette)
				draw.Draw(resizedPalette, resizedBounds, resizedImage, image.ZP, draw.Src)
				gifImage.Image[index] = resizedPalette
			}
			// Set size to resized size
			gifImage.Config.Width = int(math.Floor(width * ratio))
			gifImage.Config.Height = int(math.Floor(height * ratio))
			gif.EncodeAll(tmpFile, gifImage)
		}
	}
	return nil
}

// Check if color is already in the Palette
func contains(colorPalette color.Palette, c color.Color) bool {
	for _, tmpColor := range colorPalette {
		if tmpColor == c {
			return true
		}
	}
	return false
}
