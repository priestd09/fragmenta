// Command line tool for fragmenta which can be used to build and run websites
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

// FIXME - move all instances of hardcoded paths out into optional app config variables
// Ideally we don't care about project structure apart from the load the fragmenta.json file

const (
	// The version of this tool
	fragmentaVersion = "1.1"

	// Used for outputting console messages
	fragmentaDivider = "\n------\n"
)

var (

	// The development config from fragmenta.json
	ConfigDevelopment map[string]string

	// The development config from fragmenta.json
	ConfigProduction map[string]string

	// The app test config from fragmenta.json
	ConfigTest map[string]string
)

// serverName returns the name of our server file - TODO:read from config
func serverName() string {
	return "fragmenta-server" // for now, should use configs
}

func localServerPath(projectPath string) string {
	return fmt.Sprintf("%s/bin/%s-local", projectPath, serverName())
}

func serverPath(projectPath string) string {
	return fmt.Sprintf("%s/bin/%s", projectPath, serverName())
}

func serverCompilePath(projectPath string) string {
	// When older app converted, change to:
	// return path.Join(projectPath, "server.go")

	_, err := os.Stat(path.Join(projectPath, "server.go"))
	if err != nil {

		// Check for old style app path (no server.go in root)
		return projectPath + "/src/app"
	}

	return projectPath
}

// Return the src to scan assets for compilation
// Can this be set within the project setup instead to avoid hardcoding here?
func srcPath(projectPath string) string {
	return projectPath + "src"
}

func publicPath(projectPath string) string {
	return projectPath + "public"
}

func configPath(projectPath string) string {
	return projectPath + "/secrets/fragmenta.json"
}

func templatesPath() string {
	return os.ExpandEnv("$GOPATH/src/github.com/fragmenta/fragmenta/templates")
}

// Parse the command line arguments and respond
func main() {

	log.SetFlags(log.Ltime)

	args := os.Args
	command := ""

	if len(args) > 1 {
		command = args[1]
	}

	// We should intelligently read project path depending on the command?
	// Or just assume we act on the current directory?
	// NB projectPath might be different from the path in config, which MUST be within a GOPATH
	// this is the local project path
	projectPath, err := filepath.Abs(".")
	if err != nil {
		log.Printf("Error getting path", err)
		return
	}
	if isValidProject(projectPath) {
		readConfig(projectPath)
	}

	switch command {

	case "new", "n":
		runNew(args)

	case "version", "v":
		showVersion()

	case "help", "h", "wat", "?":
		showHelp(args)

	case "server", "s":
		if requireValidProject(projectPath) {
			runServer(projectPath)
		}

	case "test", "t":
		if requireValidProject(projectPath) {
			runTests(args)
		}

	case "build", "B":
		if requireValidProject(projectPath) {
			runBuild(args)
		}

	case "generate", "g":
		if requireValidProject(projectPath) {
			runGenerate(args)
		}

	case "migrate", "m":
		if requireValidProject(projectPath) {
			runMigrate(args)
		}

	case "backup", "b":
		if requireValidProject(projectPath) {
			runBackup(args)
		}

	case "restore", "r":
		if requireValidProject(projectPath) {
			runRestore(args)
		}

	case "deploy", "d":
		if requireValidProject(projectPath) {
			runDeploy(args)
		}

	default:
		if requireValidProject(projectPath) {
			runServer(projectPath)
		} else {
			showHelp(args)
		}
	}

}

// Show the version of this tool
func showVersion() {
	helpString := fragmentaDivider
	helpString += fmt.Sprintf("Fragmenta version: %s", fragmentaVersion)
	helpString += fragmentaDivider
	log.Print(helpString)
}

// Show the help for this tool.
func showHelp(args []string) {
	helpString := fragmentaDivider
	helpString += fmt.Sprintf("Fragmenta version: %s", fragmentaVersion)
	helpString += "\n  fragmenta version -> display version"
	helpString += "\n  fragmenta help -> display help"
	helpString += "\n  fragmenta new [app|cms|blog|URL] path/to/app -> creates a new app from the repository at URL at the path supplied"
	helpString += "\n  fragmenta -> builds and runs a fragmenta app"
	helpString += "\n  fragmenta server -> builds and runs a fragmenta app"
	helpString += "\n  fragmenta test  -> run tests"
	helpString += "\n  fragmenta backup [development|production|test] -> backup the database to db/backup"
	helpString += "\n  fragmenta restore [development|production|test] -> backup the database from latest file in db/backup"
	helpString += "\n  fragmenta deploy [development|production|test] -> build and deploy using bin/deploy"
	helpString += "\n  fragmenta migrate -> runs new sql migrations in db/migrate"
	helpString += "\n  fragmenta generate resource [name] [fieldname]:[fieldtype]* -> creates resource CRUD actions and views"
	helpString += "\n  fragmenta generate migration [name] -> creates a new named sql migration in db/migrate"

	helpString += fragmentaDivider
	log.Print(helpString)
}

// Run the server
func runServer(projectPath string) {
	showVersion()

	killServer()

	log.Println("Building server...")
	err := buildServer(localServerPath(projectPath), nil)

	if err != nil {
		log.Printf("Error building server: %s", err)
		return
	}

	log.Println("Launching server...")
	cmd := exec.Command(localServerPath(projectPath))
	stdout, err := cmd.StdoutPipe()
	stderr, err := cmd.StderrPipe()
	err = cmd.Start()
	if err != nil {
		log.Println(err)
	}
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)
	cmd.Wait()

}

func killServer() {
	runCommand("killall", "-9", serverName())
}

func runCommand(command string, args ...string) ([]byte, error) {

	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stdout
	//	cmd.Stderr = cmd.Stdout
	output, err := cmd.Output()
	if err != nil {
		return output, err
	}

	return output, nil
}

func requireValidProject(projectPath string) bool {
	if isValidProject(projectPath) {
		return true
	}

	log.Printf("\nNo fragmenta project found at this path\n")
	return false

}

func isValidProject(projectPath string) bool {

	// Make sure we have server.go
	_, err := os.Stat(serverCompilePath(projectPath))
	if err != nil {
		return false
	}

	_, err = os.Stat(configPath(projectPath))
	if err != nil {
		return false
	}

	return true
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	if err != nil && os.IsNotExist(err) {
		return false
	}

	return true
}

// Read our config file and set up the server accordingly
func readConfig(projectPath string) error {
	configPath := configPath(projectPath)

	// Read the config json file
	file, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Printf("Error opening config %s %v", configPath, err)
		return err
	}

	var data map[string]map[string]string
	err = json.Unmarshal(file, &data)
	if err != nil {
		log.Printf("Error parsing config %s %v", configPath, err)
		return err
	}

	ConfigDevelopment = data["development"]
	ConfigProduction = data["production"]
	ConfigTest = data["test"]

	return nil
}
