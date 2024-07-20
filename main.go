package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Today struct {
	XMLName       xml.Name `xml:"today"`
	SalePrice     string   `xml:"saleprice"`
	BuyPrice      string   `xml:"buyprice"`
	BuyPriceChg   string   `xml:"buypricechg"`
	SumOfChg      string   `xml:"SumOfChg"`
	UsdThb        string   `xml:"usdthb"`
	UsdThbChg     string   `xml:"usdthbchg"`
	GoldSpot      string   `xml:"goldspot"`
	GoldSpotChg   string   `xml:"goldspotchg"`
	NymexCrude    string   `xml:"nymexcrude"`
	NymexCrudeChg string   `xml:"nymexcrudechg"`
	SMS           string   `xml:"sms"`
	Update        string   `xml:"update"`
}

func fetchGoldPrice() (*Today, error) {
	resp, err := http.Get("https://www.namchiang.com/GoldPriceToday.xml")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var today Today
	err = xml.Unmarshal(body, &today)
	if err != nil {
		return nil, err
	}

	return &today, nil
}

func sendLineNotify(message string) error {
	token := os.Getenv("LINE_NOTIFY_TOKEN")
	url := "https://notify-api.line.me/api/notify"
	payload := strings.NewReader("message=" + message)

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("error: %s", string(body))
	}

	return nil
}

func formatMessage(today *Today) string {
	return fmt.Sprintf("Namchiang\n%s/%s\n%s | %s",
		today.BuyPrice,
		today.SalePrice,
		today.GoldSpot,
		today.UsdThb,
	)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	sleepTime, err := strconv.Atoi(os.Getenv("SLEEP_TIME"))
	if err != nil || sleepTime <= 0 {
		sleepTime = 30 // default sleep time in seconds
	}

	retryTime, err := strconv.Atoi(os.Getenv("RETRY_TIME"))
	if err != nil || retryTime <= 0 {
		retryTime = 10 // default retry time in seconds
	}

	var lastBuyPrice, lastSalePrice string

	for {
		today, err := fetchGoldPrice()
		if err != nil {
			log.Printf("Error fetching gold price: %v", err)
			time.Sleep(time.Duration(retryTime) * time.Second)
			continue
		}

		if today.BuyPrice != lastBuyPrice || today.SalePrice != lastSalePrice {
			message := formatMessage(today)
			err = sendLineNotify(message)
			if err != nil {
				log.Printf("Error sending Line Notify: %v", err)
			} else {
				log.Printf("Sent notification: %s", message)
				lastBuyPrice = today.BuyPrice
				lastSalePrice = today.SalePrice
			}
		} else {
			log.Printf("No price change detected.")
		}

		time.Sleep(time.Duration(sleepTime) * time.Second)
	}
}
