package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/djhworld/go-lambda-invoke/golambdainvoke"
	"github.com/fatih/color"
	"github.com/k0kubun/pp"
)

func main() {
	functionPath := flag.String("path", ".", "The function path")
	eventSourceFile := flag.String("event", "./events/base.json", "The event as json source file")
	rpcPort := flag.Int("port", 8899, "The RPC port")

	// Parse flags
	flag.Parse()

	// Get absolute path for lambda func
	realFunctionPath, err := filepath.Abs(*functionPath)
	if err != nil {
		color.Magenta("!! Invalid path")
		color.Red(err.Error())
		os.Exit(1)
	}

	realEventSourceFile, err := filepath.Abs(*eventSourceFile)
	if err != nil {
		color.Magenta("!! Invalid path")
		color.Red(err.Error())
		os.Exit(1)
	}

	color.Magenta(">> Function root : %s", realFunctionPath)
	color.Magenta(">> Event source file (JSON) : %s", realEventSourceFile)

	// Read event source file
	eventFileData, readFileErr := ioutil.ReadFile(realEventSourceFile)

	if readFileErr != nil {
		color.Magenta("!! Unable to read event source file")
		color.Red(readFileErr.Error())
		os.Exit(1)
	}

	// Parse event json file
	var eventJSON map[string]interface{}
	json.Unmarshal(eventFileData, &eventJSON)

	// Build function via go binary
	goBuildCmd := exec.Command(
		"go",
		"build",
		"-o",
		"/tmp/_tmp_go_testlambdafunc",
		fmt.Sprintf("%s/main.go", realFunctionPath),
	)

	var stdoutStderr []byte
	stdoutStderr, err = goBuildCmd.CombinedOutput()

	if err != nil {
		color.Magenta("!! Unable to build target function")
		color.Red(err.Error())
		color.Magenta("!! STDOUT / STDERR")
		fmt.Println(string(stdoutStderr))
		os.Exit(1)
	}

	color.Magenta(">> Successfully built main.go")

	// Run function
	functionExecCmd := exec.Command("/tmp/_tmp_go_testlambdafunc")
	functionExecCmd.Stdout = os.Stdout
	functionExecCmd.Stderr = os.Stderr
	functionExecCmd.Env = os.Environ()
	functionExecCmd.Env = append(functionExecCmd.Env, fmt.Sprintf("_LAMBDA_SERVER_PORT=%d", *rpcPort))
	if err := functionExecCmd.Start(); err != nil {
		color.Red(err.Error())
		return
	}

	time.Sleep(1 * time.Second)
	color.Magenta(">> Function running")

	rpcResponse, err := golambdainvoke.Run(golambdainvoke.Input{
		Port:    *rpcPort,
		Payload: eventJSON,
	})

	if err != nil {
		color.Magenta("!! Error on rpc call")
		color.Red(err.Error())
		functionExecCmd.Process.Kill()
		os.Exit(1)
	}

	var rpcResponseParsed map[string]interface{}
	json.Unmarshal(rpcResponse, &rpcResponseParsed)

	fmt.Println("")
	pp.Println(rpcResponseParsed)
	fmt.Println("")

	fmt.Println("")
	if rpcResponseParsed["body"] != nil {
		pp.Println(rpcResponseParsed["body"])
	}
	fmt.Println("")

	// Kill function process
	functionExecCmd.Process.Kill()
	color.Magenta(">> Function exited")
}
