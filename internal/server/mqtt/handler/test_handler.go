package handler

import (
	"encoding/json"
	"log"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

// DeviceData 定义物联网设备上传的数据结构
type DeviceData struct {
	ID  string `json:"id"`
	Msg string `json:"msg"`
}

// CallbackFn 测试使用
func CallbackFn(cl *mqtt.Client, sub packets.Subscription, pk packets.Packet) {
	// 解析 JSON 数据
	var data DeviceData
	err := json.Unmarshal(pk.Payload, &data)
	if err != nil {
		log.Printf("解析 JSON 失败: %v, 原始数据: %s", err, string(pk.Payload))
		return
	}

	// 处理接收到的设备数据
	log.Printf("收到设备数据 - ID: %s, Msg: %s", data.ID, data.Msg)

	// 在这里可以添加您的业务逻辑处理
	// 例如：存储到数据库、转发到其他系统等
}
