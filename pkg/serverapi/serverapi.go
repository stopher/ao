package serverapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"time"

	"github.com/pkg/errors"
	"github.com/skatteetaten/ao/pkg/configuration"
	"github.com/skatteetaten/ao/pkg/jsonutil"
	"github.com/skatteetaten/ao/pkg/openshift"
)

const badRequestString = "Internal error: Bad request"

// Structs to represent return data from the API interface

type ApplicationId struct {
	EnvironmentName string `json:"environmentName"`
	ApplicationName string `json:"applicationName"`
}

type OpenShiftResponse struct {
	Kind          string `json:"kind"`
	OperationType string `json:"operationType"` // CREATED, UPDATE eller NONE
	Payload       struct {
		Kind string `json:"payload"`
	} `json:"payload"`
	ResponseBody json.RawMessage `json:"responseBody"`
}

type DeploymentDescriptor struct {
	TemplateFile string            `json:"templateFile"`
	Template     string            `json:"template"`
	Parameters   map[string]string `json:"parameters"`
}

type Field struct {
	Path   string `json:"path"`
	Value  string `json:"value"`
	Source string `json:"source"`
}

type AuroraDeploymentConfig struct {
	SchemaVersion string            `json:"schemaVersion"`
	Affiliation   string            `json:"affiliation"`
	Cluster       string            `json:"cluster"`
	Type          string            `json:"type"`
	Name          string            `json:"name"`
	EnvName       string            `json:"envName"`
	Permissions   PermissionsStruct `json:"permissions"`
	Fields        Field             `json:"field"`
	Config        map[string]string `json:"config"`
	GroupId       string            `json:"groupId"`
	ArtifactId    string            `json:"artifactId"`
	Version       string            `json:"version"`
	//Route                bool                 `json:"route"`
	DeploymentStrategy   string               `json:"deploymentStrategy"`
	DeploymentDescriptor DeploymentDescriptor `json:"deploymentDescriptor"`
}

type ApplicationResult struct {
	ApplicationId     ApplicationId          `json:"applicationId"`
	AuroraDc          AuroraDeploymentConfig `json:"auroraDc"`
	OpenShiftResponse OpenShiftResponse      `json:"openShiftResponse"`
}

type Response struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Items   []json.RawMessage `json:"items"`
	Count   int               `json:"count"`
}

type ResponseItemError struct {
	Application string `json:"application"`
	Environment string `json:"environment"`
	Messages    []struct {
		Type    string `json:"type"`
		Message string `json:"message"`
		Field   struct {
			Path   string `json:"path"`
			Value  string `json:"value"`
			Source string `json:"source"`
		} `json:"field"`
	} `json:"messages"`
}

type AuroraConfig struct {
	Files    map[string]json.RawMessage `json:"files"`
	Versions map[string]string          `json:"versions"`
}

type PingResult struct {
	Items []struct {
		Result struct {
			Status     string `json:"status"`
			Dnsname    string `json:"dnsname"`
			ResolvedIp string `json:"resolvedIp"`
			Port       string `json:"port"`
			Message    string `json:"message"`
		} `json:"result"`
		PodIp    string `json:"podIp"`
		HostIp   string `json:"hostIp"`
		HostName string
	} `json:"items"`
}

type PermissionsStruct struct {
	Groups []string `json:"groups,omitempty"`
	Users  []string `json:"users,omitempty"`
}

const apiNotInstalledResponse = "Application is not available"
const localhostAddress = "localhost"
const localhostPort = "8080"

func ParsePingResult(responseString string) (PingResult PingResult, err error) {
	var responseData []byte
	responseData = []byte(responseString)
	err = json.Unmarshal(responseData, &PingResult)
	if err != nil {
		return
	}

	return
}

func ParseResponse(responseString string) (response Response, err error) {
	var responseData []byte
	responseData = []byte(responseString)
	err = json.Unmarshal(responseData, &response)
	if err != nil {
		return
	}

	return
}

func ResponseItems2ApplicationResults(response Response) (applicationResults []ApplicationResult, err error) {
	applicationResults = make([]ApplicationResult, len(response.Items))

	for item := range response.Items {
		err = json.Unmarshal([]byte(response.Items[item]), &applicationResults[item])
		if err != nil {
			return
		}
	}
	return
}

func ResponseItems2AuroraConfig(response Response) (auroraConfig AuroraConfig, err error) {

	if response.Count > 1 {
		err = errors.New("Internal error: Multiple items not supported in AO")
		return
	}
	for item := range response.Items {
		//ds := string(response.Items[item])
		//fmt.Println(ds)
		err = json.Unmarshal([]byte(response.Items[item]), &auroraConfig)
		if err != nil {
			return
		}
	}
	return
}

func ResponseItems2Vaults(response Response) (output string, err error) {
	var newline string = ""
	for item := range response.Items {
		output += newline + jsonutil.PrettyPrintJson(string(response.Items[item]))
		newline = "\n"
	}
	return
}

func ApplicationResult2MessageString(applicationResult ApplicationResult) (output string, err error) {

	output +=
		//applicationResult.ApplicationId.ApplicationName +
		applicationResult.AuroraDc.GroupId + "/" + applicationResult.AuroraDc.ArtifactId + "-" + applicationResult.AuroraDc.Version +
			" deployed in " + applicationResult.AuroraDc.Cluster + "/" + applicationResult.AuroraDc.EnvName
	return
}

func ResponsItems2MessageString(response Response) (output string, err error) {
	if response.Message != "" {
		output = response.Message
	}
	for item := range response.Items {
		var responseItemError ResponseItemError
		err = json.Unmarshal([]byte(response.Items[item]), &responseItemError)
		if err != nil {
			return "", err
		}
		output = output + "\n\t Env/App: " + responseItemError.Environment + "/" + responseItemError.Application + ":"

		for message := range responseItemError.Messages {
			if responseItemError.Messages[message].Field.Path != "" {
				output = output + "\n\t\t" + responseItemError.Messages[message].Field.Path + " (" +
					responseItemError.Messages[message].Field.Value + ") in " + responseItemError.Messages[message].Field.Source
			}
			if responseItemError.Messages[message].Field.Source != "" {
				output = output + "\n\t\t File: " + responseItemError.Messages[message].Field.Source
			}
			output = output + "\n\t\t\t" + responseItemError.Messages[message].Message
		}
	}
	return
}

func getConsoleAddress(clusterName string) (consoleAddress string) {
	consoleAddress = "http://console-aurora." + clusterName + ".paas.skead.no"
	return
}

func CallConsole(apiEndpoint string, arguments string, verbose bool, debug bool, openshiftConfig *openshift.OpenshiftConfig) (result json.RawMessage, err error) {
	apiCluster, err := openshiftConfig.GetApiCluster()
	consoleAddress := getConsoleAddress(apiCluster.Name)
	token := apiCluster.Token

	url := consoleAddress + "/public/" + apiEndpoint
	if arguments != "" {
		url += "?" + arguments
	}
	if debug {
		fmt.Print("Sending request to Console at " + url + "...")
	}
	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		if verbose {
			fmt.Println("FAIL.  Error connecting to Console service")
		}
		err = errors.New(fmt.Sprintf("Error connecting to the Console service on %v: %v", url, err))
		return
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	output := string(body)

	if resp.StatusCode == http.StatusOK {
		if debug {
			fmt.Println("OK")
		}
	} else {
		if debug {
			fmt.Println("ERROR: " + resp.Status)
		}
		if resp.StatusCode == http.StatusGatewayTimeout {
			return nil, errors.New("Ping request timed out")
		} else {
			return nil, errors.New(resp.Status)
		}
	}

	if debug {
		fmt.Println("Response status: " + strconv.Itoa(resp.StatusCode))
		if jsonutil.IsLegalJson(output) {
			fmt.Println(jsonutil.PrettyPrintJson(output))
		} else {
			fmt.Println(output)
		}

	}
	result = json.RawMessage(output)
	return
}

// Check for valid login, that is we have a configuration with at least one reachable cluster
func ValidateLogin(openshiftConfig *openshift.OpenshiftConfig) (output bool) {
	var openshiftCluster *openshift.OpenshiftCluster
	openshiftCluster, _ = openshiftConfig.GetApiCluster()
	if openshiftCluster != nil {
		if !openshiftCluster.HasValidToken() {
			return false
		}
	}
	return true
}

func getApiClusterAddress(configuration *configuration.ConfigurationClass) (clusterAddress string, err error) {
	for i := range configuration.OpenshiftConfig.Clusters {
		if configuration.OpenshiftConfig.Clusters[i].Reachable {
			if configuration.OpenshiftConfig.Clusters[i].Name == configuration.OpenshiftConfig.APICluster {
				if configuration.OpenshiftConfig.Clusters[i].BooberUrl == "" {
					err = errors.New("Boober URL is not configured, please log in again")
					return "", err
				}
				return configuration.OpenshiftConfig.Clusters[i].BooberUrl, nil
			}
		}
	}
	err = errors.New("No Boober API defined.")
	return "", err
}

func CallApi(httpMethod string, apiEndpoint string, combindedJson string,
	configuration *configuration.ConfigurationClass) (response Response, err error) {
	var headers = make(map[string]string)
	return CallApiWithHeaders(headers, httpMethod, apiEndpoint, combindedJson, configuration)
}

// Call the API Boober instance
func CallApiWithHeaders(headers map[string]string, httpMethod string, apiEndpoint string, combindedJson string,
	configuration *configuration.ConfigurationClass) (response Response, err error) {
	var apiCluster *openshift.OpenshiftCluster

	var token string
	var apiAddress string

	apiCluster, err = configuration.OpenshiftConfig.GetApiCluster()
	if configuration.PersistentOptions.Token == "" {
		if apiCluster != nil {
			token = apiCluster.Token
		}
	} else {
		token = configuration.PersistentOptions.Token
	}

	// TODO: Simplify, the apiAddress is the only difference between the two branches of the main if.
	if configuration.PersistentOptions.Localhost || configuration.OpenshiftConfig.Localhost {
		apiAddress = "http://" + localhostAddress + ":" + localhostPort
	} else {
		apiAddress, err = getApiClusterAddress(configuration)
		if err != nil {
			return response, err
		}
	}

	output, err := callApiInstance(headers, httpMethod, combindedJson, configuration.PersistentOptions.Verbose,
		apiAddress+apiEndpoint,
		token, configuration.PersistentOptions.DryRun, configuration.PersistentOptions.Debug)
	if err != nil {
		response, err = ParseResponse(output)
		return response, err
	}
	response, err = ParseResponse(output)
	return response, err

}

// Call all reachable Boober instances
func CallDeployWithHeaders(headers map[string]string, httpMethod string, apiEndpoint string, combindedJson string,
	localhost bool, verbose bool, openshiftConfig *openshift.OpenshiftConfig, dryRun bool, debug bool,
	apiAddress string, token string, clusterOnly string, apiClusterOnly bool) ([]Response, error) {

	if localhost || openshiftConfig.Localhost {
		apiAddress = "http://" + localhostAddress + ":" + localhostPort

		apiCluster, err := openshiftConfig.GetApiCluster()
		if token == "" && apiCluster != nil {
			token = apiCluster.Token
		}
		output, err := callApiInstance(headers, httpMethod, combindedJson, verbose, apiAddress+apiEndpoint, token, dryRun, debug)
		if err != nil {
			return nil, err
		}
		response, err := ParseResponse(output)

		return []Response{response}, nil
	}

	resp := make(chan *Response)
	defer close(resp)
	callCount := 0
	for _, cluster := range openshiftConfig.Clusters {
		if cluster.Reachable && cluster.BooberUrl != "" &&
			(cluster.Name == clusterOnly || clusterOnly == "") &&
			(cluster.Name == openshiftConfig.APICluster || apiClusterOnly == false) {
			instanceToken := token
			if instanceToken == "" {
				instanceToken = cluster.Token
			}
			callCount++
			url := cluster.BooberUrl + apiEndpoint
			go func() {
				output, err := callApiInstance(headers, httpMethod, combindedJson, verbose, url, instanceToken, dryRun, debug)
				if err != nil && verbose {
					fmt.Println(err)
				}
				response, _ := ParseResponse(string(output))
				resp <- &response
			}()
		}
	}

	responses := []Response{}
	for i := 0; i < callCount; i++ {
		r := <-resp
		responses = append(responses, *r)
	}

	return responses, nil
}

func makeResponse(message string, success bool) (responseStr string, err error) {
	var response Response

	response.Message = message
	response.Success = success
	response.Count = 0
	response.Items = make([]json.RawMessage, 0)

	responseBytes, err := json.Marshal(response)
	responseStr = string(responseBytes)

	err = errors.New(message)

	return responseStr, err
}

func callApiInstance(headers map[string]string, httpMethod string, combindedJson string, verbose bool, url string, token string, dryRun bool, debug bool) (output string, err error) {
	if verbose {
		var infoString string
		if httpMethod == http.MethodPut {
			infoString = "Sending config to"
		} else {
			infoString = "Getting config from"
		}
		fmt.Println(infoString + " Boober at " + url + "... ")
	}

	if debug {
		fmt.Println("REQUEST:")
		fmt.Print("\t" + httpMethod)
		fmt.Println(" URL: " + url)
		fmt.Println("\tToken: " + token)
		if combindedJson == "" {
			fmt.Println("\tNo JSON Payload")
		} else {
			fmt.Println("\tJSON Payload: \n" + jsonutil.PrettyPrintJson(combindedJson))
		}

	}
	var jsonStr = []byte(combindedJson)

	req, err := http.NewRequest(httpMethod, url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return "", errors.New(fmt.Sprintf("Internal error in NewRequest: %v", err))
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Add("dryrun", fmt.Sprintf("%v", dryRun))

	for header := range headers {
		req.Header.Add(header, headers[header])
		if debug {
			fmt.Println("Header: " + header + ", value: " + headers[header])
		}
	}

	client := &http.Client{}

	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		if verbose {
			fmt.Println("FAIL.  Error connecting to Boober service")
		}
		errorstring := fmt.Sprintf("Error connecting to the Boober service on %v: %v", url, err)
		return makeResponse(errorstring, false)
	}
	requestTime := time.Since(startTime)

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	output = string(body)

	if debug {
		fmt.Println("RESPONSE:")

		if jsonutil.IsLegalJson(output) {
			fmt.Println(jsonutil.PrettyPrintJson(output))
		} else {
			fmt.Println(output)
		}
		fmt.Println("\tResponse status: " + strconv.Itoa(resp.StatusCode))
		fmt.Println("\tResponse time: " + strconv.FormatFloat(requestTime.Seconds(), 'f', 2, 64) + " sec")
	}

	if jsonutil.IsLegalJson(output) {
		response, err := ParseResponse(output)
		if err != nil {
			// Legal JSON, but not a legal Response struct.  Should not happen, but handle it anyway
			return makeResponse("Internal error: Boober return not a valid response", false)
		}
		if !response.Success {
			// Something went wrong, set the error flag with the message
			err = errors.New(response.Message)
			return output, err
		}
	} else {
		// We got some non-json, return an error
		var errorstring string
		if strings.Contains(output, apiNotInstalledResponse) {
			errorstring = fmt.Sprintf("Error: Boober not available on %v", url)
		} else {
			errorstring = fmt.Sprintf("Internal error on %v: %v", url, output)
		}
		if verbose {
			fmt.Println(errorstring)
		}
		return makeResponse(errorstring, false)
	}

	return
}
