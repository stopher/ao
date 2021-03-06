package fileutil

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const SpecIllegal = -1
const SpecIsFile = 1
const SpecIsFolder = 2

func IsLegalFileFolder(filespec string) int {
	var err error
	var absolutePath string
	var fi os.FileInfo

	absolutePath, err = filepath.Abs(filespec)
	fi, err = os.Stat(absolutePath)
	if os.IsNotExist(err) {
		return SpecIllegal
	} else {
		switch mode := fi.Mode(); {
		case mode.IsDir():
			return SpecIsFolder
		case mode.IsRegular():
			return SpecIsFile
		}
	}
	return SpecIllegal
}

func CreateTempFile(content string) (filename string, err error) {
	const tmpFilePrefix = ".ao_edit_file_"
	var tmpDir = os.TempDir()
	tmpFile, err := ioutil.TempFile(tmpDir, tmpFilePrefix)
	if err != nil {
		return "", errors.New("Unable to create temporary file: " + err.Error())
	}
	if IsLegalFileFolder(tmpFile.Name()) != SpecIsFile {
		err = errors.New("Internal error: Illegal temp file name: " + tmpFile.Name())
	}
	filename = tmpFile.Name()
	err = ioutil.WriteFile(tmpFile.Name(), []byte(content), 0700)
	if err != nil {
		return
	}
	return
}

func IsFolderEmpty(filespec string) (emptyFolder bool, err error) {
	var absolutePath string
	absolutePath, err = filepath.Abs(filespec)
	if err != nil {
		return false, err
	}

	dir, err := ioutil.ReadDir(absolutePath)
	if err != nil {
		return false, err
	}
	return len(dir) == 0, err
}

func ValidateFileFolderArg(args []string) (error error) {
	var errorString string

	if len(args) == 0 {
		errorString += "Missing file/folder "
	} else {
		// Chceck argument 0 for legal file / folder
		validateCode := IsLegalFileFolder(args[0])
		if validateCode < 0 {
			errorString += fmt.Sprintf("Illegal file / folder: %v\n", args[0])
		}

	}

	if errorString != "" {
		return errors.New(errorString)
	}
	return
}

func EditFile(filename string) (err error) {
	const vi = "vim"
	var editor = os.Getenv("EDITOR")
	var editorParts []string
	if editor == "" {
		editor = vi
	}
	editorParts = strings.Split(editor, " ")
	editorPath := editorParts[0]

	path, err := exec.LookPath(editorPath)
	if err != nil {
		return errors.New("ERROR: Editor \"" + editorPath + "\" specified in environment variable $EDITOR is not a valid program")
	}

	editorParts[0] = path

	var cmd *exec.Cmd
	cmd = new(exec.Cmd)
	cmd.Path = path
	cmd.Args = append(editorParts, filename)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return
	}
	err = cmd.Wait()
	return
}

func WriteFile(folder string, filename string, content string) (err error) {
	if IsLegalFileFolder(folder) != SpecIsFolder {
		err = errors.New("Illegal folder")
		return err
	}

	absolutePath := filepath.Join(folder, filename)
	parentFolder := filepath.Dir(absolutePath)
	if IsLegalFileFolder(parentFolder) != SpecIsFolder {
		err = os.MkdirAll(parentFolder, 0755)
		if err != nil {
			return err
		}
	}

	err = ioutil.WriteFile(absolutePath, []byte(content), 0750)
	if err != nil {
		return err
	}
	return
}

func repeatString(str string, n int) (output string) {
	for i := 0; i < n; i++ {
		output += str
	}
	return
}

func RightPad(str string, length int) (output string) {
	const pad = " "
	if len(str) >= length {
		output = str[:length]
	} else {
		output = str + repeatString(pad, length-len(str))
	}

	return
}
