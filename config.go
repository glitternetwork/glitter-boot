package glitterboot

import (
	_ "embed"

	"os"
	"text/template"
)

//go:embed template/tendermint.config.toml
var tendermintConfigTpl string

//go:embed template/glitter.config.toml
var glitterConfigTpl string

//go:embed template/tendermint.service
var tendermintServiceFile []byte

//go:embed template/glitter.service
var glitterServiceFile []byte

func renderTendermintConfig(dest string, data map[string]interface{}) error {
	t, err := template.New("tendermint").Parse(tendermintConfigTpl)
	if err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, data)
}

func renderGlitterConfig(dest string, data map[string]interface{}) error {
	t, err := template.New("glitter").Parse(glitterConfigTpl)
	if err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, data)
}
