package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"
	"github.com/gofiber/websocket/v2"
	"log"
)

type message struct {
	data []byte
	room string
}

type subscription struct {
	conn *websocket.Conn
	room string
}

type hub struct {
	rooms map[string]map[*websocket.Conn]bool
	broadcast chan message
	register chan subscription
	unregister chan subscription
}

var h = hub {
	broadcast: make(chan message),
	register: make(chan subscription),
	unregister: make(chan subscription),
	rooms: make(map[string]map[*websocket.Conn]bool),
}

func (h *hub) run() {
	for {
		select {
		case s := <-h.register:
			connections := h.rooms[s.room]
			if connections == nil {
				connections = make(map[*websocket.Conn]bool)
				h.rooms[s.room] = connections
			}
			h.rooms[s.room][s.conn] = true
			log.Println("connection registered")
		case s := <-h.unregister:
			connections := h.rooms[s.room]
			if connections != nil {
				if _, ok := connections[s.conn]; ok {
					delete(connections, s.conn)
					log.Println("connection unregistered")
				}
			}
		case m := <-h.broadcast:
			log.Println("message received:", m)
			connections := h.rooms[m.room]

			// すべてのクライアントにメッセージを送信します
			for connection := range connections {
				if err := connection.WriteMessage(websocket.TextMessage, []byte(m.data)); err != nil {
					log.Println("write error:", err)

					delete(connections, connection)
					if len(connections) == 0 {
						delete(h.rooms, m.room)
					}
				}
			}
		}
	}
}

func main()  {
	go h.run()

	engine := html.New("./templates", ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Get("/:id", func(c *fiber.Ctx) error {
		return c.Render("index", nil)
	})

	socketapp := app.Group("/ws")

	socketapp.Use(func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return c.SendStatus(fiber.StatusUpgradeRequired)
	})

	socketapp.Get("/:roomId", websocket.New(func(c *websocket.Conn) {
		roomId := c.Params("roomId")

		s := subscription{c, roomId}

		defer func() {
			h.unregister <- s
			s.conn.Close()
		}()

		// クライアントを登録
		h.register <- s

		for {
			messageType, msg, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Println("read error: ", err)
				}
				return // 遅延関数を呼び出します。つまり、エラー時に接続を閉じます
			}

			if messageType == websocket.TextMessage {
				// 受信したメッセージをブロードキャストする
				m := message{msg, s.room}
				h.broadcast <- m
			} else {
				log.Println("websocket message received of type", messageType)
			}
		}
	}))

	app.Listen(":3000")
}