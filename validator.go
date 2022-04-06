package glitterboot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/crypto"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
)

type SetupNodeArgs struct {
	ValidatorMode       bool
	Seeds               string
	Moniker             string
	IndexMode           string
	GlitterBinaryURL    string // "http://host.docker.internal:8080/glitter"
	TendermintBinaryURL string
}

const workdir = ".glitter_boot"

// SetupNode set up and start a glitter validator
// 1.  download tm,glitter
// 2.  install & resgister systemd
// 3.  set seeds
// 4.  download genesis.json
// 5.  copy glitter config
// 6.  generate tm config and validator files
// 7.  update validator to old cluster
// 8.  switch to full node
// 9.  check if become a validator
// 10. switch to validator node
func SetupNode(ctx context.Context, args SetupNodeArgs) {
	p := &setupNodePipe{}

	p.
		Do("prepare", func(ctx *setupNodeCtx) error {
			ctx.WorkDir = workdir
			ctx.Moniker = args.Moniker
			ctx.IndexMode = args.IndexMode
			ctx.SeedsStr = args.Seeds
			ctx.GlitterBinaryURL = args.GlitterBinaryURL
			ctx.TendermintBinaryURL = args.TendermintBinaryURL

			for _, s := range strings.Split(ctx.SeedsStr, ",") {
				s = strings.TrimSpace(s)
				a, err := parseNodeAddr(s)
				if err != nil {
					return err
				}
				ctx.Seeds = append(ctx.Seeds, a)
			}
			if len(ctx.Seeds) == 0 {
				return errors.New("invalid argument seeds: at least provide one seed")
			}
			selectedSeed := ctx.Seeds[0]
			ctx.OldClusterTendermintRPCURL = "http://" + net.JoinHostPort(selectedSeed.Host, "26657")
			ctx.OldClusterGlitterURL = "http://" + net.JoinHostPort(selectedSeed.Host, "26659")
			ctx.LocalTendermintRPCURL = "http://127.0.0.1:26657"
			os.Mkdir(workdir, 0755)
			c, err := NewTMClient(ctx.OldClusterTendermintRPCURL)
			ctx.assert(err)

			cLocal, err := NewTMClient(ctx.LocalTendermintRPCURL)
			ctx.assert(err)

			ctx.tmClusterClient = c
			ctx.tmLocalClient = cLocal
			return nil
		})

	p.
		Do("download tendermint", stepDownloadTendermint).
		Do("download glitter", stepDownloadGlitter).
		Do("download genesis file", stepDownloadGenesis).
		Do("render glitter config", stepRenderGlitterConfig).
		Do("render tendermint config", stepRenderTendermintConfig).
		Do("render systemctl config", stepRenderSystemctlConfig).
		Do("generate validator files", stepGenerateValidatorFile).
		Do("reset and copy files", stepResetCopyFile).
		Do("start tendermint full node",
			func(ctx *setupNodeCtx) error {
				return systemctl("start", "tendermint")
			}).
		Do("start glitter",
			func(ctx *setupNodeCtx) error {
				return systemctl("start", "glitter")
			})

	if args.ValidatorMode {
		p.
			Do("make validator change", stepMakeValidatorChange).
			Do("wait received validator change", stepWaitForValidator).
			Do("stop tendermint full node",
				func(ctx *setupNodeCtx) error {
					return systemctl("stop", "tendermint")
				}).
			Do("switch to validator mode", stepSwitchToValidator)
	}

	if err := p.Error(); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("done")
}

func stepDownloadTendermint(ctx *setupNodeCtx) error {
	return downloadFile(pathJoin(workdir, "tendermint"), ctx.TendermintBinaryURL)
}

func stepDownloadGlitter(ctx *setupNodeCtx) error {
	return downloadFile(pathJoin(workdir, "glitter"), ctx.GlitterBinaryURL)
}

func stepDownloadGenesis(ctx *setupNodeCtx) error {
	g, err := ctx.tmClusterClient.Genesis(context.TODO())
	ctx.assert(err)
	return jsonToFile(g.Genesis, pathJoin(workdir, "genesis.json"))
}

func stepRenderGlitterConfig(ctx *setupNodeCtx) error {
	if ctx.IndexMode != "kv" && ctx.IndexMode != "es" {
		return errors.Errorf("invalid glitter index mode: %s", ctx.IndexMode)
	}
	return renderGlitterConfig(pathJoin(workdir, "glitter.config.toml"),
		map[string]interface{}{
			"IndexMode": ctx.IndexMode,
		})
}

func stepRenderTendermintConfig(ctx *setupNodeCtx) error {
	err := renderTendermintConfig(pathJoin(workdir, "tendermint-full.config.toml"),
		map[string]interface{}{
			"Moniker": ctx.Moniker,
			"Seeds":   ctx.SeedsStr,
			"Mode":    "full",
		})
	ctx.assert(err)
	return renderTendermintConfig(pathJoin(workdir, "tendermint-validator.config.toml"),
		map[string]interface{}{
			"Moniker": ctx.Moniker,
			"Seeds":   ctx.SeedsStr,
			"Mode":    "validator",
		})
}

func stepRenderSystemctlConfig(ctx *setupNodeCtx) error {
	err := ioutil.WriteFile(pathJoin(workdir, "tendermint.service"), tendermintServiceFile, 0644)
	ctx.assert(err)
	return ioutil.WriteFile(pathJoin(workdir, "glitter.service"), glitterServiceFile, 0644)
}

func stepGenerateValidatorFile(ctx *setupNodeCtx) error {
	wd, err := os.Getwd()
	ctx.assert(err)

	nodeKeyPath := pathJoin(wd, ctx.WorkDir, "node_key.json")
	_, err = p2p.LoadOrGenNodeKey(nodeKeyPath)
	ctx.assert(err)

	// fmt.Println("node_id=", nodeKey.ID())

	validatorKeyPath := pathJoin(wd, ctx.WorkDir, "priv_validator_key.json")
	pv := privval.GenFilePV("", "")
	jsbz, err := tmjson.Marshal(pv.Key)
	ctx.assert(err)

	err = ioutil.WriteFile(validatorKeyPath, jsbz, 0644)
	ctx.assert(err)

	validatorStatePath := pathJoin(wd, ctx.WorkDir, "priv_validator_state.json")
	stb, err := tmjson.Marshal(pv.LastSignState)
	ctx.assert(err)

	err = ioutil.WriteFile(validatorStatePath, stb, 0644)
	ctx.assert(err)

	ctx.ValidatorAddress = pv.GetAddress().String()
	ctx.ValidatorPubKey = pv.Key.PubKey
	return nil
}

func stepResetCopyFile(ctx *setupNodeCtx) error {
	systemctl("stop", "tendermint")
	systemctl("stop", "glitter")

	os.RemoveAll(pathJoin("$HOME/.tendermint"))
	os.RemoveAll(pathJoin("$HOME/.glitter"))
	os.RemoveAll(pathJoin("/tmp/kvstore"))
	os.MkdirAll(pathJoin("$HOME/.tendermint/config"), 0755)
	os.MkdirAll(pathJoin("$HOME/.tendermint/data"), 0755)
	os.MkdirAll(pathJoin("$HOME/.glitter"), 0755)

	wd, err := os.Getwd()
	ctx.assert(err)

	tmConfigSrcPath := pathJoin(wd, ctx.WorkDir, "tendermint-full.config.toml")
	genesisSrcPath := pathJoin(wd, ctx.WorkDir, "genesis.json")
	nodeKeySrcPath := pathJoin(wd, ctx.WorkDir, "node_key.json")
	validatorKeySrcPath := pathJoin(wd, ctx.WorkDir, "priv_validator_key.json")
	validatorStateSrcPath := pathJoin(wd, ctx.WorkDir, "priv_validator_state.json")
	glitterConfigSrcPath := pathJoin(wd, ctx.WorkDir, "glitter.config.toml")

	glitterServiceSrcPath := pathJoin(wd, ctx.WorkDir, "glitter.service")
	tmServiceSrcPath := pathJoin(wd, ctx.WorkDir, "tendermint.service")

	glitterBinSrcPath := pathJoin(wd, ctx.WorkDir, "glitter")
	tmBinSrcPath := pathJoin(wd, ctx.WorkDir, "tendermint")

	copys := []CopyFileDesc{
		{tmConfigSrcPath, pathJoin("$HOME/.tendermint/config", "config.toml")},
		{genesisSrcPath, pathJoin("$HOME/.tendermint/config", "genesis.json")},
		{nodeKeySrcPath, pathJoin("$HOME/.tendermint/config", "node_key.json")},
		{validatorKeySrcPath, pathJoin("$HOME/.tendermint/config", "priv_validator_key.json")},
		{validatorStateSrcPath, pathJoin("$HOME/.tendermint/data", "priv_validator_state.json")},
		{glitterConfigSrcPath, pathJoin("$HOME/.glitter", "config.toml")},
		{glitterServiceSrcPath, "/etc/systemd/system/glitter.service"},
		{tmServiceSrcPath, "/etc/systemd/system/tendermint.service"},
		{glitterBinSrcPath, "/usr/bin/glitter"},
		{tmBinSrcPath, "/usr/bin/tendermint"},
	}
	for _, c := range copys {
		err = copyFile(c)
		if err != nil {
			return errors.Errorf("copy file error: %+v err=%v", c, err)
		}
	}
	os.Chmod("/usr/bin/glitter", 0755)
	os.Chmod("/usr/bin/tendermint", 0755)
	return systemctl("daemon-reload")
}

func stepSwitchToValidator(ctx *setupNodeCtx) error {
	wd, err := os.Getwd()
	ctx.assert(err)

	tmConfigSrcPath := pathJoin(wd, ctx.WorkDir, "tendermint-full.config.toml")
	err = copyFile(CopyFileDesc{tmConfigSrcPath, pathJoin("$HOME/.tendermint/config", "config.toml")})
	ctx.assert(err)

	return systemctl("restart", "tendermint")
}

type UpdateValidatorBody struct {
	PubKey crypto.PubKey `json:"pub_key"`
	Power  int64         `json:"power"`
}

func stepMakeValidatorChange(ctx *setupNodeCtx) error {
	u := fmt.Sprintf("%s/v1/admin/update_validator", ctx.OldClusterGlitterURL)
	b, err := json.Marshal(UpdateValidatorBody{
		PubKey: ctx.ValidatorPubKey,
		Power:  1,
	})
	ctx.assert(err)

	resp, err := http.Post(u, "application/json", bytes.NewBuffer(b))
	ctx.assert(err)

	defer resp.Body.Close()
	r, err := ioutil.ReadAll(resp.Body)
	ctx.assert(err)

	fmt.Printf("%s\n", r)
	return nil
}

func stepWaitForValidator(ctx *setupNodeCtx) error {
	time.Sleep(time.Second * 5)
	errCnt := 0
	for {
		time.Sleep(time.Second)
		resp, err := ctx.tmLocalClient.Validators(context.TODO(), nil, nil, nil)
		if err != nil {
			if errCnt > 10 {
				return err
			}
			errCnt++
			continue
		}

		for _, v := range resp.Validators {
			if v.Address.String() == ctx.ValidatorAddress {
				return nil
			}
		}
	}
}

type setupNodeCtx struct {
	WorkDir string

	Moniker   string
	NodeMode  string
	IndexMode string

	TendermintBinaryURL string
	GlitterBinaryURL    string

	Seeds    []*NodeAddr
	SeedsStr string

	OldClusterTendermintRPCURL string
	OldClusterGlitterURL       string
	LocalTendermintRPCURL      string

	ValidatorAddress string
	ValidatorPubKey  crypto.PubKey

	tmClusterClient *TendermintClient
	tmLocalClient   *TendermintClient
}

/* setupNodePipe */

type setupNodePipe struct {
	ctx  setupNodeCtx
	step string
	err  error
}

func (p *setupNodePipe) Do(step string, f func(ctx *setupNodeCtx) error) *setupNodePipe {
	defer p.tryRecover()
	if p.err != nil {
		return p
	}
	p.step = step
	fmt.Printf("[step] %s\n", step)
	p.err = f(&p.ctx)
	return p
}

type pipeError error

func (p *setupNodeCtx) assert(e error) {
	if e == nil {
		return
	}
	panic(pipeError(e))
}

func (p *setupNodePipe) tryRecover() {
	iv := recover()
	if iv == nil {
		return
	}
	if e, ok := iv.(pipeError); ok {
		p.err = e
		return
	}
	panic(iv)
}

func (p *setupNodePipe) Error() error {
	if p.err == nil {
		return nil
	}
	return errors.Errorf("failed to execute setp [%s]: %v", p.step, p.err)
}
