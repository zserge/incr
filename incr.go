package main

import (
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

var DBPath = "incr.db"

// curl http://www.google-analytics.com/__utm.gif | xxd -i
var minimalGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0xff,
	0x00, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00,
	0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01, 0x00, 0x3b,
}

func corsHandler(c *gin.Context) {
	c.Writer.Header().Add("Access-Control-Allow-Origin",
		c.Request.Header.Get("Origin"))
	c.Writer.Header().Add("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Add("Access-Control-Allow-Headers",
		c.Request.Header.Get("Access-Control-Request-Headers"))
	c.Writer.Header().Add("Access-Control-Allow-Methods",
		c.Request.Header.Get("Access-Control-Request-Method"))
	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(200)
	} else {
		c.Next()
	}
}

func submit(c *gin.Context, s Store, gif bool) {
	value := strings.TrimSuffix(c.Param("value"), ".gif")
	if n, err := strconv.ParseFloat(value, 32); err != nil {
		c.AbortWithStatus(400)
	} else {
		s.Submit(c.Param("ns"), c.Param("counter"), Number(n))
		if gif {
			c.Data(200, "image/gif", minimalGIF)
		} else {
			c.AbortWithStatus(200)
		}
	}
}

func main() {
	s, err := NewStore(DBPath)
	if err != nil {
		log.Fatal(err)
	}

	r := gin.Default()
	r.Use(corsHandler)
	r.GET("/:ns/:counter/:value", func(c *gin.Context) {
		submit(c, s, true)
	})
	r.POST("/:ns/:counter/:value", func(c *gin.Context) {
		submit(c, s, false)
	})
	r.GET("/:ns", func(c *gin.Context) {
		if list, err := s.List(c.Param("ns")); err != nil {
			c.AbortWithStatus(500)
		} else {
			c.JSON(200, list)
		}
	})
	r.GET("/:ns/:counter", func(c *gin.Context) {
		if counter, err := s.Query(c.Param("ns"), c.Param("counter")); err != nil {
			log.Println(err)
			c.AbortWithStatus(500)
		} else {
			result := gin.H{"now": counter.Atime}
			for i, bucket := range Buckets {
				result[bucket.Name] = counter.Values[i]
			}
			c.JSON(200, result)
		}
	})
	r.Run() // listen and server on 0.0.0.0:8080
}
