package main

import "github.com/gin-gonic/gin"

func main()  {
	r := gin.Default()

	r.LoadHTMLGlob("templates/*.html")

	r.GET("/:roomId", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	r.Run(":8080")
}