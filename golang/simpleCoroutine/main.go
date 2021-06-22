package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/360EntSecGroup-Skylar/excelize"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
)

// 原始key 存在的文件
const INPUT_FILE  = "./长尾词数据.xlsx"
const API_URL     = "http://t.tao.mongodb.api.360che.com/foreign/long-tail-words?key="
const OUTPUT_FILE = "./fenci.csv"

type data struct {
	Key string `json:"key"`
	KeyName string `json:"keyName"`
	Nums int64 `json:"nums"`
}

type jsonData struct {
	Status int  `json:"status"`
	Data   data  `json:"data"`
}

var wg sync.WaitGroup
var mainWg sync.WaitGroup

// 限制并发的执行数量
const MAX_COROUTINE_NUM = 10

// 主方法
func main(){
	// 所有协程写入结果集到此通道
	returnCh := make(chan data , 10)
	// 使用通道限制只能启动指定数量的协成
	ch := make(chan int, MAX_COROUTINE_NUM)
	// 启动协程处理数据到 结果集通道
	mainWg.Add(1)
	go readFile(returnCh, ch)
	// 启动结果写入结果集通道
	mainWg.Add(1)
	go writeFile(returnCh)

	mainWg.Wait()
	fmt.Println("Success!!!")
}

// 写入结果集通道到文件
func writeFile(returnCh chan data){
	// 写入 csv文件
	file, err := os.OpenFile(OUTPUT_FILE, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("open file is failed, err: ", err)
	}
	defer file.Close()
	defer mainWg.Done()
	// 写入UTF-8 BOM，防止中文乱码
	w := csv.NewWriter(file)
	for da := range returnCh {
		w.Write([]string{da.Key, da.KeyName, strconv.FormatInt(da.Nums, 10)})
		w.Flush()
	}
}

// 读取文件 并启动指定数量的协程
func readFile(returnCh chan data, ch chan int){
	// 读取文件
	f, err := excelize.OpenFile(INPUT_FILE)
	if err != nil {
		fmt.Println(err)
		return
	}

	rows := f.GetRows("Sheet1")
	for k, row := range rows {
		wg.Add(1)
		ch<-1
		go execData(row[1], returnCh, ch)
		fmt.Printf("%d\r", k)
	}
	// 等待协程处理结束 关闭通道
	wg.Wait()

	defer close(returnCh)
	defer mainWg.Done()
}

// worker 协程, 调取接口并写入返回值到 结果集通道
func execData(key string, returnCh chan data, ch chan int){
	// 调取接口
	response, err := http.Get(API_URL+key)

	if err != nil {
		fmt.Println(err)
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
	}
	data := jsonData{}
	jsonErr := json.Unmarshal([]byte(string(body)), &data)
	if jsonErr != nil{
		fmt.Println(jsonErr)
	}
	if data.Status != 0{
		fmt.Println("接口返回失败")
		fmt.Println(string(body))
	}
	returnCh <- data.Data

	defer response.Body.Close()
	defer func() { <-ch }()
	defer wg.Done()
}
