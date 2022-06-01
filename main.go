package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Quote struct {
	Spark struct {
		Result []struct {
			Response []struct {
				Meta struct {
					RegularMarketPrice float64 `json:regularMarketPrice`
					Currency           string  `json:currency`
					RegularMarketTime  uint64  `json:regularMarketTime`
				} `json:meta`
			} `json:response`
		} `json:result`
	} `json:"spark"`
}

const API = "https://query1.finance.yahoo.com/v7/finance/spark?symbols=%s&range=1m"

type Price struct {
	Commodity string
	Currency  string
	Price     float64
}

func main() {
	priceDbFile := flag.String("p", "prices.db", "Price Database File")
	mappingFile := flag.String("m", "mapping", "Commodities Name Mapping File")
	flag.Parse()

	mappings := GetMapping(*mappingFile)

	pricedb, err := os.OpenFile(*priceDbFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Price database file access failed with %s\n", err)
	}
	defer pricedb.Close()

	currency := mappings["$"]
	log.Println("Currency", currency)
	prices := make(map[string]Price)

	for c, ticker := range mappings {
		if c == "$" {
			continue
		}

		price, err := GetPriceString(c, ticker)
		if err != nil {
			log.Println("Skipped " + c)
			continue
		}
		prices[c] = price
	}

	for c, p := range prices {
		if p.Currency == currency {
			p.Currency = "$"
		}
		result := fmt.Sprintf("P %s %s %f %s\n", GetTimeString(), c, p.Price, p.Currency)
		pricedb.WriteString(result)
	}
	log.Println("Stock price update complete")
}

func GetPriceString(commodity, ticker string) (price Price, err error) {
	resp, err := http.Get(fmt.Sprintf(API, ticker))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var f Quote
	err = json.Unmarshal(body, &f)
	if err != nil {
		return
	}

	price = Price{
		Commodity: commodity,
		Price:     f.Spark.Result[0].Response[0].Meta.RegularMarketPrice,
		Currency:  f.Spark.Result[0].Response[0].Meta.Currency,
	}
	return
}

func GetTimeString() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func GetMapping(mappingFile string) map[string]string {
	result := make(map[string]string)
	file, err := os.Open(mappingFile)
	if err != nil {
		log.Fatalf("Open mapping file failed: %s\n", err)
	}
	defer file.Close()

	fileScanner := bufio.NewScanner(file)

	for fileScanner.Scan() {
		arr := strings.Split(fileScanner.Text(), ":")
		if len(arr) == 2 {
			result[arr[0]] = strings.Trim(arr[1], "\n")
		}
	}
	return result
}

func IsTicker(s string) bool {
	for _, e := range s {
		if (e < 'A' || e > 'Z') && (e < '0' || e > '9') && e != '.' {
			return false
		}
	}
	return true
}
