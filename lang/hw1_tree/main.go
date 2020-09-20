package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

func printDirTree(out io.Writer, path string, printFiles bool, space string) error {
	files, err := ioutil.ReadDir(path)
	if !printFiles {
		all := files
		files = make([]os.FileInfo, 0)
		for _, file := range all {
			if file.IsDir() {
				files = append(files, file)
			}
		}
	}
	for i, file := range files {

		var sizeText string
		if file.IsDir() {
			sizeText = ""
		} else if file.Size() == 0 {
			sizeText = " (empty)"
		} else {
			sizeText = fmt.Sprintf(" (%vb)", file.Size())
		}

		var addSpace string
		if i == len(files)-1 {
			addSpace = "\t"
			fmt.Fprintf(out, "%v└───%v%v\n", space, file.Name(), sizeText)
		} else {
			addSpace = "│\t"
			fmt.Fprintf(out, "%v├───%v%v\n", space, file.Name(), sizeText)
		}

		if file.IsDir() {
			newPath := path + string(os.PathSeparator) + file.Name()
			newSpace := space + addSpace
			printDirTree(out, newPath, printFiles, newSpace)
		}
	}
	return err
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	return printDirTree(out, path, printFiles, "")
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
