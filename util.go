package glitterboot

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	tmjson "github.com/tendermint/tendermint/libs/json"
	tmos "github.com/tendermint/tendermint/libs/os"
)

// ==== commands ====

func systemctl(args ...string) error {
	return exec.Command("systemctl", args...).Run()
}

// ==== util funcs ====

type CopyFileDesc struct {
	Src  string
	Dest string
}

func copyFile(d CopyFileDesc) error {
	return tmos.CopyFile(d.Src, d.Dest)
}

func jsonToFile(v interface{}, filename string) error {
	b, err := tmjson.Marshal(v)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, b, 0644)
}

func downloadFile(filepath string, url string) (err error) {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func pathJoin(elem ...string) string {
	expands := make([]string, len(elem))
	for i, s := range elem {
		expands[i] = os.ExpandEnv(s)
	}
	return filepath.Join(expands...)
}
