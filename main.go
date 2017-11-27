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
	redisHost     = flag.String("redis", "localhost:6379", "redis host:port")
	mongoHost     = flag.String("mongo", "localhost:27017", "mongo host:port")
	userAgent     = flag.String("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36", "browser user agent")
	crawlDelay    = flag.Duration("crawl-delay", 10, "crawling delay")

	// memory queue (channels)
	linkQueue *fetchbot.Queue
	pageQueue *fetchbot.Queue
	itemQueue *fetchbot.Queue

	// redis queue
	pageQueueRedis *redis.MyRedis
	itemQueueRedis *redis.MyRedis

	// database
	db *database.MyMongo
)

func main() {
	flag.Parse()

	db = database.New(*mongoHost)
	pageQueueRedis = redis.New(*redisHost, "pages")
	itemQueueRedis = redis.New(*redisHost, "items")

	linkFetcher := fetchbot.New(fetchbot.HandlerFunc(linkHandler))
	linkFetcher.UserAgent = *userAgent
	linkQueue = linkFetcher.Start()

	pageFetcher := fetchbot.New(fetchbot.HandlerFunc(pageHandler))
	pageFetcher.UserAgent = *userAgent
	pageQueue = pageFetcher.Start()

	itemFetcher := fetchbot.New(fetchbot.HandlerFunc(itemHandler))
	itemFetcher.UserAgent = *userAgent
	itemQueue = itemFetcher.Start()

	// drop the seed, bang!
	linkQueue.SendStringGet(*seed)

	go func() {
		for {
			msg, err := pageQueueRedis.ReceiveMessage()
			if err != nil {
				panic(err)
			}
			fmt.Println("new page: " + msg.Payload)
			pageQueue.SendStringGet(msg.Payload)
			time.Sleep(*crawlDelay * time.Second)
		}
	}()

	go func() {
		for {
			msg, err := itemQueueRedis.ReceiveMessage()
			if err != nil {
				panic(err)
			}
			fmt.Println("new item: " + msg.Payload)
			itemQueue.SendStringGet(msg.Payload)
			time.Sleep(*crawlDelay * time.Second)
		}
	}()

	pageQueue.Block()
	itemQueue.Block()
}

func linkHandler(ctx *fetchbot.Context, res *http.Response, err error) {
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
		return
	} else {
		maxPageNum, strconvErr := strconv.Atoi(maxPageNumStr)
		if strconvErr != nil {
			fmt.Println("Error on converting string to int. value - " + maxPageNumStr)
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
		fmt.Printf("error on parsing page: %v\n", errParse)
		pageQueueRedis.Publish(res.Request.URL.String())
	}

	for _, vid := range vidArr {
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
		fmt.Printf("error on persisting data: %v\n", errPersist)
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
