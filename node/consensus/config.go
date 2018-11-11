package consensus

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/rand"
	"strings"
	"time"

	"github.com/fatih/structs"
	"github.com/gelembjuk/oursql/lib/net"
	"github.com/gelembjuk/oursql/node/structures"
	"github.com/mitchellh/mapstructure"
)

const (
	KindConseususPoW = "proofofwork"
)

type ConsensusConfigCost struct {
	Default     float64
	RowDelete   float64
	RowUpdate   float64
	RowInsert   float64
	TableCreate float64
}
type ConsensusConfigTable struct {
	Table            string
	AllowRowDelete   bool
	AllowRowUpdate   bool
	AllowRowInsert   bool
	AllowTableCreate bool
	TransactionCost  ConsensusConfigCost
}
type ConsensusConfigApplication struct {
	Name    string
	WebSite string
	Team    string
}
type consensusConfigState struct {
	isDefault bool
	filePath  string
}
type ConsensusConfig struct {
	Application       ConsensusConfigApplication
	Kind              string
	CoinsForBlockMade float64
	Settings          map[string]interface{}
	AllowTableCreate  bool
	AllowTableDrop    bool
	AllowRowDelete    bool
	TransactionCost   ConsensusConfigCost
	UnmanagedTables   []string
	TableRules        []ConsensusConfigTable
	InitNodesAddreses []string
	state             consensusConfigState
}

// Load config from config file. Some config options an be missed
// missed options must be replaced with default values correctly
func NewConfigFromFile(filepath string) (*ConsensusConfig, error) {
	// we open a file only if it exists. in other case options can be set with command line

	jsonStr, err := ioutil.ReadFile(filepath)

	if err != nil {
		// error is bad only if file exists but we can not open to read
		return nil, err
	}

	config := ConsensusConfig{}

	err = json.Unmarshal(jsonStr, &config)

	if err != nil {
		return nil, err
	}

	if config.CoinsForBlockMade == 0 {
		config.CoinsForBlockMade = 10
	}

	if config.Kind == "" {
		config.Kind = KindConseususPoW
	}
	if config.Kind == KindConseususPoW {
		// check all PoW settings are done
		s := ProofOfWorkSettings{}

		mapstructure.Decode(config.Settings, &s)

		s.completeSettings()

		config.Settings = structs.Map(s)
	}

	config.state.isDefault = false
	config.state.filePath = filepath

	return &config, nil
}

func NewConfigDefault() (*ConsensusConfig, error) {
	c := ConsensusConfig{}
	c.Kind = KindConseususPoW
	c.CoinsForBlockMade = 10
	c.AllowTableCreate = true
	c.AllowTableDrop = true
	c.AllowRowDelete = true
	c.UnmanagedTables = []string{}
	c.TableRules = []ConsensusConfigTable{}
	c.InitNodesAddreses = []string{}

	// make defauls PoW settings
	s := ProofOfWorkSettings{}
	s.completeSettings()

	c.Settings = structs.Map(s)

	c.state.isDefault = true
	c.state.filePath = ""

	return &c, nil
}

func (cc ConsensusConfig) GetInfoForTransactions() structures.ConsensusInfo {
	return structures.ConsensusInfo{cc.CoinsForBlockMade}
}

// Exports config to file
func (cc ConsensusConfig) ExportToFile(filepath string, defaultaddresses string, appname string, thisnodeaddr string) error {
	jsondata, err := cc.Export(defaultaddresses, appname, thisnodeaddr)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath, jsondata, 0644)

	return err
}

// Exports config to JSON string
func (cc ConsensusConfig) Export(defaultaddresses string, appname string, thisnodeaddr string) (jsondata []byte, err error) {
	addresses := []string{}

	if defaultaddresses != "" {
		list := strings.Split(defaultaddresses, ",")

		for _, a := range list {
			if a == "" {
				continue
			}
			if a == "own" {
				if thisnodeaddr != "" {
					a = thisnodeaddr
				} else {
					continue
				}
			}
			addresses = append(addresses, a)
		}
	}

	if len(addresses) > 0 {
		cc.InitNodesAddreses = addresses
	}

	if len(cc.InitNodesAddreses) == 0 && thisnodeaddr != "" {
		cc.InitNodesAddreses = []string{thisnodeaddr}
	}

	if len(cc.InitNodesAddreses) == 0 {
		err = errors.New("List of default addresses is empty")
		return
	}

	if appname != "" {
		cc.Application.Name = appname
	}

	if cc.Application.Name == "" {
		err = errors.New("Application name is empty. It is required")
		return
	}

	jsondata, err = json.Marshal(cc)

	return
}

// Returns one of addresses listed in initial addresses
func (cc ConsensusConfig) GetRandomInitialAddress() *net.NodeAddr {
	if len(cc.InitNodesAddreses) == 0 {
		return nil
	}
	rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
	addr := cc.InitNodesAddreses[rand.Intn(len(cc.InitNodesAddreses))]

	na := net.NodeAddr{}
	na.LoadFromString(addr)

	return &na
}

// Checks if a config structure was loaded from file or not
func (cc ConsensusConfig) IsDefault() bool {

	return cc.state.isDefault
}

// Set config file path. this defines a path where a config file should be, even if it is not yet here
func (cc *ConsensusConfig) SetConfigFilePath(fp string) {
	cc.state.filePath = fp
}

// Replace consensus config file . It checks if a config is correct, if can be parsed

func (cc ConsensusConfig) UpdateConfig(jsondoc []byte) error {

	if cc.state.filePath == "" {
		return errors.New("COnfig file path missed. Can not save")
	}

	return ioutil.WriteFile(cc.state.filePath, jsondoc, 0644)
}
