package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Quote struct {
	Spark struct {
		Result []struct {
			Response []struct {
				Meta struct {
					RegularMarketPrice float64 `json:regularMarketPrice`
					Currency string `json:currency`
					RegularMarketTime uint64 `json:regularMarketTime`
				} `json:meta`
			} `json:response`
		} `json:result`
	} `json:"spark"`
}

const API = "https://query1.finance.yahoo.com/v7/finance/spark?symbols=%s&range=1m"

type Price struct {
	Commodity string
	Currency string
	Price float64
}

func main() {
	ledgerBinary := flag.String("b", "ledger", "Ledger Binary")
	ledgerFile := flag.String("f", "ledger.ledger", "Ledger File")
	priceDbFile := flag.String("p", "prices.db", "Price Database File")
	mappingFile := flag.String("m", "mapping", "Commodities Name Mapping File")
	flag.Parse()

	mappings := GetMapping(*mappingFile)
	commodities := GetCommodities(*ledgerFile, *ledgerBinary)

	pricedb, err := os.OpenFile(*priceDbFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Price database file access failed with %s\n", err)
	}
	defer pricedb.Close()

	currency := mappings["$"]
	log.Println("Currency", currency)
	prices := make(map[string]Price)

	start := time.Now()
	for i, c := range commodities {
		if i+1%5 == 0 {
			elapsed := time.Now().Sub(start)
			if elapsed < time.Minute {
				log.Println("Sleeping because of API limit")
				time.Sleep(time.Minute - elapsed)
			}
			start = time.Now()
		}
		if c == "$" {
			continue
		}

		ticker := c
		if value, ok := mappings[c]; ok {
			ticker = value
		}
		price, err := GetPriceString(c, ticker)
		if err != nil {
			log.Println("Skipped " + c)
			continue
		}
		prices[c] = price
	}

	for c, p := range prices {
		result := fmt.Sprintf("P %s %s $%f\n", GetTimeString(), c, p.GetPrice(currency, prices))
		pricedb.WriteString(result)
	}
	log.Println("Stock price update complete")
}

func (p Price) GetPrice(currency string, prices map[string]Price) float64 {
	if p.Commodity == currency {
		return 1
	}
	if p.Currency == currency {
		return p.Price
	}
	if altCurrency, ok := prices[p.Currency]; ok {
		return p.Price * altCurrency.GetPrice(currency, prices)
	}
	return 0
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
		Price: f.Spark.Result[0].Response[0].Meta.RegularMarketPrice,
		Currency: f.Spark.Result[0].Response[0].Meta.Currency,
	}
	return
}

func GetTimeString() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func GetCommodities(ledger string, binary string) []string {
	cmd := exec.Command(binary, "-f", ledger, "commodities")
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("Ledger file commodity report failed with %s\n", err)
	}
	a := strings.Split(string(out), "\n")
	sliceOut := a[:len(a)-1]

	commodities := make([]string, 0)
	for _, e := range sliceOut {
		e = strings.Trim(e, `"`)
		commodities = append(commodities, e)
	}
	return commodities
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
