package main

import (
	"flag"
	"github.com/airdispatch/go-pressure"
	"os"
	"path/filepath"
)

// var WORKING_DIRECTORY string
// var TEMPLATE_DIRECTORY string
var PORT string

var flag_port = flag.String("port", "", "specify the port that the server should run on")
var blog_address = flag.String("blog_address", "", "specify the address to load the blog from")
var debug = flag.Bool("debug", false, "specify whether you want to debug the program or not")

func main() {
	flag.Parse()

	temp_port := ":" + os.Getenv("PORT")
	if temp_port == "" {
		temp_port = *flag_port
	}

	temp_wd, _ := os.Getwd()

	// Get Relevant Paths
	template_dir := filepath.Join(temp_wd, "templates")
	static_dir := filepath.Join(temp_wd, "static")

	// Create Server and Necessary Engines
	theServer := pressure.CreateServer(temp_port, *debug)
	tEng := theServer.CreateTemplateEngine(template_dir, "base.html")

	// Register Golang Import URLs
	theServer.RegisterURL(
		pressure.NewURLRoute("^/airdispatch$", &GolangFetchController{"airdispatch-protocol"}),
		pressure.NewURLRoute("^/common$", &GolangFetchController{"airdispatch-protocol"}),
		pressure.NewURLRoute("^/server$", &GolangFetchController{"airdispatch-protocol"}),
		pressure.NewURLRoute("^/client$", &GolangFetchController{"airdispatch-protocol"}),
		pressure.NewURLRoute("^/tracker$", &GolangFetchController{"airdispatch-protocol"}),
		pressure.NewURLRoute("^/server/framework$", &GolangFetchController{"airdispatch-protocol"}),
		pressure.NewURLRoute("^/client/framework$", &GolangFetchController{"airdispatch-protocol"}),
		pressure.NewURLRoute("^/tracker/framework$", &GolangFetchController{"airdispatch-protocol"}),
	)

	// Register URLs
	theServer.RegisterURL(
		pressure.NewURLRoute("^/project/airdispatch", &ProjectController{tEng}),
		pressure.NewURLRoute("^/$", &HomepageController{tEng}),
		pressure.NewStaticFileRoute("^/static/", static_dir),
	)

	// Start the Server
	theServer.RunServer()
}

// Define Custom Controllers

type ProjectController struct {
	tEng *pressure.TemplateEngine
}

func (c *ProjectController) GetResponse(p *pressure.Request, l *pressure.Logger) (pressure.View, *pressure.HTTPError) {
	return c.tEng.NewTemplateView("projects/project.html", nil), nil
}

type GolangFetchController struct {
	packageName string
}

func (c *GolangFetchController) GetResponse(p *pressure.Request, l *pressure.Logger) (pressure.View, *pressure.HTTPError) {
	return pressure.NewHTMLView(
		`<html>
			<head>
				<meta name="go-import" content="airdispat.ch git https://github.com/airdispatch/` + c.packageName + `">
			</head>
		</html>`), nil
}

type HomepageController struct {
	tEng *pressure.TemplateEngine
}

func (c *HomepageController) GetResponse(p *pressure.Request, l *pressure.Logger) (pressure.View, *pressure.HTTPError) {
	return c.tEng.NewTemplateView("index.html", nil), nil
}
