// plugin.go，用于自动获取所有插件的 Code 信息，并写入 plugins.json 文件，方便下次启动直接加载
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	pluginBaseUrl  = "https://plugins.jetbrains.com"
	pluginJsonFile = "plugins.json"
)

var (
	client = http.Client{
		Timeout: 60 * time.Second,
	}
	allPluginList []*Plugin
)

type ListPluginResponse struct {
	Plugins        []*Plugin `json:"plugins,omitempty"`
	Total          int       `json:"total,omitempty"`
	CorrectedQuery string    `json:"correctedQuery,omitempty"`
}

type Plugin struct {
	Id           int    `json:"id"`
	Code         string `json:"code,omitempty"`
	Name         string `json:"name"`
	PricingModel string `json:"pricingModel"`
	Icon         string `json:"icon"`
}

type PluginDetail struct {
	Id           int `json:"id"`
	PurchaseInfo struct {
		ProductCode   string      `json:"productCode"`
		BuyUrl        interface{} `json:"buyUrl"`
		PurchaseTerms interface{} `json:"purchaseTerms"`
		Optional      bool        `json:"optional"`
		TrialPeriod   int         `json:"trialPeriod"`
	} `json:"purchaseInfo"`
}

func init() {
	pluginFile, err := os.OpenFile(pluginJsonFile, os.O_RDONLY, 0644)
	if err == nil {
		err = json.NewDecoder(pluginFile).Decode(&allPluginList)
		if err != nil {
			panic(err)
		}
	}
	loadAllPlugin()
	savePlugin()
	return
}

func loadAllPlugin() {
	pluginIdCodeMap := make(map[int]string, len(allPluginList))
	for _, plugin := range allPluginList {
		pluginIdCodeMap[plugin.Id] = plugin.Code
	}

	pluginList, err := client.Get(pluginBaseUrl + "/api/searchPlugins?max=10000&offset=0")
	if err != nil {
		panic(err)
		return
	}
	defer pluginList.Body.Close()

	var listPluginResponse ListPluginResponse
	err = json.NewDecoder(pluginList.Body).Decode(&listPluginResponse)
	if err != nil {
		panic(err)
		return
	}

	for i, plugin := range listPluginResponse.Plugins {
		if plugin.PricingModel == "FREE" {
			continue
		}
		if pluginIdCodeMap[plugin.Id] != "" {
			continue
		}
		fmt.Println("found new plugin ", plugin.Name, plugin.PricingModel)
		listPluginResponse.Plugins[i].Icon = pluginBaseUrl + listPluginResponse.Plugins[i].Icon
		allPluginList = append(allPluginList, listPluginResponse.Plugins[i])
	}

	for _, plugin := range allPluginList {
		if plugin.Code == "" {
			plugin.Code = getCodeByPluginID(plugin.Id)
			fmt.Println("new plugin code ", plugin.Name, plugin.Code)
		}
	}
}

func getCodeByPluginID(id int) string {
	pluginDetailResp, err := client.Get(pluginBaseUrl + "/api/plugins/" + strconv.Itoa(id))
	if err != nil {
		panic(err)
	}
	defer pluginDetailResp.Body.Close()

	var pluginDetail PluginDetail
	err = json.NewDecoder(pluginDetailResp.Body).Decode(&pluginDetail)
	if err != nil {
		panic(err)
	}

	return pluginDetail.PurchaseInfo.ProductCode
}

func savePlugin() {
	f, err := os.Create(pluginJsonFile)
	if err != nil {
		panic(err)
	}
	err = json.NewEncoder(f).Encode(allPluginList)
	if err != nil {
		panic(err)
	}
}
