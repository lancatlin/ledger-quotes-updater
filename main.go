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

func main() {
	apiToken := flag.String("a", "demo", "Alpha Vantage API Token")
	ledgerBinary := flag.String("b", "ledger", "Ledger Binary")
	ledgerFile := flag.String("f", "ledger.ledger", "Ledger File")
	priceDbFile := flag.String("p", "prices.db", "Price Database File")
	mappingFile := flag.String("m", "mapping", "Commodities Name Mapping File")
	flag.Parse()

	commodities := GetCommodities(*ledgerFile, *ledgerBinary, *mappingFile)

	pricedb, err := os.OpenFile(*priceDbFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Price database file access failed with %s\n", err)
	}
	defer pricedb.Close()

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

		priceString, err := GetPriceString(c, *apiToken)
		if err != nil {
			log.Println("Skipped " + c)
			continue
		}
		pricedb.WriteString("P " + GetTimeString() + " " + c + " " + priceString[:len(priceString)-2] + "\n")
	}
	log.Println("Stock price update complete")
}

func GetPriceString(ticker string, apiToken string) (string, error) {
	resp, err := http.Get(fmt.Sprintf(API, ticker))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var f Quote
	err = json.Unmarshal(body, &f)
	if err != nil {
		return "", err
	}

	log.Println(f)

	return fmt.Sprintf("$%f", f.Spark.Result[0].Response[0].Meta.RegularMarketPrice), nil
}

func GetTimeString() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func GetCommodities(ledger string, binary string, mappingFile string) []string {
	cmd := exec.Command(binary, "-f", ledger, "commodities")
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("Ledger file commodity report failed with %s\n", err)
	}
	a := strings.Split(string(out), "\n")
	sliceOut := a[:len(a)-1]
	mapping := GetMapping(mappingFile)

	commodities := make([]string, 0)
	for _, e := range sliceOut {
		e = strings.Trim(e, `"`)
		if value, ok := mapping[e]; ok {
			e = value
		}
		commodities = append(commodities, e)
	}
	log.Println(commodities)
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
	log.Println(result)
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
