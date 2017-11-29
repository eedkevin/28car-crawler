package main

import (
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"bitbucket.org/eedkevin/28car-crawler/database"
	"bitbucket.org/eedkevin/28car-crawler/parser"
	"bitbucket.org/eedkevin/28car-crawler/redis"
	"github.com/PuerkitoBio/fetchbot"
)

var (
	// Command-line flags
	seed          = flag.String("seed", "http://28car.com/sell_lst.php", "seed URL")
	pageUrlPrefix = flag.String("page-url-prefix", "http://28car.com/sell_lst.php?h_page=", "page URL prefix")
	itemUrlPrefix = flag.String("item-url-prefix", "http://28car.com/sell_dsp.php?h_vid=", "item URL prefix")
	redisHost     = flag.String("redis", "localhost:6379", "redishost:port")
	mongoHost     = flag.String("mongo", "localhost:27017", "mongohost:port")
	userAgent     = flag.String("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36", "browser user agent")
	delay         = flag.Duration("delay", 10, "crawling delay in second")
	crawlerMode   = flag.String("crawler-mode", "", `crawler mode, "master" or "worker"`)

	// memory queue (channels)
	seedQueue *fetchbot.Queue
	pageQueue *fetchbot.Queue
	itemQueue *fetchbot.Queue

	// redis queue
	pageQueueRedis *redis.MyRedis
	itemQueueRedis *redis.MyRedis

	// database
	db *database.MyMongo
)

func main() {
	fmt.Println("web crawler started")
	flag.Parse()

	switch *crawlerMode {
	case "master":
		runMaster()
	case "worker":
		runWorker()
	default:
		fmt.Println("error: crawler-mode not assigned")
	}
	fmt.Println("web crawler terminated")
}

func runMaster() {
	fmt.Println("crawler-mode: master")

	pageQueueRedis = redis.New(*redisHost, "pages")
	itemQueueRedis = redis.New(*redisHost, "items")

	seedFetcher := fetchbot.New(fetchbot.HandlerFunc(seedHandler))
	seedFetcher.UserAgent = *userAgent
	seedQueue = seedFetcher.Start()

	pageFetcher := fetchbot.New(fetchbot.HandlerFunc(pageHandler))
	pageFetcher.UserAgent = *userAgent
	pageQueue = pageFetcher.Start()

	// drop the seed, bang!
	seedQueue.SendStringGet(*seed)

	go func() {
		for {
			msg, err := pageQueueRedis.ReceiveMessage()
			if err != nil {
				panic(err)
			}
			fmt.Println("new page: " + msg.Payload)
			pageQueue.SendStringGet(msg.Payload)
			time.Sleep(*delay * time.Second)
		}
	}()
	pageQueue.Block()
}

func runWorker() {
	fmt.Println("crawler-mode: worker")

	db = database.New(*mongoHost)
	itemQueueRedis = redis.New(*redisHost, "items")

	itemFetcher := fetchbot.New(fetchbot.HandlerFunc(itemHandler))
	itemFetcher.UserAgent = *userAgent
	itemQueue = itemFetcher.Start()

	go func() {
		for {
			msg, err := itemQueueRedis.ReceiveMessage()
			if err != nil {
				panic(err)
			}
			fmt.Println("new item: " + msg.Payload)
			itemQueue.SendStringGet(msg.Payload)
			time.Sleep(*delay * time.Second)
		}
	}()
	itemQueue.Block()
}

func seedHandler(ctx *fetchbot.Context, res *http.Response, err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Printf("[%d] %s %s\n", res.StatusCode, ctx.Cmd.Method(), ctx.Cmd.URL())

	maxPageNumStr, errParse := parser.ParseLink(res)
	if errParse != nil {
		// TODO: failback handling
		fmt.Printf("error on parsing page: %v\n", errParse)
		return
	} else if maxPageNumStr == "EOF" {
		fmt.Println("reach the end of the page")
		return
	} else {
		maxPageNum, strconvErr := strconv.Atoi(maxPageNumStr)
		if strconvErr != nil {
			fmt.Println("error on converting string to int. value - " + maxPageNumStr)
			return
		}
		for i := 0; i < maxPageNum; i++ {
			fmt.Println("push page to redis: " + *pageUrlPrefix + strconv.Itoa(i))
			pageQueueRedis.Publish(*pageUrlPrefix + strconv.Itoa(i))
		}
	}
}

func pageHandler(ctx *fetchbot.Context, res *http.Response, err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Printf("[%d] %s %s\n", res.StatusCode, ctx.Cmd.Method(), ctx.Cmd.URL())

	vidArr, errParse := parser.ParsePage(res)
	if errParse != nil {
		// failback handling
		fmt.Printf("error on parsing page, will re-push page to redis: %v\n", errParse)
		pageQueueRedis.Publish(res.Request.URL.String())
	}

	for _, vid := range vidArr {
		fmt.Println("push item to redis: " + *itemUrlPrefix + vid)
		itemQueueRedis.Publish(*itemUrlPrefix + vid)
	}
}

func itemHandler(ctx *fetchbot.Context, res *http.Response, err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Printf("[%d] %s %s\n", res.StatusCode, ctx.Cmd.Method(), ctx.Cmd.URL())

	car, errParse := parser.ParseItem(res)
	if errParse != nil {
		// failback handling
		fmt.Printf("error on parsing page: %v\n", errParse)
		itemQueueRedis.Publish(ctx.Cmd.URL().String())
	}

	errPersist := db.Persist(car)
	if errPersist != nil {
		// failback handling
		fmt.Printf("error on persisting item, will re-push item to redis: %v\n", errPersist)
		itemQueueRedis.Publish(ctx.Cmd.URL().String())
	}
}

func fakePageHandler(ctx *fetchbot.Context, res *http.Response, err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Printf("[%d] %s %s\n", res.StatusCode, ctx.Cmd.Method(), ctx.Cmd.URL())
}
