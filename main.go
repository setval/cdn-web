package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type TemplateFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64 `json:"size"`
	IsDir bool `json:"is_dir"`
}

type TemplateWebPage struct {
	Files      string
	Token      string
	IsMainPage bool
}

func main() {
	address := "127.0.0.1:8080"
	if os.Getenv("ADDRESS") != "" {
		address = os.Getenv("ADDRESS")
	}

	tmplWebPage, err := template.ParseFiles("web.html")
	if err != nil {
		panic(err)
	}

	token := os.Getenv("TOKEN")
	mainDir := os.Getenv("DIR")
	if mainDir == "" {
		mainDir = "static"
	}

	e := echo.New()
	e.HTTPErrorHandler = func(err error, ctx echo.Context) {
		fmt.Println(err)
	}


	e.GET("/favicon.ico", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	e.GET("*", func(c echo.Context) error {
		curPath := path.Clean(c.Param("*"))

		var files []TemplateFile
		var dirs []TemplateFile

		filePath := path.Join(mainDir, curPath)
		filePathLevel := strings.Count(filePath, "/") + 1

		if err := filepath.Walk(mainDir, func(p string, d fs.FileInfo, err error) error {
			if p == mainDir {
				return nil
			}

			if strings.Index(p, filePath) != 0 {
				return nil
			}

			level := strings.Count(p, "/")
			if filePathLevel != level {
				return nil
			}

			t := TemplateFile{
				Name: path.Base(p),
				Path: path.Clean(p[len(mainDir):]),
			}
			if d.IsDir() {
				t.IsDir = true
				dirs = append(dirs, t)
			} else {
				t.Size = d.Size()
				files = append(files, t)
			}

			return err
		}); err != nil {
			return err
		}


		dirs = append(dirs, files...)
		bbb, _ := json.Marshal(dirs)

		var result bytes.Buffer
		if err := tmplWebPage.Execute(&result, TemplateWebPage{
			Files: string(bbb),
			Token: token,
			IsMainPage: curPath == "",
		}); err != nil {
			return err
		}

		return c.HTML(http.StatusOK, result.String())
	})

	e.POST("/api/:type", func(c echo.Context) error {
		if c.Request().Header.Get("token") != token {
			return echo.NewHTTPError(http.StatusForbidden)
		}

		action := c.Param("type")
		p := c.Request().FormValue("path")
		fp := filepath.Join(mainDir, p)


		switch action {
		case "upload":
			file, info, err := c.Request().FormFile("file")
			if err != nil {
				return err
			}
			defer file.Close()
			pathFile := path.Clean(path.Join(fp, info.Filename))
			out, err := os.Create(pathFile)
			if err != nil {
				return err
			}
			defer out.Close()
			if _, err := io.Copy(out, file); err != nil {
				return err
			}
		case "rename":
			n := path.Join(path.Dir(fp), c.Request().FormValue("name"))
			if err := os.Rename(fp, n); err != nil {
				return err
			}
		case "delete":
			if err := os.Remove(path.Join(mainDir, p)); err != nil {
				return err
			}
		case "mkdir":
			if err := os.Mkdir(path.Join(mainDir, p), os.ModePerm); err != nil {
				return err
			}
		}

		return nil
	})

	e.Start(address)
}