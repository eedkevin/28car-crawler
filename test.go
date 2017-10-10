package main

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/PuerkitoBio/fetchbot"
	"github.com/PuerkitoBio/goquery"
)

func main() {
	f := fetchbot.New(fetchbot.HandlerFunc(handler))
	queue := f.Start()
	queue.SendStringGet("http://28car.com/sell_lst.php")
	queue.Close()
}

func handler(ctx *fetchbot.Context, res *http.Response, err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	fmt.Printf("[%d] %s %s\n", res.StatusCode, ctx.Cmd.Method(), ctx.Cmd.URL())

	r := regexp.MustCompile(`\d+`)

	doc, _ := goquery.NewDocumentFromResponse(res)
	doc.Find("div#tch_box").Each(func(i int, s *goquery.Selection) {
		row := s.Find("td[onclick^='goDsp']")

		row.Each(func(i int, s1 *goquery.Selection) {
			targetStr, _ := s1.Attr("onclick")
			vid := r.FindAllString(targetStr, -1)[1]
			fmt.Println(vid)
		})
	})
}
