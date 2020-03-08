package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"time"
)

const configPath = "./config.yml"
const defaultConfigPath = "./config.yml.example"
const readTimeoutRequest = 5 * time.Second
const writeTimeoutRequest = 10 * time.Second
const pathSeparator = "/"
const publicPath = "public" + pathSeparator
const templatePath = publicPath + "templates" + pathSeparator
const storagePath = "storage" + pathSeparator
const storageTmpFilePath = storagePath + "tmp" + pathSeparator
const kittenImgPath = publicPath + "img/kittens" + pathSeparator

const sessionUserUniKey = "SID"
const adminLogin = "admin"
const adminPassword = "adminPassword"
const authSalt = "anySalt"

type ConfigManager struct {
	current ConfigFile
	new     ConfigFile
}

func (config *ConfigManager) switchConfig() {
	config.current = config.new
	config.new = ConfigFile{}
}

type ConfigFile struct {
	Db struct {
		TypeDb string `yaml:"type",json:"type"`
		Socket string `yaml:"socket",json:"socket"`
	} `yaml:"db",json:"db"`
	Web struct {
		Ip   string `yaml:"ip",json:"ip"`
		Port string `yaml:"port",json:"port"`
	} `yaml:"web"`
	WebSocket struct {
		Ip   string `yaml:"ip",json:"ip"`
		Port string `yaml:"port",json:"port"`
	} `yaml:"web-socket",json:"web-socket"`
	Session struct {
		UniKey string `yaml:"uni-key",json:"uni-key"`
	} `yaml:"session",json:"session"`
}

func getConfigFromFile() (configFile ConfigFile) {
	var data []byte
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		data = readConfigFile(defaultConfigPath)

	} else {
		data = readConfigFile(configPath)
	}
	err := yaml.Unmarshal(data, &configFile)
	if err != nil {
		panic(err)
	}
	return
}

func readConfigFile(path string) []byte {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return data
}

func (config ConfigFile) getWebTcpSocket() string {
	return config.Web.Ip + ":" + config.Web.Port
}

func (config ConfigFile) getWebSocketTcpSocket() string {
	return config.WebSocket.Ip + ":" + config.WebSocket.Port
}

var Config = ConfigManager{current: getConfigFromFile()}

func main() {
	go runLoggerHandle()
	go runWebServerHandler()
	go runWebSocketHandler()
	go runKittenTaskHandler()
	webServerProcess.Chan <- signalUpServer
	webSocketServerProcess.Chan <- signalUpServer
	select {}
}
