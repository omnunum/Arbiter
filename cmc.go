package main

import (
	rq "github.com/levigross/grequests"
	jsp "github.com/buger/jsonparser"
	"fmt"
)

type Token struct {
	ID int64
	Name string
	Symbol string
}

var tokenMap = map[string]Token{}

func getTokenInfo(slug string) Token {
	var symbol string
	base := "https://api.coinmarketcap.com/v2/listings/"
	res, _ := rq.Get(base, nil)
	if len(tokenMap) == 0 {
		jsp.ArrayEach(res.Bytes(), func(value []byte, dataType jsp.ValueType, offset int, err error) {
			id, _ := jsp.GetInt(value, "id")
			symbol, _ = jsp.GetString(value, "symbol")
			name, _ := jsp.GetString(value, "name")
			slug, _ := jsp.GetString(value, "website_slug")
			tokenMap[slug] = Token{ID: id, Name: name, Symbol: symbol}
		}, "data")
	}
	return tokenMap[slug]
}

func getTokenPrice(id int64, conversion string) (float64, float64, float64, float64) {
	base := "https://api.coinmarketcap.com/v2/ticker"
	url := fmt.Sprintf("%s/%d/?convert=%s", base, id, conversion)
	res, _ := rq.Get(url, nil)
	data := res.Bytes()
	price, _ := jsp.GetFloat(data, "data", "quotes", "USD", "price")
	converted, _ := jsp.GetFloat(data, "data", "quotes", conversion, "price")
	pct_price, _ := jsp.GetFloat(data, "data", "quotes", "USD", "percent_change_24h")
	pct_converted, _ := jsp.GetFloat(data, "data", "quotes", "USD", "percent_change_24h")
	return price, converted, pct_price, pct_converted
}
