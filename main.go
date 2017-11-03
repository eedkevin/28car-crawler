package main

import (
	"fmt"
	"net/http"

	"bitbucket.org/eedkevin/28car-crawler/database"
	"bitbucket.org/eedkevin/28car-crawler/parser"
	"bitbucket.org/eedkevin/28car-crawler/redis"
	"github.com/PuerkitoBio/fetchbot"
)

var (
	seed = "http://28car.com/sell_lst.php"

	linkQueue *fetchbot.Queue
	pageQueue *fetchbot.Queue

	redisQueue *redis.MyRedis
)

func main() {
	redisQueue = redis.New()
	linkFetcher := fetchbot.New(fetchbot.HandlerFunc(linkHandler))
	linkQueue = linkFetcher.Start()
	pageFetcher := fetchbot.New(fetchbot.HandlerFunc(pageHandler))
	pageQueue = pageFetcher.Start()

	linkQueue.SendStringGet(seed)

	go func() {
		for {
			msg, _ := redisQueue.ReceiveMessage()
			pageQueue.SendStringGet(msg.Payload)
		}
	}()

	linkQueue.Block()
	pageQueue.Block()
}

func linkHandler(ctx *fetchbot.Context, res *http.Response, err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Printf("[%d] %s %s\n", res.StatusCode, ctx.Cmd.Method(), ctx.Cmd.URL())

	task := parser.ParseLink(res)
	if task.NextPageUrl != "EOF" {
		linkQueue.SendStringGet(task.NextPageUrl)
	}

	for _, u := range task.ItemArray {
		redisQueue.Publish(u)
	}
}

func pageHandler(ctx *fetchbot.Context, res *http.Response, err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Printf("[%d] %s %s\n", res.StatusCode, ctx.Cmd.Method(), ctx.Cmd.URL())

	car := parser.ParsePage(res)

	errPersist := database.Persist(car)
	if errPersist != nil {
		fmt.Println("error on persisting data")
	}
}

func fakePageHandler(ctx *fetchbot.Context, res *http.Response, err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Printf("[%d] %s %s\n", res.StatusCode, ctx.Cmd.Method(), ctx.Cmd.URL())
}
