package main

import (
	"flag"
	"os"
	"github.com/airdispatch/blog"
	"github.com/airdispatch/go-pressure"
	"airdispat.ch/common"
)

var WORKING_DIRECTORY string
var TEMPLATE_DIRECTORY string
var PORT string

var flag_port = flag.String("port", "", "specify the port that the server should run on")
var blog_address = flag.String("blog_address", "", "specify the address to load the blog from")

func main() {
	flag.Parse()

	temp_port := os.Getenv("PORT")
	if temp_port == "" {
		temp_port = *flag_port
	}

	temp_blog := os.Getenv("BLOG_ADDRESS")
	if temp_blog == "" {
		temp_blog = *blog_address
	}

	theServer := &pressure.Server {
		Port: temp_port,
	}
	theServer.ConfigServer()
	
	webInit(temp_blog)

	defineRoutes(theServer)

	theServer.RunServer()
}

var theBlog *blog.Blog

func webInit(blogAddr string) {
	serverKey, _ := common.CreateADKey()

	theBlog =  &blog.Blog{
		Address: blogAddr,
		Trackers: []string{"mailserver.airdispat.ch:1024", "localhost:1024"},
		Key: serverKey,
		BlogId: "ad",
	}
	theBlog.Initialize()
}

func defineRoutes(s *pressure.Server) {
	s.WebServer.Get("/", s.DisplayTemplate("index.html"))

	blogTemp, err := s.GetTemplateNamed("blog.html")
	if err != nil {
		return
	}

	s.WebServer.Get("/blog(.*)", theBlog.WebGoBlog(blogTemp, "base"))
}