package glitterboot

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/crypto"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
)

type NodeOpsArgs struct {
	Type                NodeOperateType
	Seeds               string
	Moniker             string
	IndexMode           string
	GlitterBinaryURL    string
	TendermintBinaryURL string
}

var (
	installdir   = "/usr/local/glitter"
	bootdir      = filepath.Join(installdir, "glitter-boot")
	storedir     = filepath.Join(bootdir, "store.json")
	glitterUser  = "glitter"
	glitterGroup = "glitter"
)

const (
	keySeeds          = "seeds"
	keyMoniker        = "moniker"
	keyNodeID         = "node_id"
	keyPubKey         = "pub_key"
	keyPubKeyAddress  = "pub_key_address"
	keyInitDone       = "init_done"
	keyValidatorStage = "validator_stage"
)

type NodeOperateType int

const (
	OpsInit NodeOperateType = 0 + iota
	OpsStartFullNode
	OpsStartValidator
	OpsStopNode
	OpsShowNodeInfo
)

func NodeOperate(ctx context.Context, args NodeOpsArgs) {
	switch args.Type {
	case OpsInit:
		initNode(ctx, args)
	case OpsStartFullNode:
		startFullNode(ctx, args)
	case OpsStartValidator:
		startValidator(ctx, args)
	case OpsStopNode:
		stopNode(ctx, args)
	case OpsShowNodeInfo:
		showNodeInfo(ctx, args)
	}
}

func initNode(ctx context.Context, args NodeOpsArgs) {
	p := &nodeOpsPipe{}
	p.
		Do("Prepare", func(ctx *setupNodeCtx) error {
			ctx.WorkDir = bootdir
			ctx.StoreDir = storedir
			ctx.Moniker = args.Moniker
			ctx.IndexMode = args.IndexMode
			ctx.SeedsStr = args.Seeds
			ctx.GlitterBinaryURL = args.GlitterBinaryURL
			ctx.TendermintBinaryURL = args.TendermintBinaryURL

			var err error
			err = checkUserGroup(glitterUser, glitterGroup)

			if err != nil {
				return errors.Errorf("failed to got glitter user/group: %v", err)
			}

			ctx.store, err = newFileStore(storedir, true)
			if err != nil {
				return err
			}
			done, err := ctx.store.Get(keyInitDone)
			ctx.assert(err)
			if done == "true" {
				return errors.New("Full node has already setup,please remove ~/.glitter-boot dir then redo current command if you want to reset it")
			}

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
			os.MkdirAll(bootdir, 0755)
			c, err := NewTMClient(ctx.OldClusterTendermintRPCURL)
			ctx.assert(err)

			cLocal, err := NewTMClient(ctx.LocalTendermintRPCURL)
			ctx.assert(err)

			ctx.tmClusterClient = c
			ctx.tmLocalClient = cLocal
			return nil
		}).
		Do("Download tendermint", stepDownloadTendermint).
		Do("Download glitter", stepDownloadGlitter).
		Do("Download genesis file", stepDownloadGenesis).
		Do("Render glitter config", stepRenderGlitterConfig).
		Do("Render tendermint config", stepRenderTendermintConfig).
		Do("Render systemctl config", stepRenderSystemctlConfig).
		Do("Generate nodekey files", stepGenerateNodeKeyFile).
		Do("Generate validator key files", stepGenerateValidatorFile).
		Do("Reset and copy files", stepResetCopyFile).
		Do("Save config", func(ctx *setupNodeCtx) error {
			err := ctx.store.Set(keySeeds, ctx.SeedsStr)
			ctx.assert(err)

			err = ctx.store.Set(keyMoniker, ctx.Moniker)
			ctx.assert(err)

			err = ctx.store.Set(keyInitDone, "true")
			ctx.assert(err)

			return nil
		})

	if err := p.Error(); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Init node successfully")
}

func startFullNode(ctx context.Context, args NodeOpsArgs) {
	p := &nodeOpsPipe{}
	p.
		Do("Check", func(ctx *setupNodeCtx) error {
			ctx.WorkDir = bootdir
			var err error
			ctx.store, err = newFileStore(storedir, true)
			if err != nil {
				return err
			}
			done, err := ctx.store.Get(keyInitDone)
			ctx.assert(err)
			if done != "true" {
				return errors.New("Please init node first before start the fullnode")
			}
			return nil
		}).
		Do("Switch to fullnode mode", stepSwitchToFullNode).
		Do("Restart glitter",
			func(ctx *setupNodeCtx) error {
				return systemctl("restart", "glitter")
			},
		)

	if err := p.Error(); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Start fullnode successfully")
}

func startValidator(ctx context.Context, args NodeOpsArgs) {
	p := &nodeOpsPipe{}
	p.
		Do("Prepare", func(ctx *setupNodeCtx) error {
			ctx.WorkDir = bootdir
			ctx.StoreDir = storedir

			var err error
			ctx.store, err = newFileStore(storedir, false)
			ctx.assert(err)

			done, err := ctx.store.Get(keyInitDone)
			ctx.assert(err)
			if done != "true" {
				return errors.New("Please init node first before start the validator")
			}

			ctx.IndexMode = "kv"
			ctx.Moniker, err = ctx.store.Get(keyMoniker)
			ctx.assert(err)

			ctx.LocalTendermintRPCURL = "http://127.0.0.1:26657"
			cLocal, err := NewTMClient(ctx.LocalTendermintRPCURL)
			ctx.assert(err)

			ctx.tmLocalClient = cLocal
			return nil
		}).
		Do("Waiting to receive a validator change event...", stepWaitForValidator).
		Do("Switch to validator mode", stepSwitchToValidator).
		Do("Restart glitter",
			func(ctx *setupNodeCtx) error {
				return systemctl("restart", "glitter")
			},
		)
	if err := p.Error(); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Start validator successfully")
}

func stopNode(ctx context.Context, args NodeOpsArgs) {
	p := &nodeOpsPipe{}
	p.
		Do("Check", func(ctx *setupNodeCtx) error {
			ctx.WorkDir = bootdir
			ctx.StoreDir = storedir

			var err error
			ctx.store, err = newFileStore(storedir, false)
			ctx.assert(err)

			done, err := ctx.store.Get(keyInitDone)
			ctx.assert(err)
			if done != "true" {
				return errors.New("Please init node first before stop")
			}
			return nil
		}).
		Do("Stop tendermint",
			func(ctx *setupNodeCtx) error {
				return systemctl("stop", "tendermint")
			},
		).
		Do("Stop glitter",
			func(ctx *setupNodeCtx) error {
				return systemctl("stop", "glitter")
			},
		)
	if err := p.Error(); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Stop node successfully")
}

func showNodeInfo(ctx context.Context, args NodeOpsArgs) {
	p := &nodeOpsPipe{}
	p.
		Do("Check", func(ctx *setupNodeCtx) error {
			ctx.WorkDir = bootdir
			ctx.StoreDir = storedir

			var err error
			ctx.store, err = newFileStore(storedir, false)
			ctx.assert(err)

			done, err := ctx.store.Get(keyInitDone)
			ctx.assert(err)
			if done != "true" {
				return errors.New("Please init node first")
			}
			return nil
		}).
		Do("Node Info", func(ctx *setupNodeCtx) error {
			const info = `
NodeID:		%s
Moniker:	%s

PubKey:		%s
Address:	%s

Tendermint Status: %s
Glitter	   Status: %s

PrivateKeyFile:	~/.glitter-boot/priv_validator_key.json
GlitterBootDir:	/usr/local/glitter/glitter-boot
GlitterDir:		/usr/local/glitter/glitter

`
			get := func(key string) string {
				value, err := ctx.store.Get(key)
				ctx.assert(err)
				return value
			}
			tmStatus, _ := systemctlOut("is-active", "tendermint")
			glitterStatus, _ := systemctlOut("is-active", "glitter")

			fmt.Printf(info,
				get(keyNodeID),
				get(keyMoniker),
				get(keyPubKey),
				get(keyPubKeyAddress),
				tmStatus,
				glitterStatus,
			)
			return nil
		},
		)
	if err := p.Error(); err != nil {
		fmt.Printf("%v\n%s\n", err, "Did you initialize the node?")
		return
	}
}

func stepDownloadTendermint(ctx *setupNodeCtx) error {
	return downloadFile(pathJoin(bootdir, "tendermint"), ctx.TendermintBinaryURL)
}

func stepDownloadGlitter(ctx *setupNodeCtx) error {
	return downloadFile(pathJoin(bootdir, "glitter"), ctx.GlitterBinaryURL)
}

func stepDownloadGenesis(ctx *setupNodeCtx) error {
	g, err := ctx.tmClusterClient.Genesis(context.TODO())
	ctx.assert(err)
	return jsonToFile(g.Genesis, pathJoin(bootdir, "genesis.json"))
}

func stepRenderGlitterConfig(ctx *setupNodeCtx) error {
	if ctx.IndexMode != "kv" && ctx.IndexMode != "es" {
		return errors.Errorf("invalid glitter index mode: %s", ctx.IndexMode)
	}
	return renderGlitterConfig(pathJoin(bootdir, "glitter.config.toml"),
		map[string]interface{}{
			"IndexMode": ctx.IndexMode,
		})
}

func stepRenderTendermintConfig(ctx *setupNodeCtx) error {
	err := renderTendermintConfig(pathJoin(bootdir, "tendermint-full.config.toml"),
		map[string]interface{}{
			"Moniker": ctx.Moniker,
			"Seeds":   ctx.SeedsStr,
			"Mode":    "full",
		})
	ctx.assert(err)
	return renderTendermintConfig(pathJoin(bootdir, "tendermint-validator.config.toml"),
		map[string]interface{}{
			"Moniker": ctx.Moniker,
			"Seeds":   ctx.SeedsStr,
			"Mode":    "validator",
		})
}

func stepRenderSystemctlConfig(ctx *setupNodeCtx) error {
	err := ioutil.WriteFile(pathJoin(bootdir, "tendermint.service"), tendermintServiceFile, 0644)
	ctx.assert(err)
	return ioutil.WriteFile(pathJoin(bootdir, "glitter.service"), glitterServiceFile, 0644)
}

func stepGenerateNodeKeyFile(ctx *setupNodeCtx) error {
	nodeKeyPath := pathJoin(ctx.WorkDir, "node_key.json")

	_, err := os.Stat(nodeKeyPath)
	if os.IsExist(err) {
		fmt.Printf("[WARN] Skip Generate NodeKeyFile: node_key alreay exist")
		return nil
	}

	key, err := p2p.LoadOrGenNodeKey(nodeKeyPath)
	ctx.assert(err)
	err = ctx.store.Set(keyNodeID, string(key.ID()))
	ctx.assert(err)

	return nil
}

func stepGenerateValidatorFile(ctx *setupNodeCtx) error {
	validatorKeyPath := pathJoin(ctx.WorkDir, "priv_validator_key.json")
	validatorStatePath := pathJoin(ctx.WorkDir, "priv_validator_state.json")

	_, err := os.Stat(validatorKeyPath)
	if os.IsExist(err) {
		fmt.Printf("[WARN] Skip Generate ValidatorFile: validator_key alreay exist")
		return nil
	}

	pv := privval.GenFilePV("", "")
	jsbz, err := tmjson.Marshal(pv.Key)
	ctx.assert(err)

	err = ioutil.WriteFile(validatorKeyPath, jsbz, 0644)
	ctx.assert(err)

	stb, err := tmjson.Marshal(pv.LastSignState)
	ctx.assert(err)

	err = ioutil.WriteFile(validatorStatePath, stb, 0644)
	ctx.assert(err)

	ctx.ValidatorAddress = pv.GetAddress().String()
	ctx.ValidatorPubKey = pv.Key.PubKey

	pubkeyString := base64.StdEncoding.EncodeToString(ctx.ValidatorPubKey.Bytes())
	err = ctx.store.Set(keyPubKey, pubkeyString)
	ctx.assert(err)

	err = ctx.store.Set(keyPubKeyAddress, pv.GetAddress().String())
	ctx.assert(err)
	return nil
}

func stepResetCopyFile(ctx *setupNodeCtx) error {
	systemctl("stop", "tendermint")
	systemctl("stop", "glitter")

	os.RemoveAll(pathJoin(installdir, "tendermint"))
	os.RemoveAll(pathJoin(installdir, "glitter"))
	os.RemoveAll(pathJoin("/tmp/kvstore"))
	os.MkdirAll(pathJoin(installdir, "tendermint/config"), 0755)
	os.MkdirAll(pathJoin(installdir, "tendermint/data"), 0755)
	os.MkdirAll(pathJoin(installdir, "glitter"), 0755)

	tmConfigSrcPath := pathJoin(ctx.WorkDir, "tendermint-full.config.toml")
	genesisSrcPath := pathJoin(ctx.WorkDir, "genesis.json")
	nodeKeySrcPath := pathJoin(ctx.WorkDir, "node_key.json")
	validatorKeySrcPath := pathJoin(ctx.WorkDir, "priv_validator_key.json")
	validatorStateSrcPath := pathJoin(ctx.WorkDir, "priv_validator_state.json")
	glitterConfigSrcPath := pathJoin(ctx.WorkDir, "glitter.config.toml")

	glitterServiceSrcPath := pathJoin(ctx.WorkDir, "glitter.service")
	tmServiceSrcPath := pathJoin(ctx.WorkDir, "tendermint.service")

	glitterBinSrcPath := pathJoin(ctx.WorkDir, "glitter")
	tmBinSrcPath := pathJoin(ctx.WorkDir, "tendermint")

	copys := []CopyFileDesc{
		{tmConfigSrcPath, pathJoin(installdir, "tendermint/config", "config.toml")},
		{genesisSrcPath, pathJoin(installdir, "tendermint/config", "genesis.json")},
		{nodeKeySrcPath, pathJoin(installdir, "tendermint/config", "node_key.json")},
		{validatorKeySrcPath, pathJoin(installdir, "tendermint/config", "priv_validator_key.json")},
		{validatorStateSrcPath, pathJoin(installdir, "tendermint/data", "priv_validator_state.json")},
		{glitterConfigSrcPath, pathJoin(installdir, "glitter", "config.toml")},
		{glitterServiceSrcPath, "/etc/systemd/system/glitter.service"},
		{tmServiceSrcPath, "/etc/systemd/system/tendermint.service"},
		{glitterBinSrcPath, "/usr/bin/glitter"},
		{tmBinSrcPath, "/usr/bin/tendermint"},
	}
	for _, c := range copys {
		err := copyFile(c)
		if err != nil {
			return errors.Errorf("copy file error: %+v err=%v", c, err)
		}
	}
	os.Chmod("/usr/bin/glitter", 0755)
	os.Chmod("/usr/bin/tendermint", 0755)

	err := chown(installdir, glitterUser, glitterGroup, true)
	ctx.assert(err)

	err = chown("/usr/bin/glitter", glitterUser, glitterGroup, true)
	ctx.assert(err)

	err = chown("/usr/bin/tendermint", glitterUser, glitterGroup, true)
	ctx.assert(err)

	return systemctl("daemon-reload")
}

func stepSwitchToFullNode(ctx *setupNodeCtx) error {
	tmConfigSrcPath := pathJoin(ctx.WorkDir, "tendermint-full.config.toml")
	err := copyFile(CopyFileDesc{tmConfigSrcPath, pathJoin(installdir, "tendermint/config", "config.toml")})
	ctx.assert(err)

	return systemctl("restart", "tendermint")
}

func stepSwitchToValidator(ctx *setupNodeCtx) error {
	tmConfigSrcPath := pathJoin(ctx.WorkDir, "tendermint-validator.config.toml")
	err := copyFile(CopyFileDesc{tmConfigSrcPath, pathJoin(installdir, "tendermint/config", "config.toml")})
	ctx.assert(err)

	return systemctl("restart", "tendermint")
}

func stepWaitForValidator(ctx *setupNodeCtx) error {
	stage, err := ctx.store.Get(keyValidatorStage)
	ctx.assert(err)
	if stage == "ok" {
		return nil
	}

	time.Sleep(time.Second * 5)
	address, err := ctx.store.Get(keyPubKeyAddress)
	ctx.assert(err)

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
			if v.Address.String() == address {
				err = ctx.store.Set(keyValidatorStage, "ok")
				ctx.assert(err)
				return nil
			}
		}
	}
}

type setupNodeCtx struct {
	WorkDir  string
	StoreDir string

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

	UID int
	GID int

	store           store
	tmClusterClient *TendermintClient
	tmLocalClient   *TendermintClient
}

/* setupNodePipe */

type nodeOpsPipe struct {
	ctx  setupNodeCtx
	step string
	err  error
}

func (p *nodeOpsPipe) Do(step string, f func(ctx *setupNodeCtx) error) (pp *nodeOpsPipe) {
	defer p.tryRecover(&pp)
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

func (p *nodeOpsPipe) tryRecover(v **nodeOpsPipe) {
	iv := recover()
	if iv == nil {
		*v = p
		return
	}
	if e, ok := iv.(pipeError); ok {
		p.err = e
		*v = p
		return
	}
	panic(iv)
}

func (p *nodeOpsPipe) Error() error {
	if p.err == nil {
		return nil
	}
	return errors.Errorf("failed to execute setp [%s]: %v", p.step, p.err)
}
