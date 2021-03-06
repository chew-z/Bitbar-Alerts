package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/johnmccabe/go-bitbar"
	_ "github.com/joho/godotenv/autoload"
)

/*Quotes - ..
 */
type Quotes []Quote

/*Quote - ..
 */
type Quote struct {
	Symbol           string  `json:"_symbol"`
	AskPrice         float64 `json:"_ask_price"`
	BidPrice         float64 `json:"_bid_price"`
	RefBidPrice      float64 `json:"_ref_bid_price"`
	HighBidPrice     float64 `json:"_high_bid_price"`
	LowBidPrice      float64 `json:"_low_bid_price"`
	BidDayChange     float64 `json:"_bid_day_change"`
	BidDayChangePcnt string  `json:"_bid_day_change_pcnt"`
	QuoteTm          int64   `json:"_quote_tm"`
	Pips             float64 `json:"_pips"`
	PipsLot          float64 `json:"_pips_lot"`
	Digits           float64 `json:"_digits"`
	MonthMin         float64 `json:"_30d_min_bid_price"`
	MonthMax         float64 `json:"_30d_max_bid_price"`
}

type displayQuote struct {
	time          string
	symbol        string
	bid           float64
	percentChange string
	change        float64
	high          float64
	low           float64
	webURL        string
	err           error
}

var (
	// assets         []string
	// apiURL, webURL string
	// city           string
	// ts, te         string
	apiURL = os.Getenv("API_URL")
	webURL = os.Getenv("WEB_URL")
	city   = os.Getenv("CITY")
	ts     = os.Getenv("TIME_START")
	te     = os.Getenv("TIME_END")
	assets = strings.Split(os.Getenv("ASSETS"), ":")

	// http.Clients should be reused instead of created as needed.
	client = &http.Client{
		Timeout: 5 * time.Second,
	}
	userAgent = randUserAgent()
)

func init() {
}

func main() {
	location, _ := time.LoadLocation(city)
	tn := time.Now().In(location).Format("1504")
	weekday := time.Now().Weekday()
	app := bitbar.New()
	if int(weekday) > 0 && int(weekday) < 6 && tn > ts && tn < te {
		submenu := app.NewSubMenu()
		// get all quotes in paralel
		resultsChan := make(chan *displayQuote)
		for _, asset := range assets {
			go getQuote(asset, resultsChan)
		}
		defer func() {
			close(resultsChan)
		}()
		// process results
		results := 0
		for {
			quote := <-resultsChan
			results++
			if quote.err != nil {
				// just quietly ignore errors - there is too many things that can go wrong
				// (wifi off, no internet, timeout etc.)
				// log.Println(quote.err.Error())
				submenu.Line(quote.err.Error()).Color("red")
			} else {
				var color string
				l := fmt.Sprintf("%s: %.5g %s", quote.symbol, quote.bid, quote.percentChange)
				line := app.StatusLine(l).DropDown(false)
				if quote.change < 0.0 {
					color = "red"
				} else {
					color = "green"
				}
				line.Color(color)
				m := fmt.Sprintf("%s - %s: %.5g %.5g", quote.time, quote.symbol, quote.bid, quote.change)
				a := fmt.Sprintf("%s: %.5g %s [%.5g - %.5g]", quote.symbol, quote.bid, quote.percentChange, quote.low, quote.high)
				submenu.Line(m).Href(quote.webURL).Color(color)
				submenu.Line(a).Alternate(true).Href(quote.webURL).Color(color)
			}
			// stop if we've received all quotes
			if results == len(assets) {
				break
			}
		}
	} else {
		app.StatusLine("Markets closed").DropDown(false)
	}
	app.Render()
}

/*getQuote
 */
func getQuote(asset string, ch chan<- *displayQuote) {
	var q displayQuote
	apiURL := fmt.Sprintf("%s%s.", apiURL, asset)
	request, _ := http.NewRequest("GET", apiURL, nil)
	request.Header.Set("User-Agent", userAgent)
	if response, err := client.Do(request); err == nil {
		var body Quotes
		json.NewDecoder(response.Body).Decode(&body)
		tm := time.Unix(0, body[0].QuoteTm*int64(time.Millisecond))
		location, _ := time.LoadLocation(city)
		q.time = tm.In(location).Format("15:04:05")
		q.symbol = asset
		q.bid = body[0].BidPrice
		q.change = body[0].BidDayChange
		q.percentChange = body[0].BidDayChangePcnt
		q.high = body[0].HighBidPrice
		q.low = body[0].LowBidPrice
		q.webURL = fmt.Sprintf("%s?a=%s", webURL, asset)
	} else {
		q.err = err
	}
	ch <- &q
}
