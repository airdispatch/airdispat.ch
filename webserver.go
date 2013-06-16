package main

import (
	"github.com/hoisie/web"
	"os"
	"io"
	"flag"
	"fmt"
	"html/template"
	"path/filepath"
	"github.com/russross/blackfriday"
	"airdispat.ch/common"
	"crypto/ecdsa"
	"airdispat.ch/airdispatch"
	clientFramework "airdispat.ch/client/framework"
	"code.google.com/p/goprotobuf/proto"
	"unicode"
	"errors"
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

type Blog struct {
	Address string
	Trackers []string
	Key *ecdsa.PrivateKey

	AllPosts map[string]Post
}

type Post struct {
	Title string
	Author string
	URL string
	Date string
	Content template.HTML
	plainText string
}

var theBlog *Blog

func webInit() {
	serverKey, _ := common.CreateKey()

	theBlog =  &Blog{
		Address: "e7da159a65cb19a37c86b56f789e96c410a6a5b74a8a570f",
		Trackers: []string{"localhost:1024"},
		Key: serverKey,
	}
	theBlog.Initialize()
}

func (b *Blog) Initialize() {
	b.AllPosts = make(map[string]Post)
}

func defineRoutes(s *web.Server) {
	s.Get("/", displayTemplate("index.html"))

	blogTemp, _ := PARSED_TEMPLATES["blog.html"]
	s.Get("/blog(.*)", theBlog.WebGoBlog(&blogTemp))
}

func (b *Blog) GetPost(url string) ([]Post, error) {
	thePost, ok := b.AllPosts[url]
	if !ok {
		return nil, errors.New("Unable to Find Post with that URL")
	}
	return []Post{thePost}, nil
}

func (b *Blog) GetPosts() ([]Post, error) {
	c := clientFramework.Client{}
	c.Populate(b.Key)
	allPosts, err := c.DownloadPublicMail(b.Trackers, b.Address, 0)
	if err != nil {
		return nil, err
	}

	formattedPosts := []Post{}

	for _, value := range(allPosts) {
		byteTypes := value.Data
		dataTypes := &airdispatch.MailData{}

		proto.Unmarshal(byteTypes, dataTypes)

		toFormat := Post{}
		for _, dataObject := range(dataTypes.Payload) {
			if *dataObject.TypeName == "blog/content" {
				toFormat.plainText = string(dataObject.Payload)
			} else if *dataObject.TypeName == "blog/author" {
				toFormat.Author = string(dataObject.Payload)
			} else if *dataObject.TypeName == "blog/date" {
				toFormat.Date = string(dataObject.Payload)
			} else if *dataObject.TypeName == "blog/title" {
				toFormat.Title = string(dataObject.Payload)
			}
		}

		formattedPosts = append(formattedPosts, b.CreatePost(toFormat))
	}

	return formattedPosts, nil
}

func (b *Blog) CreatePost(toFormat Post) Post {
	theContent := template.HTML(string(blackfriday.MarkdownCommon([]byte(toFormat.plainText))))
	thePost := Post{
		Title: toFormat.Title,
		Author: toFormat.Author, 
		URL: Slug(toFormat.Title),
		Date: toFormat.Date,
		Content: theContent}
	b.AllPosts[thePost.URL] = thePost
	return thePost
}

type WebGoRouter func(ctx *web.Context, val string)
func (b *Blog) WebGoBlog(template *template.Template) WebGoRouter {
	return func(ctx *web.Context, val string) {
		var err error
		context := make(map[string]interface{})
		if val == "/" || val == "" {
			context["Posts"], err = b.GetPosts()
		} else {
			context["Posts"], err = b.GetPost(val[1:])
		}
		if err != nil {
			ctx.Write([]byte(err.Error()))
			return
		}
		template.Execute(ctx, context)
		// WriteTemplateToContext("blog.html", ctx, context)
	}
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

var lat = []*unicode.RangeTable{unicode.Letter, unicode.Number}
func Slug(s string) string {
	buf := make([]rune, 0, len(s))
	dash := false
	for _, r := range s {
		switch {
		case unicode.IsOneOf(lat, r):
			buf = append(buf, unicode.ToLower(r))
			dash = true
		case dash:
			if dash {
				buf = append(buf, '-')
				dash = false
			}
		}
	}
	return string(buf)
}
