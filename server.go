package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"log"
)

type client struct{} // 必要に応じて、このタイプにデータを追加します。

var clients = make(map[*websocket.Conn]client) // 注：キーとしてポインターのようなタイプ（文字列など）を持つ大きなマップは低速ですが、キーとしてポインター自体を使用することは許容され、高速です
var register = make(chan *websocket.Conn)
var broadcast = make(chan string)
var unregister = make(chan *websocket.Conn)

func runHub()  {
	for {
		select {
		case connection := <-register:
			clients[connection] = client{}
			log.Println("connection registered")

		case message := <-broadcast:
			log.Println("message received:", message)

			// すべてのクライアントにメッセージを送信します
			for connection := range clients {
				if err := connection.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
					log.Println("write error: ", err)

					unregister <- connection
					connection.WriteMessage(websocket.CloseMessage, []byte{})
					connection.Close()
				}
			}
		case connection := <-unregister:
			// ハブからクライアントを削除します
			delete(clients, connection)

			log.Println("connection unregistered")
		}
	}
}

func main()  {
	app := fiber.New()

	app.Static("/", "./templates/index.html")

	app.Use(func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return c.SendStatus(fiber.StatusUpgradeRequired)
	})

	go runHub()

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		// 関数が戻ったら、クライアントの登録を解除して接続を閉じます
		defer func() {
			unregister <- c
			c.Close()
		}()

		// クライアントを登録する
		register <- c

		for {
			messageType, message, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Println("read error: ", err)
				}
				return // 遅延関数を呼び出します。つまり、エラー時に接続を閉じます。
			}

			if messageType == websocket.TextMessage {
				// 受信したメッセージをブロードキャストする
				broadcast <- string(message)
			} else {
				log.Println("websocket message received of type", messageType)
			}
		}
	}))

	app.Listen(":3000")
}