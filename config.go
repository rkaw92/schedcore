package main

import (
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DB_URL        url.URL
	BROKER_URL    url.URL
	TOTAL_USHARDS int16
	MY_USHARDS    []int16
}

func NewConfigFromEnv() Config {
	c := Config{}
	dbUrlRaw := os.Getenv("DB_URL")
	dbUrl, err := url.Parse(dbUrlRaw)
	if err == nil {
		c.DB_URL = *dbUrl
	}
	brokerUrlRaw := os.Getenv("BROKER_URL")
	brokerUrl, err := url.Parse(brokerUrlRaw)
	if err == nil {
		c.BROKER_URL = *brokerUrl
	}
	totalUshards, err := strconv.ParseUint(os.Getenv("TOTAL_USHARDS"), 10, 15)
	if err == nil {
		c.TOTAL_USHARDS = int16(totalUshards)
	}
	myUshardsRaw := strings.Split(os.Getenv("MY_USHARDS"), "-")
	if len(myUshardsRaw) == 2 {
		start, err1 := strconv.ParseUint(myUshardsRaw[0], 10, 15)
		end, err2 := strconv.ParseUint(myUshardsRaw[1], 10, 15)
		if err1 == nil && err2 == nil {
			for i := start; i <= end; i += 1 {
				c.MY_USHARDS = append(c.MY_USHARDS, int16(i))
			}
		}
	}
	return c
}
