package main

import (
	"flag"
	"os"
	"html/template"
	"github.com/airdispatch/blog"
	"github.com/airdispatch/go-pressure"
	"airdispat.ch/common"
)

var WORKING_DIRECTORY string
var TEMPLATE_DIRECTORY string
var PARSED_TEMPLATES map[string]template.Template
var PORT string

var flag_port = flag.String("port", "", "specify the port that the server should run on")

func main() {
	flag.Parse()

	temp_port := os.Getenv("PORT")
	if temp_port == "" {
		temp_port = *flag_port
	}

	theServer := &pressure.Server {
		Port: temp_port,
	}
	theServer.ConfigServer()

	defineRoutes(theServer)

	theServer.RunServer()
}

var theBlog *blog.Blog

func webInit() {
	serverKey, _ := common.CreateADKey()

	theBlog =  &blog.Blog{
		Address: "bb5c57ed27beecd60f659f2a8df68b8a72ccefe96cc12736",
		Trackers: []string{"mailserver.airdispat.ch:1024"},
		Key: serverKey,
	}
	theBlog.Initialize()
}

func defineRoutes(s *pressure.Server) {
	s.WebServer.Get("/", s.DisplayTemplate("index.html"))

	blogTemp, _ := PARSED_TEMPLATES["blog.html"]
	s.WebServer.Get("/blog(.*)", theBlog.WebGoBlog(&blogTemp))
}