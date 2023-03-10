package test

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/fetchbot"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

var (
	// Protect access to dup
	mu    sync.Mutex
	mutex sync.Mutex
	// Duplicates table
	dup = map[string]bool{}

	// Command-line flags
	//seed        = flag.String("seed", "http://28car.com/sell_lst.php", "seed URL")
	seed        = flag.String("seed", "http://28car.com/sell_dsp.php?h_vid=339539014", "seed URL")
	cancelAfter = flag.Duration("cancelafter", 0, "automatically cancel the fetchbot after a given time")
	cancelAtURL = flag.String("cancelat", "", "automatically cancel the fetchbot at a given URL")
	stopAfter   = flag.Duration("stopafter", 0, "automatically stop the fetchbot after a given time")
	stopAtURL   = flag.String("stopat", "", "automatically stop the fetchbot at a given URL")
	memStats    = flag.Duration("memstats", 0, "display memory statistics at a given interval")

	// regexp for links
	regexLink     = regexp.MustCompile(`\d+`)
	regexPrice    = regexp.MustCompile(`HKD\$\ ?(?P<num>[+-]?[0-9]{1,3}(?:,?[0-9])*(?:\.[0-9]{1,2})?)`)
	regexOrgPrice = regexp.MustCompile(`原價.?\$\ ?(?P<num>[+-]?[0-9]{1,3}(?:,?[0-9])*(?:\.[0-9]{1,2})?)`)

	// base url
	baseUrl = "http://28car.com/sell_dsp.php?h_vid="
)

func main() {
	flag.Parse()

	// Parse the provided seed
	u, err := url.Parse(*seed)
	if err != nil {
		log.Fatal(err)
	}

	// Create the muxer
	mux := fetchbot.NewMux()

	// Handle all errors the same
	mux.HandleErrors(fetchbot.HandlerFunc(func(ctx *fetchbot.Context, res *http.Response, err error) {
		fmt.Printf("[ERR] %s %s - %s\n", ctx.Cmd.Method(), ctx.Cmd.URL(), err)
	}))

	// Handle GET requests for html responses, to parse the body and enqueue all links as HEAD
	// requests.
	mux.Response().Method("GET").ContentType("text/html").Handler(fetchbot.HandlerFunc(
		func(ctx *fetchbot.Context, res *http.Response, err error) {
			// Process the body to find the links
			doc, err := goquery.NewDocumentFromResponse(res)
			if err != nil {
				fmt.Printf("[ERR] %s %s - %s\n", ctx.Cmd.Method(), ctx.Cmd.URL(), err)
				return
			}
			// Enqueue all links as HEAD requests
			//enqueueLinks(ctx, doc)
			parseContent(ctx, doc)
		}))

	// Handle HEAD requests for html responses coming from the source host - we don't want
	// to crawl links from other hosts.
	mux.Response().Method("HEAD").Host(u.Host).ContentType("text/html").Handler(fetchbot.HandlerFunc(
		func(ctx *fetchbot.Context, res *http.Response, err error) {
			if _, err := ctx.Q.SendStringGet(ctx.Cmd.URL().String()); err != nil {
				fmt.Printf("[ERR] %s %s - %s\n", ctx.Cmd.Method(), ctx.Cmd.URL(), err)
			}
		}))

	// Create the Fetcher, handle the logging first, then dispatch to the Muxer
	h := logHandler(mux)
	if *stopAtURL != "" || *cancelAtURL != "" {
		stopURL := *stopAtURL
		if *cancelAtURL != "" {
			stopURL = *cancelAtURL
		}
		h = stopHandler(stopURL, *cancelAtURL != "", logHandler(mux))
	}
	f := fetchbot.New(h)

	// First mem stat print must be right after creating the fetchbot
	if *memStats > 0 {
		// Print starting stats
		printMemStats(nil)
		// Run at regular intervals
		runMemStats(f, *memStats)
		// On exit, print ending stats after a GC
		defer func() {
			runtime.GC()
			printMemStats(nil)
		}()
	}

	// Start processing
	q := f.Start()

	// if a stop or cancel is requested after some duration, launch the goroutine
	// that will stop or cancel.
	if *stopAfter > 0 || *cancelAfter > 0 {
		after := *stopAfter
		stopFunc := q.Close
		if *cancelAfter != 0 {
			after = *cancelAfter
			stopFunc = q.Cancel
		}

		go func() {
			c := time.After(after)
			<-c
			stopFunc()
		}()
	}

	// Enqueue the seed, which is the first entry in the dup map
	dup[*seed] = true
	_, err = q.SendStringGet(*seed)
	if err != nil {
		fmt.Printf("[ERR] GET %s - %s\n", *seed, err)
	}
	q.Block()
}

func runMemStats(f *fetchbot.Fetcher, tick time.Duration) {
	var mu sync.Mutex
	var di *fetchbot.DebugInfo

	// Start goroutine to collect fetchbot debug info
	go func() {
		for v := range f.Debug() {
			mu.Lock()
			di = v
			mu.Unlock()
		}
	}()
	// Start ticker goroutine to print mem stats at regular intervals
	go func() {
		c := time.Tick(tick)
		for _ = range c {
			mu.Lock()
			printMemStats(di)
			mu.Unlock()
		}
	}()
}

func printMemStats(di *fetchbot.DebugInfo) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	buf := bytes.NewBuffer(nil)
	buf.WriteString(strings.Repeat("=", 72) + "\n")
	buf.WriteString("Memory Profile:\n")
	buf.WriteString(fmt.Sprintf("\tAlloc: %d Kb\n", mem.Alloc/1024))
	buf.WriteString(fmt.Sprintf("\tTotalAlloc: %d Kb\n", mem.TotalAlloc/1024))
	buf.WriteString(fmt.Sprintf("\tNumGC: %d\n", mem.NumGC))
	buf.WriteString(fmt.Sprintf("\tGoroutines: %d\n", runtime.NumGoroutine()))
	if di != nil {
		buf.WriteString(fmt.Sprintf("\tNumHosts: %d\n", di.NumHosts))
	}
	buf.WriteString(strings.Repeat("=", 72))
	fmt.Println(buf.String())
}

// stopHandler stops the fetcher if the stopurl is reached. Otherwise it dispatches
// the call to the wrapped Handler.
func stopHandler(stopurl string, cancel bool, wrapped fetchbot.Handler) fetchbot.Handler {
	return fetchbot.HandlerFunc(func(ctx *fetchbot.Context, res *http.Response, err error) {
		if ctx.Cmd.URL().String() == stopurl {
			fmt.Printf(">>>>> STOP URL %s\n", ctx.Cmd.URL())
			// generally not a good idea to stop/block from a linkHandler goroutine
			// so do it in a separate goroutine
			go func() {
				if cancel {
					ctx.Q.Cancel()
				} else {
					ctx.Q.Close()
				}
			}()
			return
		}
		wrapped.Handle(ctx, res, err)
	})
}

// logHandler prints the fetch information and dispatches the call to the wrapped Handler.
func logHandler(wrapped fetchbot.Handler) fetchbot.Handler {
	return fetchbot.HandlerFunc(func(ctx *fetchbot.Context, res *http.Response, err error) {
		if err == nil {
			fmt.Printf("[%d] %s %s - %s\n", res.StatusCode, ctx.Cmd.Method(), ctx.Cmd.URL(), res.Header.Get("Content-Type"))
		}
		wrapped.Handle(ctx, res, err)
	})
}

func enqueueLinks(ctx *fetchbot.Context, doc *goquery.Document) {
	mu.Lock()
	doc.Find("div#tch_box").Each(func(i int, s *goquery.Selection) {
		s.Find("td[onclick^='goDsp']").Each(func(i int, s1 *goquery.Selection) {
			targetStr, _ := s1.Attr("onclick")
			vid := regexLink.FindAllString(targetStr, -1)[1]
			val := baseUrl + vid
			// Resolve address
			u, err := ctx.Cmd.URL().Parse(val)
			if err != nil {
				fmt.Printf("error: resolve URL %s - %s\n", val, err)
				return
			}
			if !dup[u.String()] {
				if _, err := ctx.Q.SendStringHead(u.String()); err != nil {
					fmt.Printf("error: enqueue head %s - %s\n", u, err)
				} else {
					fmt.Println(u)
					dup[u.String()] = true
				}
			}
		})
	})
	mu.Unlock()
}

func parseContent(ctx *fetchbot.Context, doc *goquery.Document) {
	mutex.Lock()
	selector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody"
	selector = "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(10) > td.formt"
	doc.Find(selector).Each(func(i int, s *goquery.Selection) {
		str, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), s.Text())
		fmt.Println(str)

		price := regexPrice.FindStringSubmatch(str)[1]
		fmt.Println(price)
		fmt.Println(strings.Replace(price, ",", "", -1))

		oriPrice := regexOrgPrice.FindStringSubmatch(str)[1]
		fmt.Println(oriPrice)
		fmt.Println(strings.Replace(oriPrice, ",", "", -1))
	})
	mutex.Unlock()
}
