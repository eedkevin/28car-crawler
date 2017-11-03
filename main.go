package main

import (
	"fmt"
	"net/http"
	"strconv"

	"bitbucket.org/eedkevin/28car-crawler/database"
	"bitbucket.org/eedkevin/28car-crawler/parser"
	"bitbucket.org/eedkevin/28car-crawler/redis"
	"github.com/PuerkitoBio/fetchbot"
)

var (
	seed        = "http://28car.com/sell_lst.php"
	pageUrlTmpl = "http://28car.com/sell_lst.php?h_page="
	itemUrlTmpl = "http://28car.com/sell_dsp.php?h_vid="

	linkQueue *fetchbot.Queue
	pageQueue *fetchbot.Queue
	itemQueue *fetchbot.Queue

	pageQueueRedis *redis.MyRedis
	itemQueueRedis *redis.MyRedis
)

func main() {
	pageQueueRedis = redis.New("pages")
	itemQueueRedis = redis.New("items")

	linkFetcher := fetchbot.New(fetchbot.HandlerFunc(linkHandler))
	linkQueue = linkFetcher.Start()

	pageFetcher := fetchbot.New(fetchbot.HandlerFunc(pageHandler))
	pageQueue = pageFetcher.Start()

	itemFetcher := fetchbot.New(fetchbot.HandlerFunc(itemHandler))
	itemQueue = itemFetcher.Start()

	// drop the seed, bang!
	linkQueue.SendStringGet(seed)

	go func() {
		for {
			msg, _ := pageQueueRedis.ReceiveMessage()
			fmt.Println("new page: " + msg.Payload)
			pageQueue.SendStringGet(msg.Payload)
		}
	}()

	go func() {
		for {
			msg, _ := itemQueueRedis.ReceiveMessage()
			fmt.Println("new item: " + msg.Payload)
			itemQueue.SendStringGet(msg.Payload)
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

	maxPageNumStr := parser.ParseLink(res)
	if maxPageNumStr == "EOF" {
		return
	} else {
		maxPageNum, strconvErr := strconv.Atoi(maxPageNumStr)
		if strconvErr != nil {
			fmt.Println("Error on converting string to int. value - " + maxPageNumStr)
			return
		}
		for i := 0; i < maxPageNum; i++ {
			fmt.Println("push page to redis: " + pageUrlTmpl + strconv.Itoa(i))
			pageQueueRedis.Publish(pageUrlTmpl + strconv.Itoa(i))
		}
	}
}

func pageHandler(ctx *fetchbot.Context, res *http.Response, err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Printf("[%d] %s %s\n", res.StatusCode, ctx.Cmd.Method(), ctx.Cmd.URL())

	vidArr := parser.ParsePage(res)
	for _, vid := range vidArr {
		itemQueueRedis.Publish(itemUrlTmpl + vid)
	}
}

func itemHandler(ctx *fetchbot.Context, res *http.Response, err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Printf("[%d] %s %s\n", res.StatusCode, ctx.Cmd.Method(), ctx.Cmd.URL())

	car := parser.ParseItem(res)

	errPersist := database.Persist(car)
	if errPersist != nil {
		fmt.Printf("error on persisting data: %v\n", errPersist)
	}
}

func fakePageHandler(ctx *fetchbot.Context, res *http.Response, err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Printf("[%d] %s %s\n", res.StatusCode, ctx.Cmd.Method(), ctx.Cmd.URL())
}
