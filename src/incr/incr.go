package main

//go:generate go-bindata -pkg $GOPACKAGE -o assets.go -prefix ../../ui/build ../../ui/build/

import (
	"log"
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

func incr(c *gin.Context, s Store, gif bool) {
	s.Incr(c.Param("ns"), strings.TrimSuffix(c.Param("counter"), ".gif"))
	if gif {
		c.Data(200, "image/gif", minimalGIF)
	} else {
		c.AbortWithStatus(200)
	}
}

func main() {
	s, err := NewStore(DBPath)
	if err != nil {
		log.Fatal(err)
	}

	r := gin.Default()
	r.Use(corsHandler)
	r.GET("/api/:ns", func(c *gin.Context) {
		if list, err := s.List(c.Param("ns")); err != nil {
			c.AbortWithStatus(500)
		} else {
			c.JSON(200, list)
		}
	})
	r.GET("/api/:ns/:counter", func(c *gin.Context) {
		if strings.HasSuffix(c.Param("counter"), ".gif") {
			incr(c, s, true)
		} else {
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
		}
	})
	r.POST("/api/:ns/:counter", func(c *gin.Context) {
		incr(c, s, false)
	})
	r.NoRoute(func(c *gin.Context) {
		log.Println(c.Request.URL.Path)
		switch c.Request.URL.Path {
		case "/":
			fallthrough
		case "/index.html":
			c.Data(200, "text/html", MustAsset("index.html"))
		case "/bundle.js":
			c.Data(200, "application/javascript", MustAsset("bundle.js"))
		}
	})
	r.Run() // listen and server on 0.0.0.0:8080
}
