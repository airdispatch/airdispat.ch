package main

import (
	"github.com/hoisie/web"
	"os"
	"io"
	"flag"
	"fmt"
	"net"
	"html/template"
	"path/filepath"
	"github.com/russross/blackfriday"
	"airdispat.ch/common"
	"crypto/ecdsa"
	"airdispat.ch/airdispatch"
	"code.google.com/p/goprotobuf/proto"
	"unicode"
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

type Post struct {
	Title string
	Author string
	URL string
	Date string
	Content template.HTML
	plainText string
}
var all_posts map[string]Post
var serverKey *ecdsa.PrivateKey

var trackerLocation = []string{"localhost:1024"}
var adAddress = "e7da159a65cb19a37c86b56f789e96c410a6a5b74a8a570f"

func webInit() {
	all_posts = make(map[string]Post)
	serverKey, _ = common.CreateKey()
}

func defineRoutes(s *web.Server) {
	s.Get("/blog(.*)", blog)
}

func getPost(url string, ctx *web.Context) []Post {
	thePost, ok := all_posts[url]
	if !ok {
		return nil
	}
	return []Post{thePost}
}

func getPosts() []Post {
	mailserver := common.LookupLocation(adAddress, trackerLocation, serverKey)
	allPosts := retrievePublicMessages(adAddress, serverKey, mailserver)

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

		formattedPosts = append(formattedPosts, CreatePost(toFormat))
	}

	return formattedPosts
}

func connectToServer(remote string) net.Conn {
	address, _ := net.ResolveTCPAddr("tcp", remote)

	// Connect to the Remote Mail Server
	conn, err := net.DialTCP("tcp", nil, address)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Cannot connect to server.")
		return nil
	}
	return conn
}

func retrievePublicMessages(toCheck string, key *ecdsa.PrivateKey, recipientServer string) []*airdispatch.Mail {
	recipientConn := connectToServer(recipientServer)
	since := uint64(0)
	// Create the Request Object
	messageRequest := &airdispatch.RetrieveData {
		RetrievalType: common.RETRIEVAL_TYPE_PUBLIC(),
		FromAddress: &toCheck,
		SinceDate: &since,
	}
	requestData, _ := proto.Marshal(messageRequest)
	sendData := common.CreateAirdispatchMessage(requestData, key, common.RETRIEVAL_MESSAGE)

	// Send the Request to the Server
	recipientConn.Write(sendData)

	// Read the Message Response
	data, messageType, _, err := common.ReadSignedMessage(recipientConn)
	if err != nil {
		return nil
	}

	// Ensure that we have been given an array of values
	if messageType == common.ARRAY_MESSAGE {
		// Get the array from the data
		theArray := &airdispatch.ArrayedData{}
		proto.Unmarshal(data, theArray)

		// Find the number of messsages
		mesNumber := theArray.NumberOfMessages

		output := []*airdispatch.Mail{}

		// Loop over this number
		for i := uint32(0); i < *mesNumber; i++ {
			// Get the message and unmarshal it
			mesData, _, _, _ := common.ReadSignedMessage(recipientConn)
			theMessage := &airdispatch.Mail{}
			proto.Unmarshal(mesData, theMessage)

			// Print the Message
			output = append(output, theMessage)
		}

		return output
	}
	return nil
}

func loadDummyData() []Post {
	return []Post{
		CreatePost(Post{"About this Blog", "Hunter Leath", "about-this-blog", "August 5", "hello, world.", ""}),
		CreatePost(Post{"The Airdispatch Experiment", "Hunter Leath", "the-ad-expiriment", "August 1", " - Eat \n - Pray \n - Love", ""}),
	}
}

func CreatePost(toFormat Post) Post {
	theContent := template.HTML(string(blackfriday.MarkdownCommon([]byte(toFormat.plainText))))
	thePost := Post{
		Title: toFormat.Title,
		Author: toFormat.Author, 
		URL: Slug(toFormat.Title),
		Date: toFormat.Date,
		Content: theContent}
	all_posts[thePost.URL] = thePost
	return thePost
}

func blog(ctx *web.Context, val string) {
	context := make(map[string]interface{})
	if val == "/" || val == "" {
		context["Posts"] = getPosts()
	} else {
		context["Posts"] = getPost(val[1:], ctx)
	}
	WriteTemplateToContext("blog.html", ctx, context)
}

// EVERYTHING BELOW THIS LINE IS BOILERPLATE

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
