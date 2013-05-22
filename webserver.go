// +build heroku

package main

import (
	"github.com/hoisie/web"
	"os"
	"io"
	"flag"
	"fmt"
)

var WORKING_DIRECTORY string
var PORT string

var flag_port = flag.String("port", "", "specify the port that the server should run on")

func main() {
	defineConstants()
	s := web.NewServer()
	defineRoutes(s)
	s.Config.StaticDir = WORKING_DIRECTORY + "/static"
	s.Run("0.0.0.0:" + PORT)
}

func defineConstants() {
	temp_dir := os.Getenv("WORK_DIR")
	if temp_dir == "" {
		temp_dir, _ = os.Getwd()
	}
	WORKING_DIRECTORY = temp_dir

	temp_port := os.Getenv("PORT")
	fmt.Println("temp port:", temp_port)
	if temp_port == "" {
		temp_port = *flag_port
	}
	PORT = temp_port
}

func defineRoutes(s *web.Server) {
}

func blankResponse() string {
	return WORKING_DIRECTORY
}

func writeHeaders(ctx *web.Context) {
}

func writeFileToContext(filename string, ctx *web.Context) {
	file, err := os.Open(WORKING_DIRECTORY + "/" + filename)
	if err != nil {
		displayErrorPage(ctx, "Unable to Open: " + WORKING_DIRECTORY + "/" + filename)
		return
	}

	_, err = io.Copy(ctx, file)
	if err != io.EOF && err != nil {
		displayErrorPage(ctx, "Unable to Copy into Buffer. File: " + WORKING_DIRECTORY + "/" + filename)
		return
	}
}

func displayErrorPage(ctx *web.Context, error string) {
	ctx.WriteString("<!DOCTYPE html><html><head><title>Project Error</title></head>")
	ctx.WriteString("<body><h1>Application Error</h1>")
	ctx.WriteString("<p>" + error + "</p>")
	ctx.WriteString("</body></html>")
}
