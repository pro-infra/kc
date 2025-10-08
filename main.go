package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"k8s.io/client-go/tools/clientcmd"
)

var version string

const GOARCH string = runtime.GOARCH
const GOOS string = runtime.GOOS

func findKubeConfig() (string, error) {
	env := os.Getenv("KUBECONFIG")
	if env != "" {
		return env, nil
	}
	path, err := homedir.Expand("~/.kube/config")
	if err != nil {
		return "", err
	}
	return path, nil
}

func chooseContext(file string) {
	k := newKubeconfig(file)
	result := k.chooseContext("Select Context")
	fmt.Printf("You choose %q\n", result)
	k.CurrentContext = result
	err := clientcmd.WriteToFile(k.Config, file)
	if err != nil {
		panic(err)
	}
}

func stringInput(desc string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(desc)
	outstr, _ := reader.ReadString('\n')
	return strings.Replace(outstr, "\n", "", -1)
}

func stringArrayInput(desc string, sep string) []string {
	o := stringInput(desc)
	o = strings.ReplaceAll(o, " ", "")
	if o == "" {
		return []string{}
	}
	return strings.Split(o, sep)
}

func intInput(desc string, bitsize int) (out int64) {
	var err error

	for {
		o := stringInput(desc)
		out, err = strconv.ParseInt(o, 10, bitsize)
		if err == nil {
			break
		}
	}
	return out
}

func stringArray2Json(a []string) string {
	b, _ := json.Marshal(a)
	return string(b)
}

func addUserContext(file string) {
	k := newKubeconfig(file)
	result := k.chooseContext("Select Admin Context for adding UserCert")
	fmt.Printf("You choose %q\n", result)
	k.CurrentContext = result
	err := clientcmd.WriteToFile(k.Config, file)
	if err != nil {
		panic(err)
	}
	username := ""
	for {
		username = stringInput("Username                          : ")
		if username != "" {
			break
		}
	}
	groups := stringArrayInput("Groups(Comma separeted, no spaces): ", ",")
	days := intInput("days until expiration             : ", 32)

	fmt.Print("\n\nAdd User Context for:\n")
	fmt.Printf("Context to add   : %s\n", k.CurrentContext)
	fmt.Printf("user context name: %s@%s\n", username, k.getClusterName())
	fmt.Printf("Username         : %s\n", username)
	fmt.Printf("Groups           : %s\n", stringArray2Json(groups))
	fmt.Printf("Days             : %d\n", days)
	fmt.Print("\nOK (y/n)?")
	reader := bufio.NewReader(os.Stdin)
	char, _, _ := reader.ReadRune()
	switch char {
	case 'y':
		addUserCert(file, int(days), username, groups...)
	default:
		fmt.Println("Exit without change!")
	}
}

func deleteContext(file string) {
	k := newKubeconfig(file)
	result := k.chooseContext("Delete Context")

	err := clientcmd.WriteToFile(k.Config, fmt.Sprintf("%s.bak", file))
	if err != nil {
		panic(err)
	}

	k.deleteContext(result)
	k.cleanupUnusedItems()

	err = clientcmd.WriteToFile(k.Config, file)
	if err != nil {
		panic(err)
	}
}

func mergeContext(file, addFile string) {
	k := newKubeconfig(file)

	err := clientcmd.WriteToFile(k.Config, fmt.Sprintf("%s.bak", file))
	if err != nil {
		panic(err)
	}

        a := newKubeconfig(addFile)
	for key, c := range a.Clusters {
		k.Clusters[key] = c
	}
	for key, u := range a.AuthInfos {
		k.AuthInfos[key] = u
	}
	for key, c := range a.Contexts {
		k.Contexts[key] = c
	}

	err = clientcmd.WriteToFile(k.Config, file)
	if err != nil {
		panic(err)
	}

}

func main() {
	showVersion := false
	update := false
	dryupd := false
	delete := false
	addUser := false
	addFile := ""
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.BoolVar(&update, "u", false, "Update kc")
	flag.BoolVar(&dryupd, "U", false, "Dry-run update kc")
	flag.BoolVar(&delete, "d", false, "Choose context to delete")
	flag.BoolVar(&addUser, "au", false, "Add a user context")
	flag.StringVar(&addFile, "a", "", "Merge this file into kubeconfig. Use - to read from stdin")
	flag.Parse()

	switch {
	case showVersion:
		fmt.Println(version, GOOS, GOARCH)
		return
	case update || dryupd:
		updatekc(dryupd)
		return
	}

	if delete && addFile != "" {
		log.Fatalln("delete and merge is not allowed")
	}
	file, err := findKubeConfig()
	if err != nil {
		panic(err)
	}

	switch {
	case addUser:
		addUserContext(file)
	case delete:
		deleteContext(file)
	case addFile != "":
		mergeContext(file, addFile)
	default:
		chooseContext(file)
	}
}
