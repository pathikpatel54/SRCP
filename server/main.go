package main

import server "scrp/server/handlers"

func main() {
	s := server.NewServer()
	s.Listen("8080")
}
