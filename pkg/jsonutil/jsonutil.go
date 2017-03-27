package jsonutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/skatteetaten/aoc/pkg/fileutil"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// Struct to represent data to the Boober interface
type booberInferface struct {
	Env         string                     `json:"env"`
	App         string                     `json:"app"`
	Affiliation string                     `json:"affiliation"`
	Files       map[string]json.RawMessage `json:"files"`
	Overrides   map[string]json.RawMessage `json:"overrides"`
}

func GenerateJson(envFile string, envFolder string, folder string, parentFolder string, args []string,
	overrideFiles []string, affiliation string) (jsonStr string, error error) {
	var booberData booberInferface
	var returnMap map[string]json.RawMessage
	var returnMap2 map[string]json.RawMessage
	booberData.App = strings.TrimSuffix(envFile, filepath.Ext(envFile)) //envFile
	booberData.Env = envFolder

	booberData.Affiliation = affiliation

	returnMap, error = Folder2Map(folder, envFolder+"/")
	if error != nil {
		return
	}

	for foo := range returnMap {
		fmt.Println("DEBUG: Returmap: " + string(returnMap[foo]))
	}
	returnMap2, error = Folder2Map(parentFolder, "")
	if error != nil {
		return
	}

	booberData.Files = CombineMaps(returnMap, returnMap2)
	for foo := range booberData.Files {
		fmt.Println("DEBUG: Booberdata.files: " + string(booberData.Files[foo]))
	}
	booberData.Overrides = overrides2map(args, overrideFiles)

	jsonByte, ok := json.Marshal(booberData)
	if !(ok == nil) {
		return "", errors.New(fmt.Sprintf("Internal error in marshalling Boober data: %v\n", ok.Error()))
	}

	jsonStr = string(jsonByte)
	fmt.Println("DEBUG: Return value from GenerateJson: " + jsonStr)
	return
}

func overrides2map(args []string, overrideFiles []string) (returnMap map[string]json.RawMessage) {
	returnMap = make(map[string]json.RawMessage)
	for i := 0; i < len(overrideFiles); i++ {
		returnMap[overrideFiles[i]] = json.RawMessage(args[i+1])
	}
	return
}

func Folder2Map(folder string, prefix string) (map[string]json.RawMessage, error) {
	returnMap := make(map[string]json.RawMessage)
	var allFilesOK bool = true
	var output string
	fmt.Println("DEBUG: Folder2Map called, folder: " + folder + ", prefix: " + prefix)
	files, _ := ioutil.ReadDir(folder)
	var filesProcessed = 0
	for _, f := range files {
		absolutePath := filepath.Join(folder, f.Name())
		fmt.Println("DEBUG: absoultePath: " + absolutePath)
		if fileutil.IsLegalFileFolder(absolutePath) == fileutil.SpecIsFile { // Ignore folders
			matched, _ := filepath.Match("*.json", strings.ToLower(f.Name()))
			if matched {
				fileJson, err := ioutil.ReadFile(absolutePath)
				if err != nil {
					output += fmt.Sprintf("Error in reading file %v\n", absolutePath)
					allFilesOK = false
				} else {
					if IsLegalJson(string(fileJson)) {
						filesProcessed++
						returnMap[prefix+f.Name()] = fileJson
						fmt.Println("DEBUG: File read in Filder2Map: " + f.Name() + ":\n" + string(fileJson))
					} else {
						fmt.Println("DEBUG: Illegal JSON detected in Folder2Map")
						output += fmt.Sprintf("Illegal JSON in configuration file %v\n", absolutePath)
						allFilesOK = false
					}
				}
			}
		}

	}
	if !allFilesOK {
		return nil, errors.New(output)
	}
	return returnMap, nil
}

func CombineMaps(map1 map[string]json.RawMessage, map2 map[string]json.RawMessage) (returnMap map[string]json.RawMessage) {
	returnMap = make(map[string]json.RawMessage)

	for k, v := range map1 {
		returnMap[k] = v
	}
	for k, v := range map2 {
		returnMap[k] = v
	}
	return
}

func IsLegalJson(jsonString string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(jsonString), &js) == nil
}

func PrettyPrintJson(jsonString string) string {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(jsonString), "", "\t")
	if err != nil {
		return jsonString
	}
	return out.String()
}