package main

import (
	"github.com/hoisie/web"
	"os"
	"io"
	"flag"
	"fmt"
	"html/template"
	"path/filepath"
	"github.com/airdispatch/blog"
	"airdispat.ch/common"
)

var WORKING_DIRECTORY string
var TEMPLATE_DIRECTORY string
var PARSED_TEMPLATES map[string]template.Template
var PORT string

var flag_port = flag.String("port", "", "specify the port that the server should run on")

func main() {
	defineConstants()
	webInit()
	s := web.NewServer()
	defineRoutes(s)
	loadTemplates(getPath(TEMPLATE_DIRECTORY), "")
	s.Config.StaticDir = WORKING_DIRECTORY + "/static"
	s.Run("0.0.0.0:" + PORT)
}

// START APPLICAITON-SPECIFIC CODE

var theBlog *blog.Blog

func webInit() {
	serverKey, _ := common.CreateKey()

	theBlog =  &blog.Blog{
		Address: "e7da159a65cb19a37c86b56f789e96c410a6a5b74a8a570f",
		Trackers: []string{"localhost:1024"},
		Key: serverKey,
	}
	theBlog.Initialize()
}

func defineRoutes(s *web.Server) {
	s.Get("/", displayTemplate("index.html"))

	blogTemp, _ := PARSED_TEMPLATES["blog.html"]
	s.Get("/blog(.*)", theBlog.WebGoBlog(&blogTemp))
}

// EVERYTHING BELOW THIS LINE IS BOILERPLATE

type TemplateView func(ctx *web.Context)
func displayTemplate(templateName string) TemplateView {
	return func(ctx *web.Context) {
		WriteTemplateToContext(templateName, ctx, nil)
	}
}

func defineConstants() {
	temp_dir := os.Getenv("WORK_DIR")
	if temp_dir == "" {
		temp_dir, _ = os.Getwd()
	}
	WORKING_DIRECTORY = temp_dir

	temp_port := os.Getenv("PORT")
	if temp_port == "" {
		temp_port = *flag_port
	}
	PORT = temp_port

	TEMPLATE_DIRECTORY = "templates"
	PARSED_TEMPLATES = make(map[string]template.Template)
}

func loadTemplates(folder string, append string) {
	// Start looking through the original directory
	dirname := folder + string(filepath.Separator)
	d, err := os.Open(dirname)
	if err != nil {
		fmt.Println("Unable to Read Templates Folder: " + dirname)
		os.Exit(1)
	}
	files, err := d.Readdir(-1)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Loop over all files
	for _, fi := range files {
		if fi.IsDir() {
			// Call yourself if you find more tempaltes
			loadTemplates(dirname + fi.Name(), append + fi.Name() + string(filepath.Separator))
		} else {
			// Parse templates here
			templateName := append + fi.Name()
			parseTemplate(templateName, getPath(TEMPLATE_DIRECTORY + string(filepath.Separator) + templateName))
		}
	}
}

func parseTemplate(templateName string, filename string) {
	tmp, err := template.New(templateName).ParseFiles(filename)
	if err != nil {
		fmt.Println("Unable to parse template " + templateName)
		fmt.Println(err)
		os.Exit(1)
	}

	PARSED_TEMPLATES[templateName] = *tmp
}

func blankResponse() string {
	return WORKING_DIRECTORY
}

func writeHeaders(ctx *web.Context) {
}

func getPath(path string) string {
	return WORKING_DIRECTORY + string(filepath.Separator) + path
}

func WriteTemplateToContext(templatename string, ctx *web.Context, data interface{}) {
	template, ok := PARSED_TEMPLATES[templatename]
	if !ok {
		displayErrorPage(ctx, "Unable to find template. Template: " + templatename)
	}
	err := template.Execute(ctx, data)
	if err != nil {
		fmt.Println(err)
	}
}

func getFileContents(filename string) (*os.File, error) {
	file, err := os.Open(getPath(filename))
	if err != nil {
		return nil, err
	}
	return file, nil
}

func writeFileToContext(filename string, ctx *web.Context) {
	file, err := getFileContents(filename)
	if err != nil {
		displayErrorPage(ctx, "Unable to open file. File: " + getPath(filename))
		return
	}
	_, err = io.Copy(ctx, file)
	if err != io.EOF && err != nil {
		displayErrorPage(ctx, "Unable to Copy into Buffer. File: " + getPath(filename))
		return
	}
}

func displayErrorPage(ctx *web.Context, error string) {
	ctx.WriteString("<!DOCTYPE html><html><head><title>Project Error</title></head>")
	ctx.WriteString("<body><h1>Application Error</h1>")
	ctx.WriteString("<p>" + error + "</p>")
	ctx.WriteString("</body></html>")
}