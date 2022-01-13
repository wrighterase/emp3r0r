package cc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	emp3r0r_data "github.com/jm33-m0/emp3r0r/core/lib/data"
	"github.com/jm33-m0/emp3r0r/core/lib/util"
	"github.com/olekukonko/tablewriter"
)

// ModConfig config.json of a module
// {
//     "name": "LES",
//     "exec": "les.sh",
//     "platform": "Linux",
//     "author": "jm33-ng",
//     "date": "2022-01-12",
//     "comment": "https://github.com/mzet-/linux-exploit-suggester",
//     "options": {
//         "args": ["--checksec", "run les.sh with this commandline arg"]
//     }
// }
type ModConfig struct {
	Name     string `json:"name"`
	Exec     string `json:"exec"`
	Platform string `json:"platform"`
	Author   string `json:"author"`
	Date     string `json:"date"`
	Comment  string `json:"comment"`

	// option: [value, help]
	Options map[string][]string `json:"options"`
}

// stores module configs
var ModuleConfigs = make(map[string]ModConfig, 1)

// moduleCustom run a custom module
func moduleCustom() {
	config_json := ModuleDir + CurrentMod + "/config.json"
	temp_config_json := Temp + "/config.json"
	// update config.json
	util.Copy(config_json, temp_config_json)
	defer os.Rename(temp_config_json, config_json)
	config, err := readModCondig(temp_config_json)
	if err != nil {
		CliPrintError("Read config: %v", err)
		return
	}
	for opt, val := range config.Options {
		val[0] = Options[opt].Val
	}
	err = writeModCondig(config, config_json)
	if err != nil {
		CliPrintError("Update config.json: %v", err)
		return
	}

	// compress module files
	err = util.TarBz2(ModuleDir+CurrentMod, WWWRoot+CurrentMod)
	if err != nil {
		CliPrintError("Compressing %s: %v", CurrentMod, err)
		return
	}

	// tell agent to download and execute this module
	err = SendCmdToCurrentTarget("!custom_module "+CurrentMod, "")
	if err != nil {
		CliPrintError("Sending command to %s: %v", CurrentTarget.Tag, err)
	}
}

// Print module meta data
func ModuleDetails(modName string) {
	config, exists := ModuleConfigs[modName]
	if !exists {
		return
	}

	// build table
	tdata := [][]string{}
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Name", "Exec", "Platform", "Author", "Date", "Comment"})
	table.SetBorder(true)
	table.SetRowLine(true)
	table.SetAutoWrapText(true)

	// color
	table.SetHeaderColor(tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})

	table.SetColumnColor(tablewriter.Colors{tablewriter.FgHiBlueColor},
		tablewriter.Colors{tablewriter.FgBlueColor},
		tablewriter.Colors{tablewriter.FgBlueColor},
		tablewriter.Colors{tablewriter.FgBlueColor},
		tablewriter.Colors{tablewriter.FgBlueColor},
		tablewriter.Colors{tablewriter.FgBlueColor})

	// fill table
	tdata = append(tdata, []string{config.Name, config.Exec, config.Platform, config.Author, config.Date, config.Comment})
	table.AppendBulk(tdata)
	table.Render()
	out := tableString.String()
	CliPrintInfo("Module details:\n%s", out)
}

// scan custom modules in ModuleDir,
// and update ModuleHelpers, ModuleDocs
func InitModules() {
	dirs, err := ioutil.ReadDir(ModuleDir)
	if err != nil {
		CliPrintError("Failed to scan custom modules: %v", err)
		return
	}

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		config_file := ModuleDir + dir.Name() + "/config.json"
		config, err := readModCondig(config_file)
		if err != nil {
			CliPrintWarning("Reading config from %s: %v", dir.Name(), err)
			continue
		}

		ModuleHelpers[config.Name] = moduleCustom
		emp3r0r_data.ModuleComments[config.Name] = config.Comment

		err = updateModuleHelp(config)
		if err != nil {
			CliPrintWarning("Loading config from %s: %v", config.Name, err)
			continue
		}
		ModuleConfigs[config.Name] = *config
		CliPrintInfo("Loaded module %s", strconv.Quote(config.Name))
	}
	CliPrintInfo("Loaded %d modules", len(ModuleHelpers))
}

// readModCondig read config.json of a module
func readModCondig(file string) (pconfig *ModConfig, err error) {
	// read JSON
	jsonData, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("Read %s: %v", file, err)
	}

	// parse the json
	var config = ModConfig{}
	err = json.Unmarshal(jsonData, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON config: %v", err)
	}
	pconfig = &config
	return
}

// writeModCondig read config.json of a module
func writeModCondig(config *ModConfig, outfile string) (err error) {
	// parse the json
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON config: %v", err)
	}

	// write config.json
	return ioutil.WriteFile(outfile, data, 0600)
}

func updateModuleHelp(config *ModConfig) error {
	for opt, val_help := range config.Options {
		if len(val_help) < 2 {
			return fmt.Errorf("%s config error: %s incomplete", config.Name, opt)
		}
		emp3r0r_data.ModuleHelp[config.Name] = map[string]string{opt: val_help[1]}
	}
	return nil
}
